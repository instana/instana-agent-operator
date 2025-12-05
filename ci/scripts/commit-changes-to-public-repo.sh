#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

# Ensure we're in the correct directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
cd "$REPO_ROOT"

abort_if_pr_for_latest_version_exists() {
    set +e
    echo "Check if a PR is already open"
    pr_list_json=$(curl -fH "Accept: application/vnd.github+json" https://api.github.com/repos/"$OWNER"/"$REPO"/pulls)
    existing_pr_info_json=$(echo "$pr_list_json" | jq ".[] | select(.title == (\"$commit_message\"))")
    if [ -n "$existing_pr_info_json" ]; then
        echo "A PR is already open, exiting"
        exit 0
    fi
    echo "PR does not exist"
    set -e
}

# Download the latest release artifacts
echo "=== Downloading Latest GitHub Release Artifacts ==="
echo "Current working directory: $(pwd)"
mkdir -p instana-agent-operator-release

echo "Fetching latest release from GitHub..."
LATEST_RELEASE_JSON=$(curl -s https://api.github.com/repos/instana/instana-agent-operator/releases/latest)
LATEST_TAG=$(echo "$LATEST_RELEASE_JSON" | jq -r '.tag_name')

if [ -z "$LATEST_TAG" ] || [ "$LATEST_TAG" = "null" ]; then
    echo "ERROR: Failed to fetch latest release tag from GitHub"
    exit 1
fi

echo "Latest release tag: $LATEST_TAG"

# Download the OLM bundle zip from the latest release
OLM_ZIP_URL=$(echo "$LATEST_RELEASE_JSON" | jq -r '.assets[] | select(.name | startswith("olm-")) | .browser_download_url')

if [ -z "$OLM_ZIP_URL" ] || [ "$OLM_ZIP_URL" = "null" ]; then
    echo "ERROR: Failed to find OLM bundle zip in latest release"
    exit 1
fi

echo "Downloading OLM bundle from: $OLM_ZIP_URL"
OLM_ZIP_FILENAME=$(basename "$OLM_ZIP_URL")
curl -L -o "instana-agent-operator-release/$OLM_ZIP_FILENAME" "$OLM_ZIP_URL"

if [ ! -f "instana-agent-operator-release/$OLM_ZIP_FILENAME" ]; then
    echo "ERROR: Failed to download OLM bundle"
    exit 1
fi

echo "Successfully downloaded OLM bundle: $OLM_ZIP_FILENAME"
echo "=== Download Complete ==="

# Verify directory exists before entering it
if [ ! -d "instana-agent-operator-release" ]; then
    echo "ERROR: Directory instana-agent-operator-release does not exist"
    echo "Contents of current directory:"
    ls -la
    exit 1
fi

pushd instana-agent-operator-release

olm_bundle_zip=$(ls olm*.zip)
operator_release_version="$(echo "$olm_bundle_zip" | sed 's/olm-\(.*\)\.zip/\1/')"
operator_public_pr_name="operator instana-agent-operator"
if [ "$REPO" == "redhat-marketplace-operators" ]; then
    operator_public_pr_name="${operator_public_pr_name}-rhmp"
fi

olm_bundle_zip_PATH="$(pwd)/$olm_bundle_zip"
commit_message="$operator_public_pr_name ($operator_release_version)"
new_release_branch=$OPERATOR_NAME-$operator_release_version

abort_if_pr_for_latest_version_exists
popd

# Clone the repository if it doesn't exist
if [ ! -d "$PUBLIC_REPO_LOCAL_NAME" ]; then
    echo "Cloning the forked repository"
    git clone "https://github.com/instana/$REPO.git" "$PUBLIC_REPO_LOCAL_NAME"
else
    echo "Repository already exists at $PUBLIC_REPO_LOCAL_NAME"
fi

pushd "$PUBLIC_REPO_LOCAL_NAME"/operators/"$OPERATOR_NAME"

echo "Creating a new feature branch from the fork's main branch"
git remote add upstream https://github.com/"$OWNER"/"$REPO".git 2>/dev/null || echo "Remote upstream already exists"
git fetch upstream
git checkout main
# Don't reset to upstream/main to avoid workflow file changes that require workflow scope
# git reset --hard upstream/main
git checkout -b "$new_release_branch"
echo "Creation of the new PR branch was successful"

echo "commit_message=$commit_message"
set -x
mkdir -p "$operator_release_version"
unzip -o "$olm_bundle_zip_PATH" -d "$operator_release_version"

if [ "$REPO" == "redhat-marketplace-operators" ]; then
    pushd "$operator_release_version"

    pushd manifests
    yq -i '.metadata.annotations += {"marketplace.openshift.io/remote-workflow": "https://marketplace.redhat.com/en-us/operators/instana-agent-operator-rhmp/pricing?utm_source=openshift_console"}' instana-agent-operator.clusterserviceversion.yaml
    yq -i '.metadata.annotations += {"marketplace.openshift.io/support-workflow": "https://marketplace.redhat.com/en-us/operators/instana-agent-operator-rhmp/support?utm_source=openshift_console"}' instana-agent-operator.clusterserviceversion.yaml
    mv instana-agent-operator.clusterserviceversion.yaml instana-agent-operator-rhmp.clusterserviceversion.yaml
    popd

    pushd metadata
    yq -i '.annotations."operators.operatorframework.io.bundle.package.v1" |= "instana-agent-operator-rhmp"' annotations.yaml
    popd

    popd
fi


git config --global user.name "$USERNAME"
git config --global user.email "instanacd@instana.com"

# Ensure we use the correct credentials for the git push
echo "https://${USERNAME}:${GH_API_TOKEN}@github.com" > ~/.git-credentials
git config --global credential.helper store

# Set the remote URL to use the authenticated URL
git remote set-url origin "https://${USERNAME}:${GH_API_TOKEN}@github.com/instana/${REPO}.git"

# Remove any .github directory that might have been extracted from the bundle
# This prevents workflow scope issues with the PAT
if [ -d "$operator_release_version/.github" ]; then
    echo "Removing .github directory from bundle to avoid workflow scope issues"
    rm -rf "$operator_release_version/.github"
fi

# Only add the specific operator version directory to avoid touching .github workflows
# This prevents workflow scope issues with the PAT
git add "$operator_release_version"
git commit -s -m "$commit_message"
git push origin -u "${new_release_branch}" --force

# Create PR with better error handling
echo "Creating PR with title: $commit_message"
echo "Head branch: instana:${new_release_branch}"
echo "Base branch: main"

PR_RESPONSE=$(curl \
    -w "\n%{http_code}" \
    -X POST \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GH_API_TOKEN" \
    https://api.github.com/repos/$OWNER/$REPO/pulls \
    -d "{\"title\":\"$commit_message\",\"head\":\"instana:${new_release_branch}\",\"base\":\"main\"}")

HTTP_CODE=$(echo "$PR_RESPONSE" | tail -n1)
RESPONSE_BODY=$(echo "$PR_RESPONSE" | sed '$d')

echo "HTTP Status Code: $HTTP_CODE"
echo "Response Body: $RESPONSE_BODY"

if [ "$HTTP_CODE" -ne 201 ]; then
    echo "ERROR: Failed to create PR. HTTP Status: $HTTP_CODE"
    echo "Response: $RESPONSE_BODY"
    
    # Check if PR already exists
    if echo "$RESPONSE_BODY" | grep -q "A pull request already exists"; then
        echo "A PR already exists for this branch. This is not an error, exiting successfully."
        exit 0
    fi
    
    # Check for validation errors
    if echo "$RESPONSE_BODY" | grep -q "Validation Failed"; then
        echo "Validation failed. Please check the branch exists and the request is valid."
    fi
    
    exit 1
fi

echo "PR created successfully!"

popd

# Made with Bob
