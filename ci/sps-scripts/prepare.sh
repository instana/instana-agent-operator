#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Stage: prepare
# Computes the release version from the `release-line` pipeline property (format: MAJOR.MINOR,
# e.g. "2.2"), freezes operator + charts SHAs, writes the initial BOM to COS.
# Exports cos-state-path for all downstream stages via set_env / export_env.
#
set -euo pipefail
RELEASE_SCRIPTS=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${RELEASE_SCRIPTS}/release-common.sh"
echo "===== prepare.sh - start ====="

# ---------------------------------------------------------------------------
# Resolve pipeline properties
# ---------------------------------------------------------------------------
RELEASE_LINE=$(get_env release-line "")
RELEASE_BRANCH=$(get_env release-branch-operator "main")
CHARTS_BRANCH=$(get_env release-branch-charts "main")
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region "us-south")

if [[ -z "${RELEASE_LINE}" ]]; then
  echo "ERROR: pipeline property 'release-line' is not set (expected format: MAJOR.MINOR, e.g. '2.2')" >&2
  exit 1
fi
if ! [[ "${RELEASE_LINE}" =~ ^(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$ ]]; then
  echo "ERROR: 'release-line' must be in MAJOR.MINOR format (e.g. '2.2'), got: '${RELEASE_LINE}'" >&2
  exit 1
fi

echo "release-line=${RELEASE_LINE}"
echo "release-branch-operator=${RELEASE_BRANCH}"
echo "release-branch-charts=${CHARTS_BRANCH}"
echo "cos-bucket-name=${COS_BUCKET}"
echo "cos-region=${COS_REGION}"

# ---------------------------------------------------------------------------
# Install git (already present in pipeline-base-ubi; belt-and-suspenders)
# ---------------------------------------------------------------------------
command -v git >/dev/null 2>&1 || dnf install -y git

# ---------------------------------------------------------------------------
# Clone operator repo and record HEAD SHA
# ---------------------------------------------------------------------------
# operator is on github.com (public GH); charts is on github.ibm.com (GHE)
GH_TOKEN=$(get_secret github-token)
GHE_TOKEN=$(get_secret ghe-token)
OPERATOR_REPO="github.com/instana/instana-agent-operator"
CHARTS_REPO="github.ibm.com/instana/instana-agent-charts"

GH_TOKEN_FILE=$(mktemp)
echo "${GH_TOKEN}" > "${GH_TOKEN_FILE}"
chmod 600 "${GH_TOKEN_FILE}"

GHE_TOKEN_FILE=$(mktemp)
echo "${GHE_TOKEN}" > "${GHE_TOKEN_FILE}"
chmod 600 "${GHE_TOKEN_FILE}"

# github.com accepts the default GIT_AUTH_USER (x-oauth-basic); no override needed
echo "Cloning operator repo at branch ${RELEASE_BRANCH}..."
set +u
# shellcheck source=/dev/null
. "${ONE_PIPELINE_PATH}"/git/clone_repo_v2 \
  --repository "https://${OPERATOR_REPO}" \
  --token-path "${GH_TOKEN_FILE}" \
  --branch "${RELEASE_BRANCH}" \
  --depth 50 \
  --directory operator-src \
  --use-lfs false \
  --force-exit false
set -u

OPERATOR_SHA=$(git -C "${WORKSPACE}/operator-src" rev-parse HEAD)
echo "operator-sha=${OPERATOR_SHA}"

# ---------------------------------------------------------------------------
# Clone charts repo and record HEAD SHA
# ---------------------------------------------------------------------------
# github.ibm.com (GHE) requires GIT_AUTH_USER=x-access-token
echo "Cloning charts repo at branch ${CHARTS_BRANCH}..."
set +u
GIT_AUTH_USER="x-access-token"
# shellcheck source=/dev/null
. "${ONE_PIPELINE_PATH}"/git/clone_repo_v2 \
  --repository "https://${CHARTS_REPO}" \
  --token-path "${GHE_TOKEN_FILE}" \
  --branch "${CHARTS_BRANCH}" \
  --depth 1 \
  --directory charts-src \
  --use-lfs false \
  --force-exit false
unset GIT_AUTH_USER
set -u

rm -f "${GH_TOKEN_FILE}" "${GHE_TOKEN_FILE}"

CHARTS_SHA=$(git -C "${WORKSPACE}/charts-src" rev-parse HEAD)
echo "charts-sha=${CHARTS_SHA}"

# ---------------------------------------------------------------------------
# Compute next version and published OLM predecessor through authenticated APIs
# ---------------------------------------------------------------------------
# Tags use the 'v' prefix on the operator repo (e.g. v2.2.21).
# Algorithm:
#   1. List all tags matching v<MAJOR>.<MINOR>.<patch> for the given release-line.
#   2. If none found → use <MAJOR>.<MINOR>.0 (first release on this line).
#   3. If found → take the highest patch via sort -V and increment by 1.
#   The 2.3.x tags are ignored when computing the next 2.2.x version.
#
export GH_TOKEN GHE_TOKEN
VERSIONS=$(python3 "${RELEASE_SCRIPTS}/release_api.py" versions "${RELEASE_LINE}")
echo "API version discovery: ${VERSIONS}"
VERSION=$(jq -er '.version' <<< "${VERSIONS}")
PREV_VERSION=$(jq -r '."previous-version"' <<< "${VERSIONS}")
if [[ -z "${PREV_VERSION}" ]]; then
  echo "ERROR: no published OLM predecessor found; define the initial catalog upgrade policy first" >&2
  exit 1
fi
RELEASE_TAG="v${VERSION}"
echo "Computed version: ${VERSION}  (operator tag: ${RELEASE_TAG})"

# ---------------------------------------------------------------------------
# Build the timestamp-scoped COS state path
# ---------------------------------------------------------------------------
# The run URL carries the real SPS UUID; fall back to a generated UUID, not the
# generic PIPELINE_RUN_NAME (often simply 'simple-pipelinerun').
PIPELINE_RUN_ID=$(printf '%s' "${PIPELINE_RUN_URL:-}" | sed -nE 's|.*/runs/([a-f0-9-]{36}).*|\1|p')
PIPELINE_RUN_ID=${PIPELINE_RUN_ID:-$(cat /proc/sys/kernel/random/uuid)}
BUILD_TIMESTAMP=$(date -u '+%Y%m%d%H%M%S')
CANDIDATE_ID="${BUILD_TIMESTAMP}-${PIPELINE_RUN_ID}"
COS_STATE_PATH="pipeline-state/release-${VERSION}/${CANDIDATE_ID}"
echo "COS state path: ${COS_STATE_PATH}"
echo "PROMOTION INPUT: release-state-path=${COS_STATE_PATH}/bom.json"

# Export for downstream stages
set_env cos-state-path "${COS_STATE_PATH}"
export_env cos-state-path

# ---------------------------------------------------------------------------
# Write initial BOM
# ---------------------------------------------------------------------------
BOM=$(jq -n \
  --arg candidate_id "${CANDIDATE_ID}" \
  --arg previous_version "${PREV_VERSION}" \
  --arg run_url "${PIPELINE_RUN_URL:-}" \
  --arg version       "${VERSION}" \
  --arg release_tag   "${RELEASE_TAG}" \
  --arg source_branch "${RELEASE_BRANCH}" \
  --arg charts_branch "${CHARTS_BRANCH}" \
  --arg cos_state     "${COS_STATE_PATH}" \
  --arg run_id        "${PIPELINE_RUN_ID}" \
  --arg op_sha        "${OPERATOR_SHA}" \
  --arg ch_sha        "${CHARTS_SHA}" \
  '{
    "candidate-id":     $candidate_id,
    "previous-version": $previous_version,
    "pipeline-run-url": $run_url,
    "version":          $version,
    "release-tag":      $release_tag,
    "source-branch":    $source_branch,
    "charts-branch":    $charts_branch,
    "cos-state-path":   $cos_state,
    "pipeline-run-id":  $run_id,
    "sources": {
      "operator-sha": $op_sha,
      "charts-sha":   $ch_sha
    },
    "images":    {},
    "artifacts": {}
  }')

echo "Initial BOM:"
echo "${BOM}" | jq .

BOM_FILE=$(mktemp)
echo "${BOM}" > "${BOM_FILE}"

# ---------------------------------------------------------------------------
# Log in to IBM Cloud and configure COS plugin
# ---------------------------------------------------------------------------
IBMCLOUD_API_KEY=$(get_secret ibmcloud-api-key)
ibmcloud login --apikey "${IBMCLOUD_API_KEY}" --no-region
unset IBMCLOUD_API_KEY
# ---------------------------------------------------------------------------
# Push initial scoped BOM (no mutable latest pointer)
# ---------------------------------------------------------------------------
echo "Uploading BOM to COS (timestamp-scoped)..."
ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/bom.json" \
  --body "${BOM_FILE}" \
  --region "${COS_REGION}"
echo "COS upload: cos://${COS_BUCKET}/${COS_STATE_PATH}/bom.json"

rm -f "${BOM_FILE}"

echo "===== prepare.sh - end ====="
