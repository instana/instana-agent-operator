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
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== prepare.sh - start ====="

# ---------------------------------------------------------------------------
# Resolve pipeline properties
# ---------------------------------------------------------------------------
RELEASE_LINE=$(get_env release-line "")
RELEASE_BRANCH=$(get_env release-branch "main")
COS_BUCKET=$(get_env cos-bucket-name)

if [[ -z "${RELEASE_LINE}" ]]; then
  echo "ERROR: pipeline property 'release-line' is not set (expected format: MAJOR.MINOR, e.g. '2.2')" >&2
  exit 1
fi
if ! [[ "${RELEASE_LINE}" =~ ^[0-9]+\.[0-9]+$ ]]; then
  echo "ERROR: 'release-line' must be in MAJOR.MINOR format (e.g. '2.2'), got: '${RELEASE_LINE}'" >&2
  exit 1
fi

echo "release-line=${RELEASE_LINE}"
echo "release-branch=${RELEASE_BRANCH}"
echo "cos-bucket-name=${COS_BUCKET}"

# ---------------------------------------------------------------------------
# Install git (already present in pipeline-base-ubi; belt-and-suspenders)
# ---------------------------------------------------------------------------
command -v git >/dev/null 2>&1 || dnf install -y git

# ---------------------------------------------------------------------------
# Clone operator repo and record HEAD SHA
# ---------------------------------------------------------------------------
GH_TOKEN=$(get_secret github-token)
OPERATOR_REPO="github.ibm.com/instana/instana-agent-operator"
CHARTS_REPO="github.ibm.com/instana/instana-agent-charts"

echo "Cloning operator repo at branch ${RELEASE_BRANCH}..."
clone_repo_v2 \
  --repo "https://${OPERATOR_REPO}" \
  --token-path <(echo "${GH_TOKEN}") \
  --branch "${RELEASE_BRANCH}" \
  --depth 50 \
  --directory operator-src

OPERATOR_SHA=$(git -C operator-src rev-parse HEAD)
echo "operator-sha=${OPERATOR_SHA}"

# ---------------------------------------------------------------------------
# Clone charts repo and record HEAD SHA
# ---------------------------------------------------------------------------
echo "Cloning charts repo at branch ${RELEASE_BRANCH}..."
clone_repo_v2 \
  --repo "https://${CHARTS_REPO}" \
  --token-path <(echo "${GH_TOKEN}") \
  --branch "${RELEASE_BRANCH}" \
  --depth 1 \
  --directory charts-src

CHARTS_SHA=$(git -C charts-src rev-parse HEAD)
echo "charts-sha=${CHARTS_SHA}"

# ---------------------------------------------------------------------------
# Compute next version from release-line + existing tags on the operator repo
# ---------------------------------------------------------------------------
# Tags use the 'v' prefix on the operator repo (e.g. v2.2.21).
# Algorithm:
#   1. List all tags matching v<MAJOR>.<MINOR>.<patch> for the given release-line.
#   2. If none found → use <MAJOR>.<MINOR>.0 (first release on this line).
#   3. If found → take the highest patch via sort -V and increment by 1.
#   The 2.3.x tags are ignored when computing the next 2.2.x version.

REL_MAJOR=$(echo "${RELEASE_LINE}" | cut -d. -f1)
REL_MINOR=$(echo "${RELEASE_LINE}" | cut -d. -f2)

# All tags for this exact major.minor (strips leading 'v', filters to MAJOR.MINOR.*)
MATCHING_PATCHES=$(git -C operator-src tag --list "v${REL_MAJOR}.${REL_MINOR}.*" \
  | sed 's/^v//' \
  | sort -V)

echo "Existing tags for ${RELEASE_LINE}.x: ${MATCHING_PATCHES:-<none>}"

if [[ -z "${MATCHING_PATCHES}" ]]; then
  # No release on this line yet — start at .0
  NEXT_PATCH=0
else
  LATEST_PATCH=$(echo "${MATCHING_PATCHES}" | tail -1 | cut -d. -f3)
  NEXT_PATCH=$((LATEST_PATCH + 1))
fi

VERSION="${REL_MAJOR}.${REL_MINOR}.${NEXT_PATCH}"
RELEASE_TAG="v${VERSION}"
echo "Computed version: ${VERSION}  (operator tag: ${RELEASE_TAG})"

# ---------------------------------------------------------------------------
# Build the timestamp-scoped COS state path
# ---------------------------------------------------------------------------
BUILD_TIMESTAMP=$(date -u '+%Y%m%d%H%M')
COS_STATE_PATH="pipeline-state/release-${VERSION}/${BUILD_TIMESTAMP}"
echo "COS state path: ${COS_STATE_PATH}"

# Export for downstream stages
set_env cos-state-path "${COS_STATE_PATH}"
export_env cos-state-path

# ---------------------------------------------------------------------------
# Write initial BOM
# ---------------------------------------------------------------------------
PIPELINE_RUN_ID="${PIPELINE_RUN_NAME:-unknown}"

BOM=$(jq -n \
  --arg version       "${VERSION}" \
  --arg release_tag   "${RELEASE_TAG}" \
  --arg source_branch "${RELEASE_BRANCH}" \
  --arg cos_state     "${COS_STATE_PATH}" \
  --arg run_id        "${PIPELINE_RUN_ID}" \
  --arg op_sha        "${OPERATOR_SHA}" \
  --arg ch_sha        "${CHARTS_SHA}" \
  '{
    "version":          $version,
    "release-tag":      $release_tag,
    "source-branch":    $source_branch,
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
# Push BOM to two COS locations
# ---------------------------------------------------------------------------
echo "Uploading BOM to COS (timestamp-scoped)..."
ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/bom.json" \
  --body "${BOM_FILE}"
echo "COS upload: cos://${COS_BUCKET}/${COS_STATE_PATH}/bom.json"

echo "Uploading BOM to COS (latest-release-candidate pointer)..."
ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "pipeline-state/latest-release-candidate/bom.json" \
  --body "${BOM_FILE}"
echo "COS upload: cos://${COS_BUCKET}/pipeline-state/latest-release-candidate/bom.json"

rm -f "${BOM_FILE}"

echo "===== prepare.sh - end ====="
