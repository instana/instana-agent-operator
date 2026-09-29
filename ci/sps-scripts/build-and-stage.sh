#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Stage: code-build
# 1. Reads BOM from COS to get frozen SHAs and version.
# 2. Clones operator + charts at those exact SHAs.
# 3. Runs Go unit tests.
# 4. Builds and pushes the multiarch operator image to dev ICR.
# 5. Builds instana-agent-operator.yaml (make controller-yaml).
# 6. Builds OLM bundle (make bundle) → olm-<version>.zip.
# 7. Packages helm chart (ci/build.sh with OPERATOR_YAML + CHART_VERSION).
# 8. Pushes all artifacts to COS under the timestamp-scoped path.
# 9. Updates and re-pushes BOM to both COS locations.
#
set -euo pipefail
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== build-and-stage.sh - start ====="

# ---------------------------------------------------------------------------
# Read pipeline environment
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
COS_STATE_PATH=$(get_env cos-state-path)

echo "COS_BUCKET=${COS_BUCKET}"
echo "COS_STATE_PATH=${COS_STATE_PATH}"

# ---------------------------------------------------------------------------
# Read BOM from COS
# ---------------------------------------------------------------------------
BOM_FILE=$(mktemp)
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "pipeline-state/latest-release-candidate/bom.json" \
  --output json "${BOM_FILE}"

VERSION=$(jq -r '.version'                 "${BOM_FILE}")
RELEASE_TAG=$(jq -r '."release-tag"'       "${BOM_FILE}")
OPERATOR_SHA=$(jq -r '.sources."operator-sha"' "${BOM_FILE}")
CHARTS_SHA=$(jq -r '.sources."charts-sha"'     "${BOM_FILE}")
SOURCE_BRANCH=$(jq -r '."source-branch"'   "${BOM_FILE}")

echo "version=${VERSION}  release-tag=${RELEASE_TAG}"
echo "operator-sha=${OPERATOR_SHA}  charts-sha=${CHARTS_SHA}"

# ---------------------------------------------------------------------------
# Install dependencies
# ---------------------------------------------------------------------------
dnf install -y python3-pyyaml zip
./installGolang.sh amd64 2>/dev/null || source "${WORKSPACE}/${APP_REPO_FOLDER}/installGolang.sh" amd64
export PATH=$PATH:/usr/local/go/bin

# ---------------------------------------------------------------------------
# Clone repos at frozen SHAs
# ---------------------------------------------------------------------------
GH_TOKEN=$(get_secret github-token)

GH_TOKEN_FILE=$(mktemp)
echo "${GH_TOKEN}" > "${GH_TOKEN_FILE}"
chmod 600 "${GH_TOKEN_FILE}"

set +u
# shellcheck source=/dev/null
. "${ONE_PIPELINE_PATH}"/git/clone_repo_v2 \
  --repository "https://github.ibm.com/instana/instana-agent-operator" \
  --token-path "${GH_TOKEN_FILE}" \
  --branch "${SOURCE_BRANCH}" \
  --depth 1 \
  --directory operator-src \
  --use-lfs false \
  --force-exit false
set -u

git -C "${WORKSPACE}/operator-src" checkout "${OPERATOR_SHA}"

set +u
# shellcheck source=/dev/null
. "${ONE_PIPELINE_PATH}"/git/clone_repo_v2 \
  --repository "https://github.ibm.com/instana/instana-agent-charts" \
  --token-path "${GH_TOKEN_FILE}" \
  --branch "${SOURCE_BRANCH}" \
  --depth 1 \
  --directory charts-src \
  --use-lfs false \
  --force-exit false
set -u

rm -f "${GH_TOKEN_FILE}"

git -C "${WORKSPACE}/charts-src" checkout "${CHARTS_SHA}"

# ---------------------------------------------------------------------------
# Go unit tests
# ---------------------------------------------------------------------------
echo "=== Running Go unit tests ==="
cd "${WORKSPACE}/operator-src"
make test

