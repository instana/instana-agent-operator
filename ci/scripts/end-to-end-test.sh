#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -x #TODO: remove before merging
set -e
set -o pipefail

echo "running e2e tests"

POD_WAIT_TIME_OUT=120         # s  Pod-check max waiting time
POD_WAIT_INTERVAL=1           # s  Pod-check interval time

function get_public_image() {
    git pull -r
    VERSION=$(git tag | sort -r --version-sort | head -n1)
    echo "Latest release is ${VERSION}"
    export PREFIX="v"
    export OPERATOR_DOCKER_VERSION=${VERSION#"$PREFIX"}

    export IMG="icr.io/instana/instana-agent-operator:$OPERATOR_DOCKER_VERSION"
    echo "IMG=$IMG"
}

# Wait for a pod to be running
# It uses the global variables:
# POD_WAIT_TIME_OUT, POD_WAIT_INTERVAL
# Takes namespace as first arg and label as a second
function wait_for_running_pod() {
    local timeout=0
    local status=0
    set +u
    status=($(kubectl get pod -n ${1} -l=${2} -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}' | uniq))
    echo "DEBUG, the pod in namespace ${1} status is: \"$status\""
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ "${#status[@]}" -eq "1" && "${status[0]}" == "Running" ]]; then
            echo "The pod in namespace ${1} status is: \"$status\". Ending waiting
            loop here."
            break
        fi
        status=($(kubectl get pod -n ${1} -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}'| uniq))
        echo "DEBUG, the pod in namespace ${1} status is: \"$status\""
        ((timeout+=$POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
    done
    if [[ "${timeout}" -gt "${POD_WAIT_TIME_OUT}" ]]; then
        echo "${1} failed to initialize. Exceeded timeout of
        ${POD_WAIT_TIME_OUT} s. Exit here"
        exit 1
    fi
    set -u
    return 0;
}

source pipeline-source/ci/scripts/cluster-authentication.sh

## deploy operator from main branch
pushd pipeline-source
    get_public_image

    make install
    make deploy

    wait_for_running_pod instana-agent app.kubernetes.io/name=instana-agent-operator

popd

# check if the operator and agent pods are running

## upgrade to the operator from the release branch
# check if the operator and agent pods are running
