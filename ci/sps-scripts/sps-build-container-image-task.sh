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

echo "[INFO] Setting up Docker Buildx for multi-platform builds..."
docker buildx create --name multiarch-builder --use
docker buildx inspect --bootstrap

# Determine the build context based on flavor
echo "[INFO] Determining the appropriate build context..."
if [ "$2" = "dynamic" ]; then
    BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/dynamic"
    echo "[INFO] Using dynamic build context: $BUILD_CONTEXT"
elif [ "$2" = "static" ]; then
    BUILD_CONTEXT="$WORKSPACE/$APP_REPO_FOLDER/static"
    echo "[INFO] Using static build context: $BUILD_CONTEXT"
else
    echo "[ERROR] Unknown FLAVOR specified: $2"
    exit 1
fi

# Create a secure temporary file for secrets
echo "[INFO] Creating a secure temporary file for secrets..."
SECRET_DIR=$(mktemp -d $WORKSPACE/$APP_REPO_FOLDER/secrets.XXXXXX)
SECRET_FILE="$SECRET_DIR/DOWNLOAD_KEY"
echo -n "$QA_AGENT_DOWNLOAD_KEY" > "$SECRET_FILE"
chmod 600 "$SECRET_FILE"
echo "[INFO] Secret file created at $SECRET_FILE"

# Determine architecture based on TARGETPLATFORM from Dockerfile
export arch=$(case "$1" in
    'linux/amd64') echo 'x86_64' ;;
    'linux/arm64') echo 'aarch64' ;;
    'linux/s390x') echo 's390x' ;;
    'linux/ppc64le') echo 'ppc64le' ;;
    *) echo 'Unknown Architecture!!!' ;;
esac)

echo "[INFO] Verifying script arguments..."
echo "TARGETPLATFORM: $1"
echo "FLAVOR: $2"
echo "CLASSIFIER: $3"
echo "ARCHITECTURE: $arch"
echo "Version: $INSTANA_TWISTCLI_VERSION"

# Construct the image tag with optional CLASSIFIER
if [ -n "$3" ]; then
    IMAGE_TAG="instana-agent-docker-$arch-$2-$3:$INSTANA_TWISTCLI_VERSION"
else
    IMAGE_TAG="instana-agent-docker-$arch-$2:$INSTANA_TWISTCLI_VERSION"
fi
echo "IMAGE TAG: $IMAGE_TAG"

# Build the container image
echo "[INFO] Building the container image..."
docker buildx build \
    "$BUILD_CONTEXT" \
    --platform "$1" \
    --build-arg TARGETPLATFORM="$1" \
    --build-arg FLAVOR="$2" \
    --build-arg CLASSIFIER="$3" \
    --secret id=DOWNLOAD_KEY,src=$SECRET_FILE \
    -t $IMAGE_TAG \
    --load \
    --progress=plain \
    --no-cache \
    --output type=oci,dest=$WORKSPACE/$APP_REPO_FOLDER/image.tar \

echo "[INFO] Build process completed successfully."

echo "[INFO] Saving the built image to the output location..."
docker images
docker save -o "$WORKSPACE/$APP_REPO_FOLDER/image.tar" "$IMAGE_TAG"

echo "[INFO] Docker image saved successfully to $WORKSPACE/$APP_REPO_FOLDER/image.tar"