# ---------------------------------------------------------------------------
# Authenticate with ICR
# ---------------------------------------------------------------------------
ICR_REGISTRY_DOMAIN="icr.io"
echo "[INFO] Authenticating with ${ICR_REGISTRY_DOMAIN}..."
export DOCKER_API_VERSION="1.41"
docker login -u iamapikey --password-stdin "${ICR_REGISTRY_DOMAIN}" < /config/api-key

# ---------------------------------------------------------------------------
# Build and push multiarch operator image to dev ICR
# ---------------------------------------------------------------------------
OPERATOR_DEV_IMAGE="${ICR_REGISTRY_DOMAIN}/instana-agent-dev/instana-agent-operator:${OPERATOR_SHA}"

echo "=== Building multiarch operator image ==="
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

cd "${WORKSPACE}/operator-src"
docker buildx build \
  --platform linux/amd64,linux/arm64,linux/s390x,linux/ppc64le \
  --tag "${OPERATOR_DEV_IMAGE}" \
  --push \
  .

# Capture manifest digest
OPERATOR_MANIFEST_DIGEST=$(skopeo inspect --format "{{.Digest}}" "docker://${OPERATOR_DEV_IMAGE}")
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"

# ---------------------------------------------------------------------------
# Build instana-agent-operator.yaml (make controller-yaml)
# Runs BEFORE helm package so the YAML is available for ci/build.sh.
# ---------------------------------------------------------------------------
echo "=== Building instana-agent-operator.yaml ==="
OPERATOR_IMAGE_WITH_DIGEST="${ICR_REGISTRY_DOMAIN}/instana-agent-dev/instana-agent-operator@${OPERATOR_MANIFEST_DIGEST}"
BARE_VERSION="${VERSION}"   # no 'v' prefix

cd "${WORKSPACE}/operator-src"
mkdir -p target

STDERR_LOG=$(mktemp)
make --silent \
  IMG="${OPERATOR_IMAGE_WITH_DIGEST}" \
  VERSION="${BARE_VERSION}" \
  controller-yaml 2>"${STDERR_LOG}" > target/instana-agent-operator.yaml
if [[ -s "${STDERR_LOG}" ]]; then
  cat "${STDERR_LOG}"
fi
rm -f "${STDERR_LOG}"

OPERATOR_YAML_PATH="${WORKSPACE}/operator-src/target/instana-agent-operator.yaml"
echo "instana-agent-operator.yaml written to ${OPERATOR_YAML_PATH}"

# ---------------------------------------------------------------------------
# Build OLM bundle (make bundle) → olm-<version>.zip
# ---------------------------------------------------------------------------
echo "=== Building OLM bundle ==="

# Fetch previous released version tag for the 'replaces' field
GH_API_TOKEN=$(get_secret github-token)
PREV_VERSION=$(curl --silent --fail --show-error -L \
  -H "Authorization: Bearer ${GH_API_TOKEN}" \
  "https://api.github.com/repos/instana/instana-agent-operator/tags" | \
  jq 'map(select(.name | test("^v[0-9]+.[0-9]+.[0-9]+$"))) | .[0].name' | \
  sed 's/[^0-9]*\([0-9]\+\.[0-9]\+\.[0-9]\+\).*/\1/')

if [[ -z "${PREV_VERSION}" ]]; then
  echo "ERROR: Could not determine previous released version for OLM 'replaces' field"
  exit 1
fi
echo "OLM PREV_VERSION=${PREV_VERSION}"

# Get agent image digest
AGENT_IMG_DIGEST=$(skopeo inspect --format "{{.Digest}}" "docker://icr.io/instana/agent:latest")

mkdir -p "${WORKSPACE}/operator-src/bundle"
mkdir -p "${WORKSPACE}/operator-src/target"

cd "${WORKSPACE}/operator-src"
make \
  IMG="${OPERATOR_IMAGE_WITH_DIGEST}" \
  VERSION="${BARE_VERSION}" \
  PREV_VERSION="${PREV_VERSION}" \
  AGENT_IMG="icr.io/instana/agent@${AGENT_IMG_DIGEST}" \
  bundle

pushd bundle
zip -r "../target/olm-${BARE_VERSION}.zip" .
popd

OLM_ZIP_PATH="${WORKSPACE}/operator-src/target/olm-${BARE_VERSION}.zip"
echo "OLM bundle written to ${OLM_ZIP_PATH}"

