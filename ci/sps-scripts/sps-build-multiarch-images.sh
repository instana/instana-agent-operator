#!/bin/sh
#
# (c) Copyright IBM Corp. 2025
#
set -e

# Load environment variables
ICR_REGISTRY_DOMAIN="icr.io"
ARTIFACTORY_REGISTRY_DOMAIN="delivery.instana.io"
GIT_COMMIT=$(load_repo app-repo commit)
BRANCH_NAME=$(load_repo app-repo branch)
ARTIFACTORY_INTERNAL_USERNAME=$(get_env ARTIFACTORY_INTERNAL_USERNAME)
ARTIFACTORY_INTERNAL_PASSWORD=$(get_env ARTIFACTORY_INTERNAL_PASSWORD)

echo "Building commit ${GIT_COMMIT} on branch ${BRANCH_NAME}"

# Authenticate with the private registry, see https://cloud.ibm.com/docs/devsecops?topic=devsecops-cd-devsecops-build-docker#cd-devsecops-work-with-icr
echo "[INFO] Authenticating with the $ICR_REGISTRY_DOMAIN private Docker registry..."
docker login -u iamapikey --password-stdin "$ICR_REGISTRY_DOMAIN" < /config/api-key

echo "[INFO] Authenticating with artifactory registry..."
echo "$ARTIFACTORY_INTERNAL_PASSWORD" | docker login "${ARTIFACTORY_REGISTRY_DOMAIN}" --username "${ARTIFACTORY_INTERNAL_USERNAME}" --password-stdin

echo "[INFO] Setting up Docker Buildx for multi-platform builds..."
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

echo "pwd: $(pwd)"

BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/"

# Define registry and image names
REGISTRY_IMAGE_TAG_ICR="${ICR_REGISTRY_DOMAIN}/instana-agent-dev/instana-agent-operator:${GIT_COMMIT}"
REGISTRY_IMAGE_TAG_ARTIFACTORY="${ARTIFACTORY_REGISTRY_DOMAIN}/int-docker-agent-local/instana-agent-operator/dev-build:${GIT_COMMIT}"

echo "cd into build context"
cd "${BUILD_CONTEXT}"
echo "pwd: $(pwd)"

# Determine build platforms based on branch
if [ "$BRANCH_NAME" = "main" ]; then
    echo "[INFO] Building multi-architecture image for main branch..."
    PLATFORMS="linux/amd64,linux/arm64,linux/s390x,linux/ppc64le"
else
    echo "[INFO] Building amd64-only image for PR branch..."
    PLATFORMS="linux/amd64"
fi

echo "[INFO] Using platforms: ${PLATFORMS}"

docker buildx build \
    --platform ${PLATFORMS} \
    -t "${REGISTRY_IMAGE_TAG_ICR}" \
    -t "${REGISTRY_IMAGE_TAG_ARTIFACTORY}" \
    --push \
    "$BUILD_CONTEXT"

# mark for scanning
pipelinectl save_artifact operator_image "name=${REGISTRY_IMAGE_TAG_ICR}" "type=image"
