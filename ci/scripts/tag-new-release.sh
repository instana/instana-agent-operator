#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

echo "Running on branch $BRANCH"
cd agent-operator-git-source
git pull -r

# Handle different branch scenarios
if [[ $BRANCH =~ ^release-([0-9]+\.[0-9]+)$ ]]; then
    # Extract major.minor from branch name
    major_minor="${BASH_REMATCH[1]}"
    echo "Release branch detected: $BRANCH with major.minor version $major_minor"

    # Find the latest tag with this major.minor
    latest_release=$(git tag | grep "^v${major_minor}\." | sort -r --version-sort | head -n1)

    if [ -z "$latest_release" ]; then
        # No existing tag with this major.minor, create first patch version
        latest_release="v${major_minor}.0"
        echo "No existing release found for $major_minor, using $latest_release as base"
    else
        echo "Latest release for $major_minor is ${latest_release}"
    fi
else
    # Main branch - get the overall latest release
    latest_release=$(git tag | sort -r --version-sort | head -n1)
    echo "Main branch detected, latest release is ${latest_release}"
fi

new_commits=$(git log "${latest_release}"..HEAD --oneline)

if [ -z "$new_commits" ]; then
    echo "No new commits since the last release"
    exit 1
fi

# Assisted by WCA@IBM
# Latest GenAI contribution: ibm/granite-20b-code-instruct-v2

only_ci_changes=true
for file in $(git diff --name-only "$latest_release"..HEAD); do 
    # check if the file path does not start with "ci/", ".", doesn't end with .md, nor it's a Makefile
    if [[ $file == ci* ]]; then
        continue
    fi
    if [[ $file == .* ]]; then
        continue
    fi
    if [[ $file == *.md ]]; then
        continue
    fi
    if [[ $file =~ Makefile ]]; then
        continue
    fi
    echo "Found file that is not in ci/ directory, it's not a hidden file/directory, an .md file nor Makefile: $file"
    only_ci_changes=false
    break
done
# Assisted by WCA@IBM
# Latest GenAI contribution: ibm/granite-20b-code-instruct-v2

if $only_ci_changes; then 
    echo "Only ci/ files have been changed since the last release. Aborting the release"
    exit 1
fi

# Handle version increment based on the latest_release and TRIGGER_TYPE
if [[ $latest_release =~ ^v([0-9]+)\.([0-9]+)\.([0-9]+)$ ]]; then
    major="${BASH_REMATCH[1]}"
    minor="${BASH_REMATCH[2]}"
    patch="${BASH_REMATCH[3]}"

    # Check if this is a minor version increment
    if [[ "${TRIGGER_TYPE}" == "minor" ]]; then
        echo "Minor version increment requested"
        # Increment minor version and reset patch to 0
        new_minor=$((minor + 1))
        new_release="v${major}.${new_minor}.0"
    else
        echo "Patch version increment requested (default)"
        # Increment patch version
        new_patch=$((patch + 1))
        new_release="v${major}.${minor}.${new_patch}"
    fi
else
    # Fail if the tag format is unexpected
    echo "ERROR: Unexpected tag format: ${latest_release}"
    echo "Expected format: v<major>.<minor>.<patch> (e.g., v1.2.3)"
    exit 1
fi

echo "Tagging repo with the new release tag ${new_release}"
git config --global user.name "instanacd"
git config --global user.email "instanacd@instana.com"
git tag "${new_release}"
echo "${new_release}" > ci/version