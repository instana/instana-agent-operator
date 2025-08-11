#!/bin/sh
set -e

# Load environment variables
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
export ARTIFACTORY_INTERNAL_USERNAME=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".username")
export ARTIFACTORY_INTERNAL_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")
export INSTANA_TWISTCLI_VERSION=$(get_env INSTANA_TWISTCLI_VERSION)
export GIT_COMMIT="$(get_env branch || echo "latest")"

# Authenticate with the private registry
echo "[INFO] Authenticating with the private Docker registry..."
echo "$ARTIFACTORY_INTERNAL_PASSWORD" | docker login delivery.instana.io --username $ARTIFACTORY_INTERNAL_USERNAME --password-stdin

BUILD_CONTEXT=$WORKSPACE/$APP_REPO_FOLDER/ci/images/e2e-base-image/
REGISTRY_PATH="delivery.instana.io/int-docker-agent-local/instana-agent-operator/e2e-base-image"
IMAGE_TAG="${GIT_COMMIT}"

# Build and publish the container image
echo "[INFO] Building and publishing the operator base image..."

docker buildx build \
    --push \
    --tag "${REGISTRY_PATH}:${IMAGE_TAG}" \
    "$BUILD_CONTEXT"

echo "[INFO] Operator base image built and published successfully to ${REGISTRY_PATH}:${IMAGE_TAG}"
