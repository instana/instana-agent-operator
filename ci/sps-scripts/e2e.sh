#!/usr/bin/env bash
set -euo pipefail
echo "===== e2e.sh - start ====="
pwd
export SOURCE_DIRECTORY="${WORKSPACE}/${APP_REPO_FOLDER}"
echo "${SOURCE_DIRECTORY}"
ls "${SOURCE_DIRECTORY}"
CLUSTER_ID=$1
echo "Running e2e chart tests for ${CLUSTER_ID}"
TASK_NAME=$2

if [[ "$(get_env pipeline_namespace)" == *"pr"* ]]; then
    set-commit-status \
        --repository "$(load_repo app-repo url)" \
        --commit-sha "$(load_repo app-repo commit)" \
        --state "pending" \
        --description "OCP e2e test" \
        --context "tekton/e2e-${CLUSTER_ID}" \
        --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
else
    set-commit-status \
        --repository "$(load_repo app-repo url)" \
        --commit-sha "$(load_repo app-repo commit)" \
        --state "pending" \
        --description "OCP e2e test" \
        --context "tekton/e2e-${CLUSTER_ID}" \
        --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/deploy\?}"
fi

COMMIT_STATUS="error"

CLUSTER_DETAILS=$(get_env "${CLUSTER_ID}")
CLUSTER_TYPE=$(echo "${CLUSTER_DETAILS}" | jq -r ".type")
CLUSTER_NAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".name")
# required in e2e test
export CLUSTER_NAME

if [ "${CLUSTER_TYPE}" == "fyre-ocp" ]; then
    echo "Fyre OCP Cluster detected"
    CLUSTER_SERVER=$(echo "${CLUSTER_DETAILS}" | jq -r ".server")
    CLUSTER_USERNAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".username")
    CLUSTER_PASSWORD=$(echo "${CLUSTER_DETAILS}" | jq -r ".password")
    mkdir -p bin
    cd bin
    # late install, as the oc cli is fetched from the target cluster to ensure proper versions
    echo "=== Installing oc cli ==="
    echo "trying to download oc from https://downloads-openshift-console.apps.${CLUSTER_NAME}.cp.fyre.ibm.com/amd64/linux/oc.tar"
    curl -sk "https://downloads-openshift-console.apps.${CLUSTER_NAME}.cp.fyre.ibm.com/amd64/linux/oc.tar" -o oc.tar
    ls -lah oc.tar
    tar -xf oc.tar
    rm -f oc.tar

    PATH=$(pwd):$PATH
    export PATH

    # ensure that debug will not print cluster credentials to the log
    if [[ "$PIPELINE_DEBUG" == 1 ]]; then
        set +x
    fi

    echo "Logging into ${CLUSTER_SERVER}"
    oc login --insecure-skip-tls-verify=true -u "${CLUSTER_USERNAME}" -p "${CLUSTER_PASSWORD}" --server="${CLUSTER_SERVER}"

    if [[ "$PIPELINE_DEBUG" == 1 ]]; then
        set -x
    fi

elif [ "${CLUSTER_TYPE}" == "gke" ]; then
    echo "GKE Cluster detected"
    CLUSTER_ZONE=$(echo "${CLUSTER_DETAILS}" | jq -r ".zone")
    CLUSTER_PROJECT=$(echo "${CLUSTER_DETAILS}" | jq -r ".project")
    # login into GCP
    get_env gcp-service-account > keyfile.json
    gcloud auth activate-service-account --key-file keyfile.json
    gcloud container clusters get-credentials "${CLUSTER_NAME}" --zone "${CLUSTER_ZONE}" --project "${CLUSTER_PROJECT}"
else
    echo "Unknown cluster type, failing build as it is unclear how to connect to the cluster"
    exit 1
fi


cd "${SOURCE_DIRECTORY}"
echo "Showing connected cluster nodes"
kubectl get nodes -o wide

export PATH=$PATH:/usr/local/go/bin:/usr/local/bin
go version

# fetching e2e test backend details
INSTANA_E2E_BACKEND_DETAILS=$(get_env instana-e2e-backend-details)
INSTANA_ENDPOINT_HOST=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".endpoint_host")
INSTANA_ENDPOINT_PORT=443
INSTANA_AGENT_KEY=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".agent_key")
INSTANA_API_URL=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_url")
INSTANA_API_TOKEN=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_token")

export INSTANA_ENDPOINT_HOST INSTANA_ENDPOINT_PORT INSTANA_AGENT_KEY INSTANA_API_URL INSTANA_API_TOKEN

echo "=== Claim cluster lock ==="

bash "${SOURCE_DIRECTORY}/ci/sps/reslock.sh" claim "${CLUSTER_ID}"

# Ensure that cluster is released after a successful claim, even if tests fail
cleanup() {
    set-commit-status \
        --repository "$(load_repo app-repo url)" \
        --commit-sha "$(load_repo app-repo commit)" \
        --state "${COMMIT_STATUS}" \
        --description "OCP e2e test" \
        --context "tekton/e2e-${CLUSTER_ID}" \
        --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
    if [[ "$(get_env pipeline_namespace)" == *"pr"* ]]; then
        set-commit-status \
            --repository "$(load_repo app-repo url)" \
            --commit-sha "$(load_repo app-repo commit)" \
            --state "${COMMIT_STATUS}" \
            --description "OCP e2e test" \
            --context "tekton/e2e-${CLUSTER_ID}" \
            --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
    else
        set-commit-status \
            --repository "$(load_repo app-repo url)" \
            --commit-sha "$(load_repo app-repo commit)" \
            --state "${COMMIT_STATUS}" \
            --description "OCP e2e test" \
            --context "tekton/e2e-${CLUSTER_ID}" \
            --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/deploy\?}"
    fi
    bash "${SOURCE_DIRECTORY}/ci/sps/reslock.sh" release "${CLUSTER_ID}"
    echo "===== e2e.sh - end ====="
}

trap cleanup EXIT
echo

echo "=== Showing versions ==="
oc version
kubectl version
helm version
echo

echo "Showing available helm charts"
ls -lah artefacts

echo "=== Running e2e tests ==="
INSTANA_AGENT_HELM_CHART_LOCATION="$(ls artefacts/instana-agent-*.tgz)"
export INSTANA_AGENT_HELM_CHART_LOCATION
echo "INSTANA_AGENT_HELM_CHART_LOCATION=$INSTANA_AGENT_HELM_CHART_LOCATION"
helm version
make e2e
echo "Tests finished"
# trap handler will automatically report status
COMMIT_STATUS="success"