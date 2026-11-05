#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# One supported SPS task, explicit login, no reliance on previous run containers.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
export RELEASE_SCRIPTS="${SCRIPT_DIR}"
source "${SCRIPT_DIR}/release-common.sh"
trap release_check_exit EXIT
promotion_step() {
  local check="$1" detail="$2"
  shift 2
  release_check_start "${check}" "${detail}"
  "$@"
  release_check_pass
}
release_check_start tools 'Install dependencies and check required promotion tools'
dnf install -y python3-pyyaml zip unzip
for tool in jq yq helm skopeo preflight gcloud gsutil; do
  command -v "${tool}" >/dev/null || { echo "ERROR: required tool not found: ${tool}" >&2; exit 1; }
done
release_check_pass
promotion_step candidate-and-e2e 'Validate candidate, artifact checksums and E2E evidence' \
  bash "${SCRIPT_DIR}/validate-release.sh"
# Every reader uses this local validated snapshot, never re-resolves COS/latest.
export SELECTED_BOM_FILE="${WORKSPACE}/selected-bom.json"
export GH_TOKEN GHE_TOKEN
release_check_start github-credentials 'Load GitHub credentials for release checks and publication'
GH_TOKEN=$(get_secret github-token)
GHE_TOKEN=$(get_secret ghe-token)
release_check_pass
promotion_step github-tags-and-version 'Check release version and GitHub tags for conflicts' \
  python3 "${SCRIPT_DIR}/release_api.py" check-tags "${SELECTED_BOM_FILE}"
export PROMOTION_PHASE=redhat
promotion_step redhat-images 'Publish Red Hat images and verify digests' bash "${SCRIPT_DIR}/promote-images.sh"
promotion_step redhat-preflight 'Run Red Hat preflight and certification submission' bash "${SCRIPT_DIR}/preflight-release.sh"
promotion_step repackage-public-artifacts 'Repackage public Helm, OLM and operator YAML locally' \
  bash "${SCRIPT_DIR}/repackage-helm.sh"
release_check_start public-artifact-paths 'Resolve local repackaged artifact paths'
# Pass workspace files explicitly between child shells; do not depend on a
# shell-local implementation of set_env for this handoff.
VERSION=$(jq -er '.version' "${SELECTED_BOM_FILE}")
export PUBLIC_HELM_TGZ="${WORKSPACE}/helm-public/instana-agent-${VERSION}.tgz"
export PUBLIC_OLM_ZIP="${WORKSPACE}/operator-public/olm-${VERSION}.zip"
export PUBLIC_OPERATOR_YAML="${WORKSPACE}/operator-public/instana-agent-operator.yaml"
release_check_pass
promotion_step public-artifact-smoke 'Smoke-test repackaged public artifacts' bash "${SCRIPT_DIR}/smoke-public-artifacts.sh"
export PROMOTION_PHASE=versioned
promotion_step versioned-images 'Publish versioned images and verify digests' bash "${SCRIPT_DIR}/promote-images.sh"
promotion_step publish-helm 'Publish public Helm artifacts' bash "${SCRIPT_DIR}/publish-helm.sh"
promotion_step publish-olm 'Publish OLM releases and catalog pull requests' bash "${SCRIPT_DIR}/publish-olm.sh"
export PROMOTION_PHASE=latest
promotion_step latest-images 'Update latest image tags and verify digests' bash "${SCRIPT_DIR}/promote-images.sh"
release_check_start promotion-receipt 'Write and upload the publication receipt'
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region us-south)
STATE=$(jq -er '."cos-state-path"' "${SELECTED_BOM_FILE}")
jq -n --slurpfile bom "${SELECTED_BOM_FILE}" --slurpfile prs "${WORKSPACE}/olm-prs.json" \
  --arg run "${PIPELINE_RUN_URL:-}" \
  --arg bypass "$(get_env skip-e2e-check false)" \
  '{candidate:$bom[0], "olm-prs":$prs[0], "promotion-run":$run, "skip-e2e-check":$bypass, status:"published"}' \
  > "${WORKSPACE}/promotion-receipt.json"
cos_put "${STATE}/promotion/receipt.json" "${WORKSPACE}/promotion-receipt.json"
release_check_pass
echo 'Promotion complete: all publication stages passed and the promotion receipt was uploaded.'
