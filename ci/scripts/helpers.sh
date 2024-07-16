#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

abort_if_pr_for_latest_version_exists() {
    echo "Check if a PR is already open"
    pr_list_json=$(curl -fH "Accept: application/vnd.github+json" https://api.github.com/repos/"$OWNER"/"$REPO"/pulls)
    existing_pr_info_json=$(echo "$pr_list_json" | jq ".[] | select(.title == (\"$commit_message\"))")
    if [ -n "$existing_pr_info_json" ]; then
        echo "A PR is already open, exiting"
        exit 0
    fi
    echo "PR does not exist, creating a PR"
}
