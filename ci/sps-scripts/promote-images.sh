#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Stage: promote-images (release-promotion pipeline)
# Uses the frozen manifest digest from the BOM to copy the dev ICR image
# to all public registries via skopeo.
# Only runs for production versions (format: X.Y.Z, no pre-release suffix).
#
set -euo pipefail
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== promote-images.sh - start ====="

# ---------------------------------------------------------------------------
# Read BOM
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region "us-south")
RELEASE_STATE_PATH="$(get_env cos-state-path)/bom.json"

BOM_FILE="${WORKSPACE}/bom-promote-$$.json"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${RELEASE_STATE_PATH}" \
  --region "${COS_REGION}" \
  --output json "${BOM_FILE}"

VERSION=$(jq -r '.version'                             "${BOM_FILE}")
OPERATOR_MANIFEST_DIGEST=$(jq -r '.images."operator-manifest-digest"' "${BOM_FILE}")
SOURCE_BRANCH=$(jq -r '."source-branch"'               "${BOM_FILE}")
rm -f "${BOM_FILE}"

echo "version=${VERSION}"
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"
echo "source-branch=${SOURCE_BRANCH}"

# ---------------------------------------------------------------------------
# Guard: only promote clean release versions (no pre-release suffix)
# ---------------------------------------------------------------------------
if ! [[ "${VERSION}" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "ERROR: version '${VERSION}' does not match ^[0-9]+\\.[0-9]+\\.[0-9]+$ — refusing to promote a non-production build"
  exit 1
fi

# ---------------------------------------------------------------------------
# Authenticate registries
# ---------------------------------------------------------------------------
ICR_API_KEY=$(cat /config/api-key)
QUAY_CREDENTIALS=$(get_secret quay-redhat-isv-push-credentials)
QUAY_USERNAME=$(echo "${QUAY_CREDENTIALS}" | jq -r ".username")
QUAY_PASSWORD=$(echo "${QUAY_CREDENTIALS}" | jq -r ".password")
ICRREL_API_KEY=$(get_secret icr-rel-api-key)

# Source image (frozen digest prevents race conditions)
SRC_IMAGE="icr.io/instana-agent-dev/instana-agent-operator@${OPERATOR_MANIFEST_DIGEST}"
echo "Source: ${SRC_IMAGE}"

skopeo_copy() {
  local dst="$1"
  local dst_creds="$2"
  echo "Promoting → ${dst}"
  skopeo copy --all \
    --src-creds "iamapikey:${ICR_API_KEY}" \
    --dest-creds "${dst_creds}" \
    "docker://${SRC_IMAGE}" \
    "${dst}"
}

# ---------------------------------------------------------------------------
# 1. public ICR — versioned tag (strip 'v' prefix)
# ---------------------------------------------------------------------------
skopeo_copy \
  "docker://icr.io/instana/instana-agent-operator:${VERSION}" \
  "iamapikey:${ICR_API_KEY}"

# ---------------------------------------------------------------------------
# 2. public ICR — :latest (only when releasing from main)
# ---------------------------------------------------------------------------
if [[ "${SOURCE_BRANCH}" == "main" ]]; then
  skopeo_copy \
    "docker://icr.io/instana/instana-agent-operator:latest" \
    "iamapikey:${ICR_API_KEY}"
fi

# ---------------------------------------------------------------------------
# 3. icr.io/instana-rel (internal release registry)
# ---------------------------------------------------------------------------
skopeo_copy \
  "docker://icr.io/instana-rel/rel-docker-agent-local/instana-agent-operator:${VERSION}" \
  "iamapikey:${ICRREL_API_KEY}"

# ---------------------------------------------------------------------------
# 4. quay.io (Red Hat ISV)
# ---------------------------------------------------------------------------
skopeo_copy \
  "docker://quay.io/redhat-isv-containers/5e961c2c93604e02afa9ebdf:${VERSION}" \
  "${QUAY_USERNAME}:${QUAY_PASSWORD}"

echo "===== promote-images.sh - end ====="
