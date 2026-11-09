#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Helper script to calculate CI Server specific metadata to be used for locking.
# When invoked, it generates a unique lock owner name from concourse meta-data and
# fetched the most recent go binary from github to execute the lock or release command. 
set -euo pipefail
echo "===== reslock.sh - start ====="
RESLOCK_COMMAND=$1
RESLOCK_RESOURCE_NAME=$2

echo "${RESLOCK_COMMAND} lock ${RESLOCK_RESOURCE_NAME}"

export RESLOCK_GITHUB_REPO_OWNER=instana
export RESLOCK_LOCK_OWNER="${APP_REPO_NAME}/${BRANCH}/${PIPELINE_RUN_NAME}/${BUILD_NUMBER}"
RESLOCK_GITHUB_TOKEN=$(get_secret reslock-github-token)
export RESLOCK_GITHUB_TOKEN

echo "RESLOCK_GITHUB_REPO_OWNER=${RESLOCK_GITHUB_REPO_OWNER}"
echo "RESLOCK_LOCK_OWNER=${RESLOCK_LOCK_OWNER}"

RESLOCK_RUNNER=$(mktemp)
trap 'rm -f "${RESLOCK_RUNNER}"' EXIT
curl --fail --silent --show-error --retry 3 \
  -H "Authorization: Bearer ${RESLOCK_GITHUB_TOKEN}" \
  'https://raw.github.ibm.com/instana/reslock/main/run.sh' > "${RESLOCK_RUNNER}"
test -s "${RESLOCK_RUNNER}"
bash -n "${RESLOCK_RUNNER}"
if [ "${RESLOCK_COMMAND}" == "claim" ]; then
    bash "${RESLOCK_RUNNER}" claim k8s-clusters "${RESLOCK_RESOURCE_NAME}" -t 70m -w
else
    bash "${RESLOCK_RUNNER}" "${RESLOCK_COMMAND}" k8s-clusters "${RESLOCK_RESOURCE_NAME}"
fi
echo "===== reslock.sh - end ====="
