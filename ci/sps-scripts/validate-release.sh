#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Explicit candidate selection; passing any attempt is sufficient for each scenario.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region us-south)
RELEASE_STATE_PATH=$(get_env release-state-path "")
SKIP_E2E_CHECK=$(get_env skip-e2e-check false)
if [[ -z "${RELEASE_STATE_PATH}" || "${RELEASE_STATE_PATH}" == *latest* ]]; then
  echo 'ERROR: release-state-path must name the exact candidate bom.json printed by release-build' >&2
  exit 1
fi
if [[ "${SKIP_E2E_CHECK}" != true && "${SKIP_E2E_CHECK}" != false ]]; then exit 1; fi
cos_login
BOM_FILE="${WORKSPACE}/selected-bom.json"
cos_get "${RELEASE_STATE_PATH}" "${BOM_FILE}"
if [[ "${1:-}" == --validation-only ]]; then
  python3 "${SCRIPT_DIR}/release_state.py" validation-bom "${BOM_FILE}" "${RELEASE_STATE_PATH}" \
    "$(get_env allowed-release-branch-operator main)" "$(get_env allowed-release-branch-chart main)"
else
  # Production promotion never reads validation branch override properties.
  python3 "${SCRIPT_DIR}/release_state.py" bom "${BOM_FILE}" "${RELEASE_STATE_PATH}"
fi
COS_STATE_PATH=$(jq -er '."cos-state-path"' "${BOM_FILE}")
set_env cos-state-path "${COS_STATE_PATH}"
export_env cos-state-path
echo "Selected immutable candidate: ${RELEASE_STATE_PATH}"
echo "BOM SHA256: $(sha256sum "${BOM_FILE}" | cut -d' ' -f1)"
for key in helm-chart olm-bundle operator-yaml; do
  cos_get "$(jq -er --arg k "${key}" '.artifacts[$k]' "${BOM_FILE}")" "${WORKSPACE}/validate-${key}"
  verify_sha256 "${WORKSPACE}/validate-${key}" "$(jq -er --arg k "${key}" '.checksums[$k]' "${BOM_FILE}")"
done
if [[ "${SKIP_E2E_CHECK}" == true ]]; then
  echo 'WARNING: FORCED PROMOTION: all E2E evidence checks bypassed; other safety gates remain enabled'
else
  for cluster in ocp-fyre-latest gke-operator-lowest ocp-fyre-lowest gke-operator-latest; do
    result="${WORKSPACE}/success-${cluster}.json"
    cos_get "${COS_STATE_PATH}/e2e/success/${cluster}.json" "${result}"
    python3 "${SCRIPT_DIR}/release_state.py" success "${BOM_FILE}" "${result}" "${cluster}"
    cos_get "$(jq -er '.log' "${result}")" "${WORKSPACE}/validate-${cluster}.log"
    verify_sha256 "${WORKSPACE}/validate-${cluster}.log" "$(jq -er '."log-sha256"' "${result}")"
  done
fi
# Readers in this one-container promotion always get the exact validated bytes.
set_env selected-bom-file "${BOM_FILE}"
