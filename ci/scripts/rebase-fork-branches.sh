#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

collect_pr_branches() {
    echo "Collect all open PR branches"
    pr_list_json=$(curl -fH "Accept: application/vnd.github+json" https://api.github.com/repos/"$OWNER"/"$REPO"/pulls)
    pr_branches=$(echo "$pr_list_json" | jq -r ".[] | select(.title | contains(\"$operator_public_pr_name\")) | .head.ref")

    if [ -z "$pr_branches" ]; then 
        echo "No open PRs were found with the $operator_public_pr_name title"
    else 
        echo "Found the open PRs with the following branch names:"
        echo "$pr_branches"
    fi
}

rebase_branches() {
    echo "Rebasing the forked main and the found PR branches from the upstream main branch"
    pushd "$PUBLIC_REPO_LOCAL_NAME"
    echo "Rebasing the forked main"
    git config --global user.name "instanacd"
    git config --global user.email "instanacd@instana.com"
    git remote add upstream https://github.com/"$OWNER"/"$REPO".git 2>/dev/null || echo "Remote upstream already exists"
    git fetch upstream
    git checkout main
    git rebase upstream/main
    echo "Rebase of the forked main branch was successful"

    echo "Rebasing the feature branches"
    for branch in $pr_branches; do
        echo "Rebasing PR branch $branch"
        git checkout "$branch"
        git rebase main
        echo "Rebase of PR branch $branch was successful"
    done
    popd
}


operator_public_pr_name="operator instana-agent-operator"
if [ "$REPO" == "redhat-marketplace-operators" ]; then
    operator_public_pr_name="${operator_public_pr_name}-rhmp"
fi

collect_pr_branches
rebase_branches