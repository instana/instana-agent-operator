#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# One supported SPS task, explicit login, no reliance on previous run containers.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
export RELEASE_SCRIPTS="${SCRIPT_DIR}"
dnf install -y python3-pyyaml zip unzip
for tool in jq yq helm skopeo preflight gcloud gsutil; do command -v "${tool}" >/dev/null; done
bash "${SCRIPT_DIR}/validate-release.sh"
# Every reader uses this local validated snapshot, never re-resolves COS/latest.
export SELECTED_BOM_FILE="${WORKSPACE}/selected-bom.json"
CREDS=$(get_secret github-instana-agent-build)
export GH_TOKEN GHE_TOKEN
GH_TOKEN=$(jq -er '.token' <<< "${CREDS}")
GHE_TOKEN=$(get_secret ghe-token)
python3 "${SCRIPT_DIR}/release_api.py" check-tags "${SELECTED_BOM_FILE}"
export PROMOTION_PHASE=redhat
bash "${SCRIPT_DIR}/promote-images.sh"
bash "${SCRIPT_DIR}/preflight-release.sh"
bash "${SCRIPT_DIR}/repackage-helm.sh"
# Pass workspace files explicitly between child shells; do not depend on a
# shell-local implementation of set_env for this handoff.
VERSION=$(jq -er '.version' "${SELECTED_BOM_FILE}")
export PUBLIC_HELM_TGZ="${WORKSPACE}/helm-public/instana-agent-${VERSION}.tgz"
export PUBLIC_OLM_ZIP="${WORKSPACE}/operator-public/olm-${VERSION}.zip"
export PUBLIC_OPERATOR_YAML="${WORKSPACE}/operator-public/instana-agent-operator.yaml"
bash "${SCRIPT_DIR}/smoke-public-artifacts.sh"
export PROMOTION_PHASE=versioned
bash "${SCRIPT_DIR}/promote-images.sh"
bash "${SCRIPT_DIR}/publish-helm.sh"
bash "${SCRIPT_DIR}/publish-olm.sh"
export PROMOTION_PHASE=latest
bash "${SCRIPT_DIR}/promote-images.sh"
source "${SCRIPT_DIR}/release-common.sh"
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region us-south)
STATE=$(jq -er '."cos-state-path"' "${SELECTED_BOM_FILE}")
jq -n --slurpfile bom "${SELECTED_BOM_FILE}" --slurpfile prs "${WORKSPACE}/olm-prs.json" \
  --arg run "${PIPELINE_RUN_URL:-}" \
  --arg bypass "$(get_env skip-e2e-check false)" \
  '{candidate:$bom[0], "olm-prs":$prs[0], "promotion-run":$run, "skip-e2e-check":$bypass, status:"published"}' \
  > "${WORKSPACE}/promotion-receipt.json"
cos_put "${STATE}/promotion/receipt.json" "${WORKSPACE}/promotion-receipt.json"
