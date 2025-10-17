#!/bin/bash
set -euo pipefail

# Extract the new version from the release
NEW_VERSION=$(ls instana-agent-operator-release/olm-*.zip | grep -oP 'olm-\K[0-9]+\.[0-9]+\.[0-9]+' || echo "")
if [ -z "$NEW_VERSION" ]; then
echo "Failed to extract version from release artifact"
exit 1
fi

echo "New version: $NEW_VERSION"

# Get the latest release version from GitHub API
LATEST_RELEASE=$(curl -s https://api.github.com/repos/instana/instana-agent-operator/releases/latest | jq -r '.tag_name' | grep -oP 'v\K[0-9]+\.[0-9]+\.[0-9]+' || echo "")
if [ -z "$LATEST_RELEASE" ]; then
echo "Failed to get latest release version, proceeding with PR creation"
exit 0
fi

echo "Latest release: $LATEST_RELEASE"

# Compare versions using semver logic
# Split versions into components
IFS='.' read -r -a NEW_PARTS <<< "$NEW_VERSION"
IFS='.' read -r -a LATEST_PARTS <<< "$LATEST_RELEASE"

# Compare major version
if [ "${NEW_PARTS[0]}" -gt "${LATEST_PARTS[0]}" ]; then
echo "New version $NEW_VERSION is semver-greater than latest release $LATEST_RELEASE (major version higher)"
echo "Proceeding with PR creation"
exit 0
elif [ "${NEW_PARTS[0]}" -eq "${LATEST_PARTS[0]}" ]; then
# Compare minor version
if [ "${NEW_PARTS[1]}" -gt "${LATEST_PARTS[1]}" ]; then
echo "New version $NEW_VERSION is semver-greater than latest release $LATEST_RELEASE (minor version higher)"
echo "Proceeding with PR creation"
exit 0
elif [ "${NEW_PARTS[1]}" -eq "${LATEST_PARTS[1]}" ]; then
# Compare patch version
if [ "${NEW_PARTS[2]}" -gt "${LATEST_PARTS[2]}" ]; then
    echo "New version $NEW_VERSION is semver-greater than latest release $LATEST_RELEASE (patch version higher)"
    echo "Proceeding with PR creation"
    exit 0
fi
fi
fi

# If versions are identical, proceed with PR creation
if [ "$NEW_VERSION" = "$LATEST_RELEASE" ]; then
echo "New version $NEW_VERSION is identical to latest release $LATEST_RELEASE"
echo "Proceeding with PR creation"
exit 0
fi

# Only skip if new version is lower than latest release
echo "New version $NEW_VERSION is lower than latest release $LATEST_RELEASE"
echo "Skipping PR creation"
exit 1