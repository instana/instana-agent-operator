#!/usr/bin/env bash
set -e

# This script checks if all architecture-specific images exist before creating a manifest list
# It's designed to be used in the CI/CD pipeline to prevent failures when images are missing

# Load environment variables
export ARTIFACTORY_INTERNAL_USERNAME=${ARTIFACTORY_INTERNAL_USERNAME:-$(get_env ARTIFACTORY_INTERNAL_USERNAME)}
export ARTIFACTORY_INTERNAL_PASSWORD=${ARTIFACTORY_INTERNAL_PASSWORD:-$(get_env ARTIFACTORY_INTERNAL_PASSWORD)}
export GIT_COMMIT=${GIT_COMMIT:-$(get_env branch || echo "latest")}
export IMAGE_TAG=${IMAGE_TAG:-${GIT_COMMIT}}

# Set up the image name
OPERATOR_IMAGE_NAME=${OPERATOR_IMAGE_NAME:-delivery.instana.io/int-docker-agent-local/instana-agent-operator/dev-build}

# Login to the registry
echo "Logging in to registry..."
echo "$ARTIFACTORY_INTERNAL_PASSWORD" | docker login delivery.instana.io --username $ARTIFACTORY_INTERNAL_USERNAME --password-stdin

# Check if all architecture-specific images exist
ARCHITECTURES=("x86_64" "aarch64" "ppc64le" "s390x")
MISSING_IMAGES=()

echo "Checking if all architecture-specific images exist..."
for arch in "${ARCHITECTURES[@]}"; do
    image_name="${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}-${arch}"
    echo "Checking if image exists: $image_name"
    
    # Use skopeo to check if the image exists
    if ! skopeo inspect --creds="${ARTIFACTORY_INTERNAL_USERNAME}:${ARTIFACTORY_INTERNAL_PASSWORD}" "docker://${image_name}" &>/dev/null; then
        echo "Warning: Image $image_name does not exist."
        MISSING_IMAGES+=("$arch")
    else
        echo "Image $image_name exists."
    fi
done

# If any images are missing, exit with a helpful message
if [ ${#MISSING_IMAGES[@]} -gt 0 ]; then
    echo "Error: The following architecture-specific images are missing:"
    for arch in "${MISSING_IMAGES[@]}"; do
        echo "  - ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}-${arch}"
    done
    echo "Please ensure all architecture-specific images are built and pushed before creating the manifest list."
    echo "Skipping manifest creation to prevent failure."
    
    # Create an empty digest file to indicate failure
    mkdir -p manifest-list
    echo "sha256:0000000000000000000000000000000000000000000000000000000000000000" > manifest-list/digest
    echo "Created placeholder digest due to missing images."
    echo "ERROR: Manifest creation was skipped due to missing images."
    exit 1
fi

# If all images exist, proceed with creating the manifest
echo "All architecture-specific images exist. Creating manifest list..."

# Install manifest-tool if not already installed
if ! command -v manifest-tool &>/dev/null; then
    echo "Installing manifest-tool..."
    git clone https://github.com/estesp/manifest-tool
    cd manifest-tool && make binary
    export PATH=$PATH:$(pwd)
    cd ..
fi

# Create the manifest list
echo "Creating manifest list..."
mkdir -p manifest-list

# Create a temporary YAML file for manifest-tool
cat > manifest-config.yaml << EOF
image: ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}
manifests:
  - image: ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}-x86_64
    platform:
      architecture: amd64
      os: linux
  - image: ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}-aarch64
    platform:
      architecture: arm64
      os: linux
  - image: ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}-ppc64le
    platform:
      architecture: ppc64le
      os: linux
  - image: ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG}-s390x
    platform:
      architecture: s390x
      os: linux
EOF

manifest-tool \
  --username ${ARTIFACTORY_INTERNAL_USERNAME} \
  --password ${ARTIFACTORY_INTERNAL_PASSWORD} \
  push from-spec manifest-config.yaml | tee manifest-output.txt

# Clean up the temporary file
rm manifest-config.yaml

# Extract and save the digest
OPERATOR_IMG_DIGEST=$(awk '{ print $2 }' manifest-output.txt)
echo "OPERATOR_IMG_DIGEST=$OPERATOR_IMG_DIGEST"
echo ${OPERATOR_IMG_DIGEST} > manifest-list/digest

echo "Manifest list created successfully."

