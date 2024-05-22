#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -x #TODO: remove before merging
set -e
set -o pipefail

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
# Takes namespace as first arg and label as a second and a third arg is deployment
function wait_for_running_pod() {
    local timeout=0
    local status=0
    local namespace=${1}
    local label=${2}
    local deployment=${3}

    status=($(kubectl get pod -n ${namespace} -l=${label} -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}' | uniq))
    echo "The status of pods from deployment ${deployment} in namespace ${namespace} is: \"$status\""
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ "${#status[@]}" -eq "1" && "${status[0]}" == "Running" ]]; then
            echo "The status of pods from deployment ${deployment} in namespace ${namespace} is: \"$status\". Ending waiting
            loop here."
            break
        fi
        status=($(kubectl get pod -n ${namespace} -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}'| uniq))
        echo "DEBUG, the status of pods from deployment ${deployment} in namespace ${namespace} is: \"$status\""
        ((timeout+=$POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
    done
    if [[ "${timeout}" -gt "${POD_WAIT_TIME_OUT}" ]]; then
        echo "${namespace} failed to initialize. Exceeded timeout of
        ${POD_WAIT_TIME_OUT} s. Exit here"
        exit 1
    fi
    set -u
    return 0;
}

# Checks if one of the controller-manager pods logged successful installation
function wait_for_successfull_agent_installation() {
    local timeout=0
    local namespace=${1}
    local label=${2}
    local deployment=${3}

    crd_installed_successfully=($(kubectl logs -l=${label} -n ${namespace} --tail=-1 | grep "Agent installed/upgraded successfully"))
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ -n "${crd_installed_successfully}" ]]; then
            echo "The agent has been installed/upgraded successfully. Ending waiting loop here."
            break
        fi
        ((timeout+=$POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
        crd_installed_successfully=($(kubectl logs -l=${label} -n ${namespace} --tail=-1 | grep "Agent installed/upgraded successfully"))
    done
    if [[ "${timeout}" -gt "${POD_WAIT_TIME_OUT}" ]]; then
        echo "Agent failed to be installed/upgraded successfully. Exceeded timeout of ${POD_WAIT_TIME_OUT} s. Exit here"
        exit 1
    fi
    return 0;
}

source pipeline-source/ci/scripts/cluster-authentication.sh

## deploy operator from main branch
pushd pipeline-source
    get_public_image

    make install
    make deploy

    echo "Verify that the controller manager pods are running"
    wait_for_running_pod instana-agent app.kubernetes.io/name=instana-agent-operator controller-manager

    # install the CRD
    echo "Contruct CRD with the agent key, zone, port, and the host"
    local path_to_crd="config/samples/instana_v1_instanaagent.yaml"
    yq w -i ${path_to_crd} 'spec.zone.name' "${NAME}"
    yq w -i ${path_to_crd} 'spec.cluster.name' "${NAME}"
    yq w -i ${path_to_crd} 'spec.agent.key' "${INSTANA_API_KEY}"
    yq w -i ${path_to_crd} 'spec.agent.endpointPort' "${INSTANA_ENDPOINT_PORT}"
    yq w -i ${path_to_crd} 'spec.agent.endpointHost' "${INSTANA_ENDPOINT_HOST}"

    echo "Install the CRD"
    kubectl apply -f ${path_to_crd}
    echo "Verify that the agent pods are running"
    wait_for_running_pod instana-agent app.kubernetes.io/name=instana-agent instana-agent



    # upgrade the operator
    # verifications

popd

# check if the operator and agent pods are running

## upgrade to the operator from the release branch
# check if the operator and agent pods are running
