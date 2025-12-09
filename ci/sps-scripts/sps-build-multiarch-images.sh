#!/bin/sh
#
# (c) Copyright IBM Corp. 2025
#
set -e

# Load environment variables
ICR_REGISTRY_DOMAIN="icr.io"
GIT_COMMIT=$(load_repo app-repo commit)
BRANCH_NAME=$(load_repo app-repo branch)

echo "Building commit ${GIT_COMMIT} on branch ${BRANCH_NAME}"

# Authenticate with the private registry, see https://cloud.ibm.com/docs/devsecops?topic=devsecops-cd-devsecops-build-docker#cd-devsecops-work-with-icr
echo "[INFO] Authenticating with the $ICR_REGISTRY_DOMAIN private Docker registry..."
# see: https://instana.slack.com/archives/CP56J2USY/p1763054646373449?thread_ts=1762872461.330849&cid=CP56J2USY
export DOCKER_API_VERSION="1.41"
docker login -u iamapikey --password-stdin "$ICR_REGISTRY_DOMAIN" < /config/api-key

echo "[INFO] Setting up Docker Buildx for multi-platform builds..."
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

echo "pwd: $(pwd)"

BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/"

# Define registry and image names
REGISTRY_IMAGE_TAG_ICR="${ICR_REGISTRY_DOMAIN}/instana-agent-dev/instana-agent-operator:${GIT_COMMIT}"

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
    --push \
    "$BUILD_CONTEXT"

# mark for scanning
pipelinectl save_artifact operator_image "name=${REGISTRY_IMAGE_TAG_ICR}" "type=image"
