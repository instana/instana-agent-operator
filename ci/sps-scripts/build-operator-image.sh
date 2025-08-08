#!/bin/sh
set -e

# Load environment variables
export ARTIFACTORY_INTERNAL_USERNAME=$(get_env ARTIFACTORY_INTERNAL_USERNAME)
export ARTIFACTORY_INTERNAL_PASSWORD=$(get_env ARTIFACTORY_INTERNAL_PASSWORD)
export QA_AGENT_DOWNLOAD_KEY=$(get_env QA_AGENT_DOWNLOAD_KEY)
export INSTANA_TWISTCLI_VERSION=$(get_env INSTANA_TWISTCLI_VERSION)

# Authenticate with the private registry
echo "[INFO] Authenticating with the private Docker registry..."
echo "$ARTIFACTORY_INTERNAL_PASSWORD" | docker login delivery.instana.io --username $ARTIFACTORY_INTERNAL_USERNAME --password-stdin

BUILD_CONTEXT=$WORKSPACE/$APP_REPO_FOLDER/ci/images/e2e-base-image/

# Build the container image
echo "[INFO] Building the operator image..."

docker buildx build \
    "$BUILD_CONTEXT"

echo "[INFO] Build process completed successfully."
