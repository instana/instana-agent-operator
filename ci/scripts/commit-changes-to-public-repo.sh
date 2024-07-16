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
operator_release_version="v$(echo "$olm_bundle_zip" | sed 's/olm-\(.*\)\.zip/\1/')"
operator_public_pr_name="operator instana-agent-operator"
if [ "$REPO" == "redhat-marketplace-operators" ]; then
    operator_public_pr_name="${operator_public_pr_name}-rhmp"
fi

olm_bundle_zip_PATH="$(pwd)/$olm_bundle_zip"
commit_message="$operator_public_pr_name ($operator_release_version)"
new_release_branch=$OPERATOR_NAME-$operator_release_version

abort_if_pr_for_latest_version_exists
popd

pushd "$PUBLIC_REPO_LOCAL_NAME"/operators/"$OPERATOR_NAME"

echo "Creating a new feature branch from the upstream main"
git remote add upstream https://github.com/"$OWNER"/"$REPO".git 2>/dev/null || echo "Remote upstream already exists"
git fetch upstream
git checkout main
git reset --hard upstream/main
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

echo "https://${USERNAME}:${GH_API_TOKEN}@github.com" >> ~/.git-credentials
git config --global credential.helper store

git add .
git commit -s -m "$commit_message"
git push origin -u "${new_release_branch}"

popd