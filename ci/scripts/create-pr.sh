#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

source pipeline-source/ci/scripts/helpers.sh

pushd instana-agent-operator-release

olm_bundle_zip=$(ls olm*.zip)
operator_release_version=$(echo "$olm_bundle_zip" | sed 's/olm-\(.*\)\.zip/\1/')
operator_public_pr_name="operator instana-agent-operator"
if [ "$REPO" == "redhat-marketplace-operators" ]; then
    operator_public_pr_name="${operator_public_pr_name}-rhmp"
fi
commit_message="$operator_public_pr_name ($operator_release_version)"
export operator_public_pr_name OWNER REPO

abort_if_pr_for_latest_version_exists

popd

# Create the PR
set -x
curl \
    -fX POST \
    -H "Accept: application/vnd.github+json" \
    -H "Authorization: Bearer $GH_API_TOKEN" \
    https://api.github.com/repos/$OWNER/$REPO/pulls \
    -d "{\"title\":\"$commit_message\",\"head\":\"instana:main\",\"base\":\"main\"}"