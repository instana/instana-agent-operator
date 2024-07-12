#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

cd agent-operator-git-source
git pull -r
latest_release=$(git tag | sort -r --version-sort | head -n1)
echo "Latest release is ${latest_release}"

new_commits=$(git log "${latest_release}"..HEAD --oneline)

if [ -z "$new_commits" ]; then 
    echo "No new commits since the last release"
    exit 1
fi

# Assisted by WCA@IBM
# Latest GenAI contribution: ibm/granite-20b-code-instruct-v2

only_ci_changes=true
for file in $(git diff --name-only "$latest_release"..HEAD); do 
    # check if $file does not start with "ci/"
    if [[ ! $file =~ ^ci/ ]]; then
        echo "Found file that is not in ci/ directory: $file"
        only_ci_changes=false
        break
    fi
done
# Assisted by WCA@IBM
# Latest GenAI contribution: ibm/granite-20b-code-instruct-v2

if $only_ci_changes; then 
    echo "Only ci/ files have been changed since the last release. Aborting the release"
    exit 1
fi

new_release=$(echo "$latest_release" | awk -F. '/[0-9]+\./{$NF++;print}' OFS=.)

echo "Tagging repo with the new release tag ${new_release}"
git config --global user.name "instanacd"
git config --global user.email "instanacd@instana.com"
git tag "${new_release}"
echo "${new_release}" > ci/version.txt