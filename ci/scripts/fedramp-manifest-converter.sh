#!/bin/bash

#
# (c) Copyright IBM Corp. 2025
#

# Script to download a specific version of the Instana Agent Operator manifest
# and replace the container registry path with the FedRAMP version

set -e

# Check if COMBINED_VERSION is provided
if [ -z "$COMBINED_VERSION" ]; then
  echo "Warning: COMBINED_VERSION environment variable is not set, aborting."
  exit 1
fi

# Check if VERSION is provided
if [ -z "$VERSION" ]; then
  echo "Warning: VERSION environment variable is not set, aborting."
  exit 1
fi

echo "Downloading Instana Agent Operator manifest version $VERSION..."

# Download the manifest file
MANIFEST_URL="https://github.com/instana/instana-agent-operator/releases/download/v${VERSION}/instana-agent-operator.yaml"
MANIFEST_FILE="instana-agent-operator-downloaded.yaml"
MANIFEST_FILE_FEDRAMP="instana-agent-operator.yaml"

echo "Downloading from: $MANIFEST_URL"
if ! curl -fL "$MANIFEST_URL" -o "$MANIFEST_FILE"; then
  echo "Error: Failed to download manifest from $MANIFEST_URL"
  exit 1
fi

echo "Download complete to $MANIFEST_FILE. Converting registry paths for FedRAMP..."

# Replace the container registry path and version
sed -e 's|icr.io/instana/|artifact-public.instana.io/rel-docker-agent-fedramp-virtual/|g' \
    -e "s|:${VERSION}|:${COMBINED_VERSION}|g" \
    "$MANIFEST_FILE" > "$MANIFEST_FILE_FEDRAMP"

echo "Conversion complete. FedRAMP manifest saved to: $MANIFEST_FILE_FEDRAMP"

# Made with Bob
