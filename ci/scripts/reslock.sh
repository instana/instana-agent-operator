#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

# Helper script to calculate CI Server specific metadata to be used for locking.
# When invoked, it generates a unique lock owner name from concourse meta-data and
# fetched the most recent go binary from github to execute the lock or release command. 
set -e
set -o pipefail
set -u # fail if env var is not set

echo "${RESLOCK_COMMAND} lock ${RESLOCK_RESOURCE_NAME}"

echo "Reading concourse metada"
ls -lah metadata/
# injecting concourse metadata to be added as lock owner
BUILD_PIPELINE_NAME="$(cat metadata/build_pipeline_name)"
BUILD_JOB_NAME="$(cat metadata/build_job_name)"
BUILD_NAME="$(cat metadata/build_name)"
BUILD_ID="$(cat metadata/build_id)"

export RESLOCK_GITHUB_REPO_OWNER=instana
export RESLOCK_LOCK_OWNER="${BUILD_PIPELINE_NAME}/${BUILD_JOB_NAME}/${BUILD_NAME}/${BUILD_ID}"

echo "RESLOCK_GITHUB_REPO_OWNER=${RESLOCK_GITHUB_REPO_OWNER}"
echo "RESLOCK_LOCK_OWNER=${RESLOCK_LOCK_OWNER}"

curl -s "https://${RESLOCK_GITHUB_TOKEN}@raw.github.ibm.com/instana/reslock/main/run.sh" > run.sh
if [ "${RESLOCK_COMMAND}" == "claim" ]; then
    bash run.sh "${RESLOCK_COMMAND}" k8s-clusters "${RESLOCK_RESOURCE_NAME}" -t 30m
else
    bash run.sh "${RESLOCK_COMMAND}" k8s-clusters "${RESLOCK_RESOURCE_NAME}"
fi
