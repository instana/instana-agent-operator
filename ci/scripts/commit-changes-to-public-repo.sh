#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

abort_if_pr_exists() {
    echo "Check if a PR is already open"
    PR_LIST_JSON=$(curl -fH "Accept: application/vnd.github+json" https://api.github.com/repos/"$OWNER"/"$REPO"/pulls)
    EXISTING_PR_INFO_JSON=$(echo "$PR_LIST_JSON" | jq ".[] | select(.title | contains(\"$OPERATOR_PUBLIC_PR_NAME\"))")
    if [ -n "$EXISTING_PR_INFO_JSON" ]; then
        echo "A PR is already open, exiting"
        echo "A job can be retriggered once the open PR is resolved"
        exit 0
    fi
    echo "PR does not exist, creating a PR"
}

pushd instana-agent-operator-release

OLM_BUNDLE_ZIP=$(ls olm*.zip)
OPERATOR_RELEASE_VERSION="v$(echo "$OLM_BUNDLE_ZIP" | sed 's/olm-\(.*\)\.zip/\1/')"
OPERATOR_PUBLIC_PR_NAME="operator instana-agent-operator"
if [ "$REPO" == "redhat-marketplace-operators" ]; then
    OPERATOR_PUBLIC_PR_NAME="${OPERATOR_PUBLIC_PR_NAME}-rhmp"
fi

OLM_BUNDLE_ZIP_PATH="$(pwd)/$OLM_BUNDLE_ZIP"
export OPERATOR_PUBLIC_PR_NAME OWNER REPO

abort_if_pr_exists

popd

pushd "$PUBLIC_REPO_LOCAL_NAME"/operators/"$OPERATOR_NAME"

echo "Rebasing the fork from the upstream"
git remote add upstream https://github.com/"$OWNER"/"$REPO".git
git fetch upstream
git rebase upstream/main
git push --force
echo "Rebase successful"

COMMIT_MESSAGE="$OPERATOR_PUBLIC_PR_NAME ($OPERATOR_RELEASE_VERSION)"
echo "COMMIT_MESSAGE=$COMMIT_MESSAGE"
set -x
git pull -r
mkdir -p "$OPERATOR_RELEASE_VERSION"
unzip -o "$OLM_BUNDLE_ZIP_PATH" -d "$OPERATOR_RELEASE_VERSION"

if [ "$REPO" == "redhat-marketplace-operators" ]; then
    pushd "$OPERATOR_RELEASE_VERSION"

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


git config --global user.name "instanacd"
git config --global user.email "instanacd@instana.com"

git add .
git commit -s -m "$COMMIT_MESSAGE" --allow-empty

popd