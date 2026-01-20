#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail
# Set up git configuration
BRANCH=$(load_repo app-repo branch)
GH_API_TOKEN=$(get_env github-token)
git config --global user.email "instanacd@instana.com"
git config --global user.name "Instana CD"

# Check if we're on main branch
if [[ "${BRANCH}" == "main" ]]; then
echo "Skipping FEDRAMP_VERSION update on main branch"
exit 0
fi

# Check if we're on a release branch
if [[ "${BRANCH}" != release-* ]]; then
echo "Not on a release branch (${BRANCH}), skipping FEDRAMP_VERSION update"
exit 0
fi

echo "On release branch ${BRANCH}, updating FEDRAMP_VERSION"
echo "Operator version: ${VERSION}"

# Update the FEDRAMP_VERSION file
echo "${VERSION}" > ci/FEDRAMP_VERSION
echo "Updated FEDRAMP_VERSION to ${VERSION}"

# Commit the change
git add ci/FEDRAMP_VERSION
git commit -m "Update FEDRAMP_VERSION to ${VERSION}"

# Push to the current release branch
echo "Pushing changes to ${BRANCH}"
git push https://instanacd:${GH_API_TOKEN}@github.com/instana/instana-agent-operator.git HEAD:${BRANCH}

echo "FEDRAMP_VERSION update completed successfully"