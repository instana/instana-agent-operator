#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Explicit candidate selection; passing any attempt is sufficient for each scenario.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
trap release_check_exit EXIT
release_check_start candidate-configuration 'Require an explicit immutable BOM path and a boolean E2E bypass setting'
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region us-south)
RELEASE_STATE_PATH=$(get_env release-state-path "")
SKIP_E2E_CHECK=$(get_env skip-e2e-check false)
if [[ -z "${RELEASE_STATE_PATH}" || "${RELEASE_STATE_PATH}" == *latest* ]]; then
  echo 'ERROR: release-state-path must name the exact candidate bom.json printed by release-build' >&2
  exit 1
fi
if [[ "${SKIP_E2E_CHECK}" != true && "${SKIP_E2E_CHECK}" != false ]]; then
  echo 'ERROR: skip-e2e-check must be true or false' >&2
  exit 1
fi
release_check_pass
release_check_start cos-login 'Authenticate to IBM Cloud for candidate and evidence downloads'
cos_login
release_check_pass
BOM_FILE="${WORKSPACE}/selected-bom.json"
release_check_start candidate-download "Download immutable candidate cos://${COS_BUCKET}/${RELEASE_STATE_PATH}"
cos_get "${RELEASE_STATE_PATH}" "${BOM_FILE}"
release_check_pass
release_check_start candidate-bom 'Validate BOM schema, candidate identity, artifact paths and allowed source branches'
python3 "${SCRIPT_DIR}/release_state.py" bom "${BOM_FILE}" "${RELEASE_STATE_PATH}"
release_check_pass
COS_STATE_PATH=$(jq -er '."cos-state-path"' "${BOM_FILE}")
set_env cos-state-path "${COS_STATE_PATH}"
export_env cos-state-path
echo "Selected immutable candidate: ${RELEASE_STATE_PATH}"
echo "BOM SHA256: $(sha256sum "${BOM_FILE}" | cut -d' ' -f1)"
jq '{version, "candidate-id", "source-branch", "charts-branch", sources, images}' "${BOM_FILE}"
for key in helm-chart olm-bundle operator-yaml; do
  release_check_start "artifact-${key}" "Download ${key} and verify its SHA256 against the selected BOM"
  cos_get "$(jq -er --arg k "${key}" '.artifacts[$k]' "${BOM_FILE}")" "${WORKSPACE}/validate-${key}"
  verify_sha256 "${WORKSPACE}/validate-${key}" "$(jq -er --arg k "${key}" '.checksums[$k]' "${BOM_FILE}")"
  release_check_pass
done
if [[ "${SKIP_E2E_CHECK}" == true ]]; then
  echo 'WARNING: FORCED PROMOTION: all E2E evidence checks bypassed; other safety gates remain enabled'
  release_check_start e2e-bypass 'All E2E evidence checks explicitly bypassed by skip-e2e-check=true; NOT validated'
  release_check_result skipped
  RELEASE_CHECK=''
else
  for cluster in ocp-fyre-latest gke-operator-lowest ocp-fyre-lowest gke-operator-latest; do
    result="${WORKSPACE}/success-${cluster}.json"
    evidence_key="${COS_STATE_PATH}/e2e/success/${cluster}.json"
    release_check_start "e2e-${cluster}-evidence" \
      "Require passing E2E evidence for ${cluster}: cos://${COS_BUCKET}/${evidence_key}"
    echo 'A missing success record means no passing evidence is available at this key: check E2E run status and dispatch.'
    cos_get "${evidence_key}" "${result}"
    release_check_pass
    release_check_start "e2e-${cluster}-identity" \
      "Validate ${cluster} success status, scenario/mode, candidate identity, sources, images and checksums"
    python3 "${SCRIPT_DIR}/release_state.py" success "${BOM_FILE}" "${result}" "${cluster}"
    release_check_pass
    release_check_start "e2e-${cluster}-log" "Download ${cluster} evidence log and verify its recorded SHA256"
    cos_get "$(jq -er '.log' "${result}")" "${WORKSPACE}/validate-${cluster}.log"
    verify_sha256 "${WORKSPACE}/validate-${cluster}.log" "$(jq -er '."log-sha256"' "${result}")"
    release_check_pass
  done
fi
# Readers in this one-container promotion always get the exact validated bytes.
set_env selected-bom-file "${BOM_FILE}"
echo 'PASSED: candidate validation, all three artifact checksums, and required E2E gates (unless explicitly bypassed).'
