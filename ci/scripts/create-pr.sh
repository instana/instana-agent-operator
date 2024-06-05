#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

source pipeline-source/ci/scripts/commit-changes-to-public-repo.sh

pushd instana-agent-operator-release

OLM_BUNDLE_ZIP=$(ls olm*.zip)
OPERATOR_RELEASE_VERSION=$(echo "$OLM_BUNDLE_ZIP" | sed 's/olm-\(.*\)\.zip/\1/')
OPERATOR_PUBLIC_PR_NAME="operator instana-agent-operator"
if [ "$REPO" == "redhat-marketplace-operators" ]; then
    OPERATOR_PUBLIC_PR_NAME="${OPERATOR_PUBLIC_PR_NAME}-rhmp"
fi
COMMIT_MESSAGE="$OPERATOR_PUBLIC_PR_NAME ($OPERATOR_RELEASE_VERSION)"
export OPERATOR_PUBLIC_PR_NAME OWNER REPO

abort_if_pr_exists

popd

# Create the PR
set -x
curl \
-fX POST \
-H "Accept: application/vnd.github+json" \
-H "Authorization: Bearer $GH_API_TOKEN" \
https://api.github.com/repos/$OWNER/$REPO/pulls \
-d "{\"title\":\"$COMMIT_MESSAGE\",\"head\":\"instana:main\",\"base\":\"main\"}"