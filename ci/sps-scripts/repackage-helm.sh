#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Step: repackage-helm (inside publish-helm-and-release stage)
# Downloads the staged helm chart tarball from COS, unpacks it,
# substitutes dev ICR image references with the public image digest,
# and re-packages it ready for publication.
#
# Output: ${WORKSPACE}/helm-public/instana-agent-<version>.tgz
#
set -euo pipefail
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== repackage-helm.sh - start ====="

# ---------------------------------------------------------------------------
# Read BOM
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
RELEASE_STATE_PATH=$(get_env release-state-path "pipeline-state/latest-release-candidate/bom.json")

BOM_FILE=$(mktemp)
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${RELEASE_STATE_PATH}" \
  --output json "${BOM_FILE}"

VERSION=$(jq -r '.version'                             "${BOM_FILE}")
OPERATOR_MANIFEST_DIGEST=$(jq -r '.images."operator-manifest-digest"' "${BOM_FILE}")
HELM_CHART_COS=$(jq -r '.artifacts."helm-chart"'       "${BOM_FILE}")
rm -f "${BOM_FILE}"

echo "version=${VERSION}"
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"
echo "helm-chart COS key=${HELM_CHART_COS}"

# ---------------------------------------------------------------------------
# Download staged helm tarball from COS
# ---------------------------------------------------------------------------
STAGED_TGZ="${WORKSPACE}/instana-agent-${VERSION}-staged.tgz"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${HELM_CHART_COS}" \
  --output json "${STAGED_TGZ}"
echo "Downloaded ${STAGED_TGZ}"

# ---------------------------------------------------------------------------
# Unpack, patch image references, repackage
# ---------------------------------------------------------------------------
UNPACK_DIR="${WORKSPACE}/helm-unpack"
rm -rf "${UNPACK_DIR}"
mkdir -p "${UNPACK_DIR}"
tar -xzf "${STAGED_TGZ}" -C "${UNPACK_DIR}"

# The tarball root is the chart directory (instana-agent/)
CHART_DIR="${UNPACK_DIR}/instana-agent"

# Substitute: dev image tag → public image digest
# values.yaml: controllerManager.image.name and .tag are both set by ci/build.sh;
# replace the entire reference so it uses the public icr.io/instana path + digest.
PUBLIC_REPO="icr.io/instana/instana-agent-operator"
PUBLIC_REF="${PUBLIC_REPO}@${OPERATOR_MANIFEST_DIGEST}"

echo "Patching image reference in values.yaml: → ${PUBLIC_REF}"
yq eval -i \
  ".controllerManager.image.name = \"${PUBLIC_REPO}\" | \
   .controllerManager.image.tag  = \"\"" \
  "${CHART_DIR}/values.yaml"

# The operator deployment template uses the tag value; replace with digest reference
# Use sed since Helm templates contain {{ }} syntax that yq cannot handle
DEV_IMAGE_PATTERN="icr\.io/instana-agent-dev/instana-agent-operator:[^\"]*"
sed -i "s|${DEV_IMAGE_PATTERN}|${PUBLIC_REF}|g" \
  "${CHART_DIR}/templates/operator_deployment_instana-agent-controller-manager.yml"

echo "Image reference patched."

# Re-run helm package
HELM_OUT_DIR="${WORKSPACE}/helm-public"
mkdir -p "${HELM_OUT_DIR}"

APP_VERSION=$(yq eval '.appVersion' "${CHART_DIR}/Chart.yaml")

helm package "${CHART_DIR}/." \
  --version "${VERSION}" \
  --app-version "${APP_VERSION}" \
  --destination "${HELM_OUT_DIR}/"

helm lint "${HELM_OUT_DIR}/instana-agent-${VERSION}.tgz"
echo "Repackaged: ${HELM_OUT_DIR}/instana-agent-${VERSION}.tgz"

# Export for the publish step that runs in the same stage
set_env repackaged-helm-tgz "${HELM_OUT_DIR}/instana-agent-${VERSION}.tgz"

echo "===== repackage-helm.sh - end ====="
