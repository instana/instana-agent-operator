#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

POD_WAIT_TIME_OUT=180         # s  Pod-check max waiting time
POD_WAIT_INTERVAL=5           # s  Pod-check interval time
OPERATOR_LOG_LINE='Agent installed/upgraded successfully'

# Wait for a pod to be running
# It uses the global variables:
# POD_WAIT_TIME_OUT, POD_WAIT_INTERVAL
# Takes label as a first arg and a second arg is deployment
function wait_for_running_pod() {
    timeout=0
    status=0
    namespace="instana-agent"
    label=${1}
    deployment=${2}

    status=$(kubectl get pod -n "${namespace}" -l="${label}" -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}' | uniq)
    echo "The status of pods from deployment ${deployment} in namespace ${namespace} is: \"$status\""
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ "${#status[@]}" -eq "1" && "${status[0]}" == "Running" ]]; then
            echo "The status of pods from deployment ${deployment} in namespace ${namespace} is: \"$status\". Ending waiting
            loop here."
            break
        fi
        status=$(kubectl get pod -n "${namespace}" -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}'| uniq)
        echo "DEBUG, the status of pods from deployment ${deployment} in namespace ${namespace} is: \"$status\""
        ((timeout+=POD_WAIT_INTERVAL))
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
    local label="app.kubernetes.io/name=instana-agent-operator"

    #Workaround as grep will return -1 if the line is not found.
    #With pipefail enabled, this would fail the script if the if statement omitted.
    if ! crd_installed_successfully=$(kubectl logs -l=${label} -n ${namespace} --tail=-1 | grep "${OPERATOR_LOG_LINE}"); then
        crd_installed_successfully=""
    fi
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ -n "${crd_installed_successfully}" ]]; then
            echo "The agent has been installed/upgraded successfully. Ending waiting loop here."
            break
        fi
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
        #Workaround as grep will return -1 if the line is not found.
        #With pipefail enabled, this would fail the script if the if statement omitted.
        if ! crd_installed_successfully=$(kubectl logs -l=${label} -n ${namespace} --tail=-1 | grep "${OPERATOR_LOG_LINE}"); then
            crd_installed_successfully=""
        fi
    done
    if [[ "${timeout}" -gt "${POD_WAIT_TIME_OUT}" ]]; then
        echo "Agent failed to be installed/upgraded successfully. Exceeded timeout of ${POD_WAIT_TIME_OUT} s. Exit here"
        exit 1
    fi
    return 0;
}

function install_crd() {
    # install the CRD
    echo "Contruct CRD with the agent key, zone, port, and the host"
    path_to_crd="config/samples/instana_v1_instanaagent.yaml"
    yq eval -i '.spec.zone.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.cluster.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.agent.key = env(INSTANA_API_KEY)' ${path_to_crd}
    yq eval -i '.spec.agent.endpointPort = strenv(INSTANA_ENDPOINT_PORT)' ${path_to_crd}
    yq eval -i '.spec.agent.endpointHost = env(INSTANA_ENDPOINT_HOST)' ${path_to_crd}

    echo "Install the CRD"
    kubectl apply -f ${path_to_crd}
}

source pipeline-source/ci/scripts/cluster-authentication.sh

echo "Deploying the public operator"
wget https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml
kubectl apply -f instana-agent-operator.yaml
echo "Verify that the controller manager pods are running"
wait_for_running_pod app.kubernetes.io/name=instana-agent-operator controller-manager

pushd pipeline-source
    install_crd
    echo "Verify that the agent pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    wait_for_successfull_agent_installation

    # upgrade the operator
    echo "Deploying the operator from feature branch"
    IMG="$(cat ../agent-operator-image-amd64/repository):${BUILD_BRANCH}"
    export IMG
    echo "Create secret for $IMG"
    kubectl create secret -n instana-agent docker-registry delivery.instana \
        --docker-server=delivery.instana.io \
        --docker-username="$ARTIFACTORY_USERNAME" \
        --docker-password="$ARTIFACTORY_PASSWORD"

    make install deploy

    echo "Add imagePullSecrets to the controller-manager deployment"
    kubectl patch deployment controller-manager -n instana-agent -p '"spec": { "template" : {"spec": { "imagePullSecrets": [{"name": "delivery.instana"}]}}}'
    wait_for_running_pod app.kubernetes.io/name=instana-agent-operator controller-manager

    echo "Verify that the agent pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    wait_for_successfull_agent_installation
    echo "Upgrade has been successful"
popd
