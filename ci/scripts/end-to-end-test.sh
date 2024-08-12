#!/bin/bash

#
# (c) Copyright IBM Corp. 2024
# (c) Copyright Instana Inc.
#

set -e
set -o pipefail

POD_WAIT_TIME_OUT=120         # s  Pod-check max waiting time
POD_WAIT_INTERVAL=5           # s  Pod-check interval time
OPERATOR_LOG_LINE='Agent installed/upgraded successfully'
OPERATOR_LOG_LINE_NEW='successfully finished reconcile on agent CR'
NAMESPACE="instana-agent"

# Wait for a pod to be running
# It uses the global variables:
# POD_WAIT_TIME_OUT, POD_WAIT_INTERVAL
# Takes label as a first arg and a second arg is deployment
function wait_for_running_pod() {
    echo "=== wait_for_running_pod ==="
    timeout=0
    status=0
    label=${1}
    deployment=${2}
    pods_are_running=false

    echo "Showing running pods"
    kubectl get pods -n "${NAMESPACE}"
    status=$(kubectl get pod -n "${NAMESPACE}" -l="${label}" -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}' | uniq)
    echo "The status of pods from deployment ${deployment} in namespace ${NAMESPACE} is: \"$status\""
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ "${#status[@]}" -eq "1" && "${status[0]}" == "Running" ]]; then
            echo "The status of pods from deployment ${deployment} in namespace ${NAMESPACE} is: \"$status\". Ending waiting
            loop here."
            pods_are_running=true
            break
        fi
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
        echo "Showing running pods"
        kubectl get pods -n "${NAMESPACE}"
        status=$(kubectl get pod -n "${NAMESPACE}" -o go-template='{{ range .items }}{{ println .status.phase }}{{ end }}'| uniq)
        echo "DEBUG, the status of pods from deployment ${deployment} in namespace ${NAMESPACE} is: \"$status\""
    done
    if [[ "${pods_are_running}" == "false" ]]; then
        echo "${NAMESPACE} failed to initialize. Exceeded timeout of
        ${POD_WAIT_TIME_OUT} s. Exit here"
        echo "Showing running pods"
        kubectl get pods -n "${NAMESPACE}"
        exit 1
    fi
    return 0;
}

# Checks if one of the controller-manager pods logged successful installation
function wait_for_successfull_agent_installation() {
    echo "=== wait_for_successfull_agent_installation ==="
    local timeout=0
    local label="app.kubernetes.io/name=instana-agent-operator"
    local agent_found=false

    #Workaround as grep will return -1 if the line is not found.
    #With pipefail enabled, this would fail the script if the if statement omitted.
    if ! crd_installed_successfully=$(kubectl logs -l=${label} -n ${NAMESPACE} --tail=-1 | grep "${OPERATOR_LOG_LINE}"); then
        # Try to fetch the new log line if the old one is not there
        if ! crd_installed_successfully=$(kubectl logs -l=${label} -n ${NAMESPACE} --tail=-1 | grep "${OPERATOR_LOG_LINE_NEW}"); then
            crd_installed_successfully=""
        fi
    fi
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        if [[ -n "${crd_installed_successfully}" ]]; then
            echo "The agent has been installed/upgraded successfully. Ending waiting loop here."
            agent_found=true
            break
        fi
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
        #Workaround as grep will return -1 if the line is not found.
        #With pipefail enabled, this would fail the script if the if statement omitted.
        if ! crd_installed_successfully=$(kubectl logs -l=${label} -n ${NAMESPACE} --tail=-1 | grep "${OPERATOR_LOG_LINE}"); then
            # Try to fetch the new log line if the old one is not there
            if ! crd_installed_successfully=$(kubectl logs -l=${label} -n ${NAMESPACE} --tail=-1 | grep "${OPERATOR_LOG_LINE_NEW}"); then
                crd_installed_successfully=""
            fi
        fi
    done
    if [[ "${agent_found}" == "false" ]]; then
        echo "Agent failed to be installed/upgraded successfully. Exceeded timeout of ${POD_WAIT_TIME_OUT} s. Exit here"
        exit 1
    fi

    return 0;
}

