#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Separate from run-promotion.sh deliberately: no publication code is invoked.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
export SELECTED_BOM_FILE="${WORKSPACE}/selected-bom.json"
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region us-south)
RELEASE_STATE_PATH=$(get_env release-state-path "")
SKIP_E2E_CHECK=$(get_env skip-e2e-check false)
UPLOAD_REPORT=$(get_env validation-upload-report false)
VALIDATION_ID="$(date -u '+%Y%m%d%H%M%S')-$(cat /proc/sys/kernel/random/uuid)"
STARTED_AT=$(date -u '+%Y-%m-%dT%H:%M:%SZ')
REPORT_FILE="${WORKSPACE}/promotion-validation-report.json"
REPORT_KEY="pipeline-validation/${VALIDATION_ID}/report.json"
CURRENT_STEP=configuration
COMPLETED_CHECKS=()

write_validation_report() {
  local rc=$? status=failed bom_sha='' public_sha='{}' checks
  trap - EXIT TERM INT
  set +e
  if [[ ${rc} -eq 0 && "${CURRENT_STEP}" == complete ]]; then
    status=passed
  elif [[ ${rc} -eq 0 ]]; then
    rc=1
  fi
  if [[ -f "${SELECTED_BOM_FILE}" ]]; then
    bom_sha=$(sha256sum "${SELECTED_BOM_FILE}" | cut -d' ' -f1)
  fi
  if [[ "${status}" == passed ]]; then
    public_sha=$(jq -n \
      --arg helm "$(sha256sum "${PUBLIC_HELM_TGZ}" | cut -d' ' -f1)" \
      --arg olm "$(sha256sum "${PUBLIC_OLM_ZIP}" | cut -d' ' -f1)" \
      --arg yaml "$(sha256sum "${PUBLIC_OPERATOR_YAML}" | cut -d' ' -f1)" \
      '{"helm-chart":$helm,"olm-bundle":$olm,"operator-yaml":$yaml}')
  fi
  checks=$(printf '%s\n' "${COMPLETED_CHECKS[@]}" | jq -Rsc 'split("\n") | map(select(length > 0))')
  if ! jq -n --arg status "${status}" --arg step "${CURRENT_STEP}" \
      --arg id "${VALIDATION_ID}" --arg bom "${RELEASE_STATE_PATH}" --arg hash "${bom_sha}" \
      --arg started "${STARTED_AT}" --arg finished "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
      --arg run "${PIPELINE_RUN_URL:-}" --arg bypass "${SKIP_E2E_CHECK}" \
      --argjson checks "${checks}" --argjson artifacts "${public_sha}" \
      '{mode:"validation-only", status:$status, step:$step, "validation-id":$id,
        "release-state-path":$bom, "bom-sha256":$hash, "started-at":$started,"finished-at":$finished,
        "pipeline-run-url":$run, "e2e-check-bypassed":($bypass == "true"), checks:$checks,
        "public-artifact-sha256":$artifacts, published:false}' > "${REPORT_FILE}"; then
    echo 'ERROR: could not write validation report' >&2
    exit 1
  fi
  echo "Validation-only report: ${REPORT_FILE}"
  cat "${REPORT_FILE}"
  if [[ "${UPLOAD_REPORT}" == true ]]; then
    if ! cos_put "${REPORT_KEY}" "${REPORT_FILE}"; then
      echo 'ERROR: validation report upload failed' >&2
      rc=1
    fi
  fi
  exit "${rc}"
}
trap write_validation_report EXIT
trap 'exit 143' TERM
trap 'exit 130' INT

echo 'VALIDATION ONLY: no registry pushes, tags, releases, catalog PRs, Helm publication, or Pyxis submission.'
if [[ "${UPLOAD_REPORT}" != true && "${UPLOAD_REPORT}" != false ]]; then
  echo 'ERROR: validation-upload-report must be true or false' >&2
  exit 1
fi
if [[ -z "${RELEASE_STATE_PATH}" || "${RELEASE_STATE_PATH}" == *latest* ]]; then
  echo 'ERROR: select an explicit release-state-path ending in bom.json' >&2
  exit 1
fi

CURRENT_STEP=tools
dnf install -y python3-pyyaml zip unzip
for tool in python3 jq yq helm skopeo ibmcloud; do command -v "${tool}" >/dev/null; done
COMPLETED_CHECKS+=(tools)

# This is the same candidate, branch, artifact and E2E gate as real promotion.
# It logs in explicitly, reads COS, and never writes candidate/evidence objects.
CURRENT_STEP=candidate-and-e2e
bash "${SCRIPT_DIR}/validate-release.sh"
COMPLETED_CHECKS+=(candidate-and-e2e)

CURRENT_STEP=source-image-digest
verify_tag "$(jq -er '.images["operator-manifest"]' "${SELECTED_BOM_FILE}")" \
  "$(jq -er '.images["operator-manifest-digest"]' "${SELECTED_BOM_FILE}")"
COMPLETED_CHECKS+=(source-image-digest)

# Read-only tokens suffice: never request the publication identity or credentials.
CURRENT_STEP=github-tags-and-version
export GH_TOKEN GHE_TOKEN
GH_TOKEN=$(get_secret github-token)
GHE_TOKEN=$(get_secret ghe-token)
python3 "${SCRIPT_DIR}/release_api.py" check-tags "${SELECTED_BOM_FILE}"
COMPLETED_CHECKS+=(github-tags-and-version)

CURRENT_STEP=repackage-public-artifacts
bash "${SCRIPT_DIR}/repackage-helm.sh"
VERSION=$(jq -er '.version' "${SELECTED_BOM_FILE}")
export PUBLIC_HELM_TGZ="${WORKSPACE}/helm-public/instana-agent-${VERSION}.tgz"
export PUBLIC_OLM_ZIP="${WORKSPACE}/operator-public/olm-${VERSION}.zip"
export PUBLIC_OPERATOR_YAML="${WORKSPACE}/operator-public/instana-agent-operator.yaml"
COMPLETED_CHECKS+=(repackage-public-artifacts)

CURRENT_STEP=public-artifact-smoke
bash "${SCRIPT_DIR}/smoke-public-artifacts.sh"
COMPLETED_CHECKS+=(public-artifact-smoke)
CURRENT_STEP=complete
echo 'Validation complete. Nothing has been published; this is not proof of publication permissions or certification.'