# ---------------------------------------------------------------------------
# Build helm chart using charts-src/ci/build.sh with OPERATOR_YAML + CHART_VERSION
# ---------------------------------------------------------------------------
echo "=== Building helm chart ==="
cd "${WORKSPACE}/charts-src"

OPERATOR_YAML="${OPERATOR_YAML_PATH}" \
CHART_VERSION="${BARE_VERSION}" \
  bash ci/build.sh

HELM_TGZ=$(ls "${WORKSPACE}/charts-src/artefacts/instana-agent-${BARE_VERSION}.tgz" 2>/dev/null || \
           ls "${WORKSPACE}/charts-src/artefacts/instana-agent-*.tgz" | head -1)
echo "Helm chart built: ${HELM_TGZ}"

# ---------------------------------------------------------------------------
# Upload artifacts to COS
# ---------------------------------------------------------------------------
echo "=== Pushing artifacts to COS under ${COS_STATE_PATH} ==="

ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/helm/instana-agent-${BARE_VERSION}.tgz" \
  --body "${HELM_TGZ}"
echo "COS upload: cos://${COS_BUCKET}/${COS_STATE_PATH}/helm/instana-agent-${BARE_VERSION}.tgz"

ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/olm/olm-${BARE_VERSION}.zip" \
  --body "${OLM_ZIP_PATH}"
echo "COS upload: cos://${COS_BUCKET}/${COS_STATE_PATH}/olm/olm-${BARE_VERSION}.zip"

ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/olm/instana-agent-operator.yaml" \
  --body "${OPERATOR_YAML_PATH}"
echo "COS upload: cos://${COS_BUCKET}/${COS_STATE_PATH}/olm/instana-agent-operator.yaml"

# ---------------------------------------------------------------------------
# Update BOM with image and artifact entries, re-push to both COS locations
# ---------------------------------------------------------------------------
echo "=== Updating BOM ==="
UPDATED_BOM=$(jq \
  --arg op_manifest     "${OPERATOR_DEV_IMAGE}" \
  --arg op_digest       "${OPERATOR_MANIFEST_DIGEST}" \
  --arg helm_path       "${COS_STATE_PATH}/helm/instana-agent-${BARE_VERSION}.tgz" \
  --arg olm_path        "${COS_STATE_PATH}/olm/olm-${BARE_VERSION}.zip" \
  --arg op_yaml_path    "${COS_STATE_PATH}/olm/instana-agent-operator.yaml" \
  '.images["operator-manifest"]        = $op_manifest |
   .images["operator-manifest-digest"] = $op_digest |
   .artifacts["helm-chart"]            = $helm_path |
   .artifacts["olm-bundle"]            = $olm_path |
   .artifacts["operator-yaml"]         = $op_yaml_path' \
  "${BOM_FILE}")

UPDATED_BOM_FILE=$(mktemp)
echo "${UPDATED_BOM}" > "${UPDATED_BOM_FILE}"

echo "Updated BOM:"
echo "${UPDATED_BOM}" | jq .

ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/bom.json" \
  --body "${UPDATED_BOM_FILE}"
echo "COS upload: cos://${COS_BUCKET}/${COS_STATE_PATH}/bom.json"

ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "pipeline-state/latest-release-candidate/bom.json" \
  --body "${UPDATED_BOM_FILE}"
echo "COS upload: cos://${COS_BUCKET}/pipeline-state/latest-release-candidate/bom.json"

rm -f "${BOM_FILE}" "${UPDATED_BOM_FILE}"

# ---------------------------------------------------------------------------
# Trigger async E2E tests
# ---------------------------------------------------------------------------
echo "=== Triggering async E2E tests ==="
E2E_STAGES=(
  "e2e-operator-ocp-latest"
  "e2e-operator-gke-lowest"
  "e2e-helm-ocp-lowest"
  "e2e-helm-gke-latest"
)

for stage in "${E2E_STAGES[@]}"; do
  echo "Triggering: ${stage}"
  trigger-task "${stage}" || {
    echo "WARNING: Failed to trigger ${stage}, continuing..."
  }
done

echo "===== build-and-stage.sh - end ====="
