#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

POD_WAIT_TIME_OUT=120         # s  Pod-check max waiting time
POD_WAIT_INTERVAL=1           # s  Pod-check interval time
OPERATOR_LOG_LINE="Agent installed/upgraded successfully"

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
# Takes label as a first arg and a second arg is deployment
function wait_for_running_pod() {
    local timeout=0
    local status=0
    local namespace="instana-agent"
    local label=${1}
    local deployment=${2}

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
    return 0;
}

# Checks if one of the controller-manager pods logged successful installation
function wait_for_successfull_agent_installation() {
    local timeout=0
    local namespace="instana-agent"
    local label=${1}

    crd_installed_successfully=($(kubectl logs -l=${label} -n ${namespace} --tail=-1 | grep ${OPERATOR_LOG_LINE}))
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ -n "${crd_installed_successfully}" ]]; then
            echo "The agent has been installed/upgraded successfully. Ending waiting loop here."
            break
        fi
        ((timeout+=$POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
        crd_installed_successfully=($(kubectl logs -l=${label} -n ${namespace} --tail=-1 | grep ${OPERATOR_LOG_LINE}))
    done
    if [[ "${timeout}" -gt "${POD_WAIT_TIME_OUT}" ]]; then
        echo "Agent failed to be installed/upgraded successfully. Exceeded timeout of ${POD_WAIT_TIME_OUT} s. Exit here"
        exit 1
    fi
    return 0;
}

source pipeline-source/ci/scripts/cluster-authentication.sh

## deploy operator from main branch
pushd operator-git-main
    echo "Deploying the public operator"
    get_public_image

    make install deploy

    echo "Verify that the controller manager pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent-operator controller-manager

    # install the CRD
    echo "Contruct CRD with the agent key, zone, port, and the host"
    path_to_crd="config/samples/instana_v1_instanaagent.yaml"
    yq eval -i '.spec.zone.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.cluster.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.agent.key = env(INSTANA_API_KEY)' ${path_to_crd}
    yq eval -i '.spec.agent.endpointPort = env(toString(INSTANA_ENDPOINT_PORT))' ${path_to_crd}
    yq eval -i '.spec.agent.endpointHost = env(INSTANA_ENDPOINT_HOST)' ${path_to_crd}

    echo "Install the CRD"
    kubectl apply -f ${path_to_crd}
    echo "Verify that the agent pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    wait_for_successfull_agent_installation app.kubernetes.io/name=instana-agent-operator
popd


pushd pipeline-source
    # upgrade the operator
    echo "Deploying the operator from feature branch"
    IMG=$(cat ../agent-operator-bundle-image/repository):${BUILD_BRANCH}
    make install deploy
    # install the CRD
    echo "Contruct CRD with the agent key, zone, port, and the host"
    path_to_crd="config/samples/instana_v1_instanaagent.yaml"
    yq eval -i '.spec.zone.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.cluster.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.agent.key = env(INSTANA_API_KEY)' ${path_to_crd}
    yq eval -i '.spec.agent.endpointPort = env(toString(INSTANA_ENDPOINT_PORT))' ${path_to_crd}
    yq eval -i '.spec.agent.endpointHost = env(INSTANA_ENDPOINT_HOST)' ${path_to_crd}

    echo "Install the CRD"
    kubectl apply -f ${path_to_crd}
    echo "Verify that the agent pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    wait_for_successfull_agent_installation app.kubernetes.io/name=instana-agent-operator
    echo "Upgrade has been successful"
popd