function ensure_new_operator_deployment() {
    echo "=== ensure_new_operator_deployment ==="
    local timeout=0

    echo "Scaling controller-manager deployment down to zero"
    kubectl scale -n ${NAMESPACE} --replicas=0 deployment/controller-manager
    set +e
    local operator_present=true
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        echo "Showing pods"
        kubectl get -n ${NAMESPACE} pods
        controller_manager_gone=$(kubectl get -n ${NAMESPACE} pods | grep controller-manager)
        if [ "$controller_manager_gone" == "" ]; then
            echo "Operator pods are gone"
            operator_present=false
            break
        else
            echo "Operator pods are still present"
        fi
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
    done

    echo "=== Operator logs start ==="
    kubectl logs -n ${NAMESPACE} -l "app.kubernetes.io/name=instana-agent-operator"
    echo "=== Operator logs end ==="
    echo

    if [[ "${operator_present}" == "true" ]]; then
        echo "Failed to scale operator to 0 instance. Exceeded timeout of ${POD_WAIT_TIME_OUT} s. Exit here"
        echo "Showing running pods"
        kubectl get pods -n "${NAMESPACE}"
        exit 1
    fi

    set -e

    echo "Scaling operator deployment to 1 instance"
    kubectl scale -n ${NAMESPACE} --replicas=1 deployment/controller-manager

    set +e
    timeout=0
    operator_present=false
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        echo "Showing pods"
        kubectl get -n ${NAMESPACE} pods
        controller_manager_present=$(kubectl get -n ${NAMESPACE} pods | grep "controller-manager" | grep "Running" | grep "1/1")
        if [ "$controller_manager_present" == "" ]; then
            echo "Operator pod is not running yet"
        else
            echo "Operator pod is running now"
            operator_present=true
            break
        fi
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
    done
    set -e

    echo "=== Operator logs start ==="
    kubectl logs -n ${NAMESPACE} -l "app.kubernetes.io/name=instana-agent-operator"
    echo "=== Operator logs end ==="
    echo

    if [[ "${operator_present}" == "false" ]]; then
        echo "Failed to scale operator to 1 instance. Exceeded timeout of ${POD_WAIT_TIME_OUT} s. Exit here"
        echo "Showing running pods"
        kubectl get pods -n "${NAMESPACE}"
        exit 1
    fi
}

function wait_for_running_cr_state() {
    echo "=== wait_for_running_cr_state ==="
    local timeout=0
    local cr_status="Failed"

    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        cr_status=$(kubectl -n ${NAMESPACE} get agent instana-agent -o yaml | yq .status.status)
        echo "CR state: ${cr_status}"
        if [[ "${cr_status}" == "Running" ]]; then
            echo "The custom resource reflects the Running state correctly. Ending waiting loop here."
            break
        fi
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
    done

    if [[ "${cr_status}" != "Running" ]]; then
        echo "The custom resource did not reflect the Running state correctly."
        echo "Displaying state found on the CR"
        kubectl -n ${NAMESPACE} get agent instana-agent -o yaml | yq .status
        exit 1
    fi
}

function install_cr() {
    echo "=== install_cr ==="
    # install the Custom Resource
    echo "Contruct CR with the agent key, zone, port, and the host"
    path_to_crd="config/samples/instana_v1_instanaagent.yaml"
    yq eval -i '.spec.zone.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.cluster.name = env(NAME)' ${path_to_crd}
    yq eval -i '.spec.agent.key = env(INSTANA_API_KEY)' ${path_to_crd}
    yq eval -i '.spec.agent.endpointPort = strenv(INSTANA_ENDPOINT_PORT)' ${path_to_crd}
    yq eval -i '.spec.agent.endpointHost = env(INSTANA_ENDPOINT_HOST)' ${path_to_crd}

    echo "Install the CR"
    kubectl apply -f ${path_to_crd}
}

function install_cr_multi_backend_external_keyssecret() {
    echo "=== install_cr_multi_backend_external_keyssecret ==="

    # install the Custom Resource
    path_to_crd="config/samples/instana_v1_instanaagent_multiple_backends_external_keyssecret.yaml"
    path_to_keyssecret="config/samples/external_secret_instana_agent_key.yaml"

    echo "Install the keysSecret and CR"
    # credentials are invalid here, but that's okay, we just test the operator behavior, not the agent
    kubectl apply -f ${path_to_keyssecret}
    kubectl apply -f ${path_to_crd}
}

