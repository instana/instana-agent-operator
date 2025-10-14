#!/bin/bash

#
# (c) Copyright IBM Corp. 2025
#

# Script to find the latest release-* branch and read the FEDRAMP_VERSION file from it
# This replaces the current approach of reading from agent-operator-git-source/ci/FEDRAMP_VERSION

set -e

# Current branch is always main
CURRENT_BRANCH="main"

# Fetch all remote branches to ensure we have the latest information
echo "Fetching remote branches..." >&2
git config remote.origin.fetch "+refs/heads/*:refs/remotes/origin/*"
git fetch --all --quiet

# Find all release-* branches and sort them by version number
echo "Finding latest release branch..." >&2
LATEST_RELEASE_BRANCH=$(git branch -r | grep -E 'origin/release-[0-9]+\.[0-9]+' | sed 's/origin\///' | sort -V | tail -1)

if [ -z "$LATEST_RELEASE_BRANCH" ]; then
  echo "Error: No release-* branches found" >&2
  exit 1
fi

echo "Latest release branch: $LATEST_RELEASE_BRANCH" >&2

# Create a temporary branch to avoid detached HEAD state
TEMP_BRANCH="temp-${LATEST_RELEASE_BRANCH}-$(date +%s)"

# Check out the latest release branch to a temporary branch
echo "Checking out latest release branch to temporary branch $TEMP_BRANCH..." >&2
git checkout -b "$TEMP_BRANCH" "origin/$LATEST_RELEASE_BRANCH" --quiet

# Check if the FEDRAMP_VERSION file exists
if [ ! -f "ci/FEDRAMP_VERSION" ]; then
  echo "Error: ci/FEDRAMP_VERSION file not found in $LATEST_RELEASE_BRANCH" >&2
  git checkout "$CURRENT_BRANCH" --quiet
  git branch -D "$TEMP_BRANCH" --quiet
  exit 1
fi

# Read the FEDRAMP_VERSION file
ARTIFACT_VERSION=$(cat ci/FEDRAMP_VERSION)
echo "ARTIFACT_VERSION=$ARTIFACT_VERSION" >&2

# Return to main branch
echo "Returning to main branch..." >&2
git checkout main --quiet

# Clean up the temporary branch
git branch -D "$TEMP_BRANCH" --quiet

# Output the version for use in scripts - this is the only output to stdout
echo "$ARTIFACT_VERSION"

# Made with Bob
