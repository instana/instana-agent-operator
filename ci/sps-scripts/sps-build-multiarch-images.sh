#!/bin/sh
set -e

# Load environment variables
export ARTIFACTORY_INTERNAL_USERNAME=$(get_env ARTIFACTORY_INTERNAL_USERNAME)
export ARTIFACTORY_INTERNAL_PASSWORD=$(get_env ARTIFACTORY_INTERNAL_PASSWORD)
export QA_AGENT_DOWNLOAD_KEY=$(get_env QA_AGENT_DOWNLOAD_KEY)
export INSTANA_TWISTCLI_VERSION=$(get_env INSTANA_TWISTCLI_VERSION)
export GIT_COMMIT=$(get_env branch || echo "latest")

# Authenticate with the private registry
echo "[INFO] Authenticating with the private Docker registry..."
echo "$ARTIFACTORY_INTERNAL_PASSWORD" | docker login delivery.instana.io --username $ARTIFACTORY_INTERNAL_USERNAME --password-stdin

echo "[INFO] Setting up Docker Buildx for multi-platform builds..."
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/"

# Create a secure temporary file for secrets
echo "[INFO] Creating a secure temporary file for secrets..."
SECRET_DIR=$(mktemp -d $WORKSPACE/$APP_REPO_FOLDER/secrets.XXXXXX)
SECRET_FILE="$SECRET_DIR/DOWNLOAD_KEY"
echo -n "$QA_AGENT_DOWNLOAD_KEY" > "$SECRET_FILE"
chmod 600 "$SECRET_FILE"
echo "[INFO] Secret file created at $SECRET_FILE"

# Define registry and image names
REGISTRY="delivery.instana.io"
REPO_PATH="int-docker-agent-local/instana-agent-operator/dev-build"
FULL_REPO_PATH="${REGISTRY}/${REPO_PATH}"

# Define all target platforms
PLATFORMS=("linux/amd64" "linux/arm64" "linux/s390x" "linux/ppc64le")

# Build and push images for each platform
for platform in "${PLATFORMS[@]}"; do
    # Determine architecture based on platform
    arch=$(case "$platform" in
        'linux/amd64') echo 'x86_64' ;;
        'linux/arm64') echo 'aarch64' ;;
        'linux/s390x') echo 's390x' ;;
        'linux/ppc64le') echo 'ppc64le' ;;
        *) echo 'Unknown Architecture!!!' ;;
    esac)
    
    echo "[INFO] Building for platform: $platform, architecture: $arch"
    
    # Construct the image tags
    LOCAL_IMAGE_TAG="instana-agent-docker-$arch:$INSTANA_TWISTCLI_VERSION"
    REGISTRY_IMAGE_TAG="${FULL_REPO_PATH}:${GIT_COMMIT}-${arch}"
    
    echo "[INFO] LOCAL IMAGE TAG: $LOCAL_IMAGE_TAG"
    echo "[INFO] REGISTRY IMAGE TAG: $REGISTRY_IMAGE_TAG"
    
    # Build the container image
    echo "[INFO] Building the container image for $platform..."
    docker buildx build \
        "$BUILD_CONTEXT" \
        --platform "$platform" \
        --build-arg TARGETPLATFORM="$platform" \
        --secret id=DOWNLOAD_KEY,src=$SECRET_FILE \
        -t $LOCAL_IMAGE_TAG \
        -t $REGISTRY_IMAGE_TAG \
        --load \
        --progress=plain \
        --no-cache
    
    echo "[INFO] Build process completed successfully for $platform."
    
    # Push the image to the registry with the architecture-specific tag
    echo "[INFO] Pushing image to registry with tag: $REGISTRY_IMAGE_TAG"
    docker push $REGISTRY_IMAGE_TAG
    
    echo "[INFO] Image successfully pushed to registry: $REGISTRY_IMAGE_TAG"
done

echo "[INFO] All architecture builds completed successfully."

# Now create the manifest list
echo "[INFO] Creating manifest list for all architectures..."
dnf -y install microdnf
source $WORKSPACE/$APP_REPO_FOLDER/installGolang.sh 1.24.4
export PATH=$PATH:/usr/local/go/bin
source $WORKSPACE/$APP_REPO_FOLDER/ci/sps-scripts/check-and-create-manifest.sh

echo "[INFO] Multi-architecture build and manifest creation completed."

# Made with Bob
