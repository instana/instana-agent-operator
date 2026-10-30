#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Step: repackage-helm (inside publish-helm-and-release stage)
# Downloads the staged helm chart tarball from COS, unpacks it,
# substitutes dev ICR image references with public references, and re-packages it.
# Also prepares public copies of the operator YAML and OLM bundle. Staged COS
# artifacts remain unchanged so release-build and E2E continue using dev images.
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
COS_REGION=$(get_env cos-region "us-south")
RELEASE_STATE_PATH="$(get_env cos-state-path)/bom.json"

BOM_FILE="${WORKSPACE}/bom-repackage-$$.json"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${RELEASE_STATE_PATH}" \
  --region "${COS_REGION}" \
  --output json "${BOM_FILE}"

VERSION=$(jq -r '.version'                             "${BOM_FILE}")
OPERATOR_MANIFEST_DIGEST=$(jq -r '.images."operator-manifest-digest"' "${BOM_FILE}")
HELM_CHART_COS=$(jq -r '.artifacts."helm-chart"'       "${BOM_FILE}")
OLM_BUNDLE_COS=$(jq -r '.artifacts."olm-bundle"'       "${BOM_FILE}")
OPERATOR_YAML_COS=$(jq -r '.artifacts."operator-yaml"' "${BOM_FILE}")
rm -f "${BOM_FILE}"

echo "version=${VERSION}"
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"
echo "helm-chart COS key=${HELM_CHART_COS}"

# ---------------------------------------------------------------------------
# Download staged helm tarball from COS
# ---------------------------------------------------------------------------
STAGED_TGZ="${WORKSPACE}/instana-agent-${VERSION}-staged.tgz"
rm -f "${STAGED_TGZ}"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${HELM_CHART_COS}" \
  --region "${COS_REGION}" \
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

# Substitute: dev image reference → public image tag
PUBLIC_REPO="icr.io/instana/instana-agent-operator"
PUBLIC_REF="${PUBLIC_REPO}:${VERSION}"

echo "Patching image reference in values.yaml: → ${PUBLIC_REF}"
yq eval -i \
  ".controllerManager.image.name = \"${PUBLIC_REPO}\" | \
   .controllerManager.image.tag  = \"${VERSION}\" | \
   del(.controllerManager.image.digest)" \
  "${CHART_DIR}/values.yaml"

# The operator deployment template uses the fallback image; replace with public version tag
DEV_IMAGE_PATTERN="icr\.io/instana-agent-dev/instana-agent-operator(@sha256:[a-f0-9]+|:[^\"]*)"
sed -E -i "s#${DEV_IMAGE_PATTERN}#${PUBLIC_REF}#g" \
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

# ---------------------------------------------------------------------------
# Prepare public operator YAML and OLM bundle without modifying staged artifacts
# ---------------------------------------------------------------------------
if ! command -v unzip >/dev/null || ! command -v zip >/dev/null; then
  dnf install -y zip unzip
fi

OPERATOR_OUT_DIR="${WORKSPACE}/operator-public"
mkdir -p "${OPERATOR_OUT_DIR}"
OPERATOR_YAML="${OPERATOR_OUT_DIR}/instana-agent-operator.yaml"
STAGED_OLM="${OPERATOR_OUT_DIR}/olm-staged.zip"
OLM_ZIP="${OPERATOR_OUT_DIR}/olm-${VERSION}.zip"
rm -f "${OPERATOR_YAML}" "${STAGED_OLM}" "${OLM_ZIP}"

ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${OPERATOR_YAML_COS}" \
  --region "${COS_REGION}" \
  --output json "${OPERATOR_YAML}"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${OLM_BUNDLE_COS}" \
  --region "${COS_REGION}" \
  --output json "${STAGED_OLM}"

OLM_UNPACK_DIR=$(mktemp -d "${OPERATOR_OUT_DIR}/olm-unpack-XXXXXX")
unzip -q "${STAGED_OLM}" -d "${OLM_UNPACK_DIR}"

# Retain the version tag or digest; only the registry namespace changes.
sed -i 's|icr.io/instana-agent-dev/instana-agent-operator|icr.io/instana/instana-agent-operator|g' \
  "${OPERATOR_YAML}"
find "${OLM_UNPACK_DIR}" -type f -exec \
  sed -i 's|icr.io/instana-agent-dev/instana-agent-operator|icr.io/instana/instana-agent-operator|g' {} +

(cd "${OLM_UNPACK_DIR}" && zip -qr "${OLM_ZIP}" .)
rm -rf "${OLM_UNPACK_DIR}"
rm -f "${STAGED_OLM}"

set_env repackaged-operator-yaml "${OPERATOR_YAML}"
set_env repackaged-olm-zip "${OLM_ZIP}"

echo "===== repackage-helm.sh - end ====="