function verify_multi_backend_config_generation_and_injection() {
    echo "=== function verify_multi_backend_config_generation_and_injection ==="
    local timeout=0

    echo "Checking if instana-agent-config secret is present with 2 backends"
    kubectl get secret -n ${NAMESPACE} instana-agent-config -o yaml
    kubectl get secret -n ${NAMESPACE} instana-agent-config -o yaml | yq '.data["com.instana.agent.main.sender.Backend-1.cfg"]' | base64 -d > backend.cfg
    echo "Validate backend config structure for backend 1"
    grep "host=first-backend.instana.io" backend.cfg
    grep "port=443" backend.cfg
    grep "protocol=HTTP/2" backend.cfg
    # check for key, safe to log as just a dummy value
    grep "key=xxx" backend.cfg

    kubectl get secret -n ${NAMESPACE} instana-agent-config -o yaml | yq '.data["com.instana.agent.main.sender.Backend-2.cfg"]' | base64 -d > backend.cfg
    echo "Validate backend config structure for backend 2"
    grep "host=second-backend.instana.io" backend.cfg
    grep "port=443" backend.cfg
    grep "protocol=HTTP/2" backend.cfg
    # check for key, safe to log as just a dummy value
    grep "key=yyy" backend.cfg

    echo "Validate that backend config files are available inside the agent pod"
    echo "Getting pod name for exec"
    pod_name=$(kubectl get pods -n ${NAMESPACE} -l app.kubernetes.io/component=instana-agent -o yaml  | yq ".items[0].metadata.name")

    exec_successful=false
    while [[ "${timeout}" -le "${POD_WAIT_TIME_OUT}" ]]; do
        set +e
        echo "Exec into pod ${pod_name} and see if etc/instana/com.instana.agent.main.sender.Backend-2.cfg is present"

        if kubectl exec -n ${NAMESPACE} "${pod_name}" -- cat /opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-2.cfg; then
            echo "Could cat file"
            exec_successful=true
            break
        fi
        set -e
        ((timeout+=POD_WAIT_INTERVAL))
        sleep $POD_WAIT_INTERVAL
        echo "Getting pod name for exec"
        pod_name=$(kubectl get pods -n ${NAMESPACE} -l app.kubernetes.io/component=instana-agent -o yaml  | yq ".items[0].metadata.name")
    done

    if [[ "${exec_successful}" == "false" ]]; then
        echo "Failed to cat file, check if the symlink logic in the entrypoint script of the agent container image is correct"
        echo "Showing running pods"
        kubectl get pods -n "${NAMESPACE}"
        exit 1
    fi

    echo "Check if the right backend was mounted in Backend-1.cfg"
    echo "Exec into pod ${pod_name} and see if etc/instana/com.instana.agent.main.sender.Backend-1.cfg is present"
    kubectl exec -n ${NAMESPACE} "${pod_name}" -- cat /opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-1.cfg
    kubectl exec -n ${NAMESPACE} "${pod_name}" -- cat /opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-1.cfg | grep "host=first-backend.instana.io"

    echo "Check if the right backend was mounted in Backend-2.cfg"
    kubectl exec -n ${NAMESPACE}  "${pod_name}" -- cat /opt/instana/agent/etc/instana/com.instana.agent.main.sender.Backend-2.cfg | grep "host=second-backend.instana.io"
    kubectl -n ${NAMESPACE} get agent instana-agent -o yaml
}

source pipeline-source/ci/scripts/cluster-authentication.sh

echo "Deploying the public operator"
wget https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml
kubectl apply -f instana-agent-operator.yaml
echo "Verify that the controller manager pods are running"
wait_for_running_pod app.kubernetes.io/name=instana-agent-operator controller-manager

pushd pipeline-source
    install_cr
    echo "Verify that the agent pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    wait_for_successfull_agent_installation

    # upgrade the operator
    echo "Deploying the operator from feature branch"
    IMG="delivery.instana.io/int-docker-agent-local/instana-agent-operator/dev-build:${GIT_COMMIT}"
    export IMG
    echo "Create secret for $IMG"
    kubectl create secret -n instana-agent docker-registry delivery.instana \
        --docker-server=delivery.instana.io \
        --docker-username="$ARTIFACTORY_USERNAME" \
        --docker-password="$ARTIFACTORY_PASSWORD"

    make install deploy

    echo "Add imagePullSecrets to the controller-manager deployment"
    kubectl patch deployment controller-manager -n instana-agent -p '"spec": { "template" : {"spec": { "imagePullSecrets": [{"name": "delivery.instana"}]}}}'
    ensure_new_operator_deployment
    wait_for_running_pod app.kubernetes.io/name=instana-agent-operator controller-manager

    echo "Verify that the agent pods are running"
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    wait_for_successfull_agent_installation
    wait_for_running_cr_state
    echo "Upgrade has been successful"

    echo "Install CR to connect to an additional backend with external keysSecret"
    install_cr_multi_backend_external_keyssecret
    wait_for_running_pod app.kubernetes.io/name=instana-agent instana-agent
    verify_multi_backend_config_generation_and_injection
popd
