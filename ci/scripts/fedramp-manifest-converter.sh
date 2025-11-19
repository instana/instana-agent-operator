#!/bin/bash

#
# (c) Copyright IBM Corp. 2025
#

# Script to download a specific version of the Instana Agent Operator manifest
# and replace the container registry path with the FedRAMP version

set -e

# Check if COMBINED_VERSION is provided and valid
if [ -z "$COMBINED_VERSION" ]; then
  echo "Warning: COMBINED_VERSION environment variable is not set, aborting."
  exit 1
fi

# Validate COMBINED_VERSION format (should contain "fedramp-")
if [[ ! "$COMBINED_VERSION" =~ .*fedramp-.* ]]; then
  echo "Error: COMBINED_VERSION ($COMBINED_VERSION) does not have the expected format (should contain 'fedramp-')."
  exit 1
fi

# Check if VERSION is provided
if [ -z "$VERSION" ]; then
  echo "Warning: VERSION environment variable is not set, aborting."
  exit 1
fi

# Validate VERSION format (should be a semver-like string)
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
  echo "Error: VERSION ($VERSION) does not have the expected format (should be semver-like, e.g., 1.0.0 or 1.0.0-beta)."
  exit 1
fi

echo "Downloading Instana Agent Operator manifest version $VERSION..."

# Download the manifest file
MANIFEST_URL="https://github.com/instana/instana-agent-operator/releases/download/v${VERSION}/instana-agent-operator.yaml"
MANIFEST_FILE="instana-agent-operator-downloaded.yaml"
MANIFEST_FILE_FEDRAMP="instana-agent-operator.yaml"

echo "Downloading from: $MANIFEST_URL"

# Use curl with more verbose error reporting
HTTP_CODE=$(curl -fL "$MANIFEST_URL" -o "$MANIFEST_FILE" -w "%{http_code}" --connect-timeout 30 --retry 3 --retry-delay 5 --retry-max-time 60 2>/dev/null || echo "000")

if [ "$HTTP_CODE" != "200" ]; then
  echo "Error: Failed to download manifest from $MANIFEST_URL (HTTP code: $HTTP_CODE)"
  echo "Please verify that the release version v${VERSION} exists and contains the manifest file."
  exit 1
fi

# Verify the downloaded file is not empty and contains expected content
if [ ! -s "$MANIFEST_FILE" ] || ! grep -q "kind:" "$MANIFEST_FILE"; then
  echo "Error: Downloaded manifest file is empty or does not contain expected Kubernetes content."
  exit 1
fi

echo "Download complete to $MANIFEST_FILE. Converting registry paths for FedRAMP..."

# Replace the container registry path and version
sed -e 's|icr.io/instana/|containers.instana.io/instana/release/fedramp/agent/|g' \
    -e "s|:${VERSION}|:${COMBINED_VERSION}|g" \
    "$MANIFEST_FILE" > "$MANIFEST_FILE_FEDRAMP"

echo "Conversion complete. FedRAMP manifest saved to: $MANIFEST_FILE_FEDRAMP"

# Made with Bob
