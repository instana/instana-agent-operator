#!/bin/sh
#
# (c) Copyright IBM Corp. 2025
#
set -e

# Load environment variables
export ICR_REGISTRY_DOMAIN="icr.io"
GIT_COMMIT=$(load_repo app-repo commit)

echo "Building commit ${GIT_COMMIT}"

# Authenticate with the private registry, see https://cloud.ibm.com/docs/devsecops?topic=devsecops-cd-devsecops-build-docker#cd-devsecops-work-with-icr
echo "[INFO] Authenticating with the $ICR_REGISTRY_DOMAIN private Docker registry..."
docker login -u iamapikey --password-stdin "$ICR_REGISTRY_DOMAIN" < /config/api-key

echo "[INFO] Setting up Docker Buildx for multi-platform builds..."
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

echo "pwd: $(pwd)"

BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/"

# Define registry and image names
REGISTRY="icr.io"
REPO_PATH="instana-agent-dev/instana-agent-operator"
FULL_REPO_PATH="${REGISTRY}/${REPO_PATH}"
REGISTRY_IMAGE_TAG="${FULL_REPO_PATH}:${GIT_COMMIT}"

echo "cd into build context"
cd "${BUILD_CONTEXT}"
echo "pwd: $(pwd)"

docker buildx build \
    --platform linux/amd64,linux/arm64,linux/s390x,linux/ppc64le \
    -t "${REGISTRY_IMAGE_TAG}" \
    --push \
    "$BUILD_CONTEXT"

pipelinectl save_artifact operator_image "name=${REGISTRY_IMAGE_TAG}" "type=image"