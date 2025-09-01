#!/bin/sh
#
# (c) Copyright IBM Corp. 2025
#
set -e

# Load environment variables
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
ARTIFACTORY_INTERNAL_USERNAME=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".username")
ARTIFACTORY_INTERNAL_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")
GIT_COMMIT=$(load_repo app-repo commit)

echo "Building commit ${GIT_COMMIT}"

# Authenticate with the private registry
echo "[INFO] Authenticating with the private Docker registry..."
echo "${ARTIFACTORY_INTERNAL_PASSWORD}" | docker login delivery.instana.io --username "${ARTIFACTORY_INTERNAL_USERNAME}" --password-stdin

echo "[INFO] Setting up Docker Buildx for multi-platform builds..."
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

echo "pwd: $(pwd)"

BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/"

# Define registry and image names
REGISTRY="delivery.instana.io"
REPO_PATH="int-docker-agent-local/instana-agent-operator/dev-build"
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

git branch
