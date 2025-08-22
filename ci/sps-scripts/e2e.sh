#!/usr/bin/env bash
set -euo pipefail
# note: PIPELINE_CONFIG_REPO_PATH will point to config, not to the app folder with the current branch, use APP_REPO_FOLDER instead
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
    trap env EXIT
    env
    set -x
fi
echo "===== e2e.sh - start ====="

echo "CLUSTER_ID=${CLUSTER_ID}, TASK_NAME=${TASK_NAME}"
cd "${WORKSPACE}/${APP_REPO_FOLDER}"
pwd
if [[ $(get_env run-"${CLUSTER_ID}") == "false" ]]; then
    echo "skipping tests due to run-${CLUSTER_ID} being false"
    exit 0
fi

CLUSTER_DETAILS=$(get_env "${CLUSTER_ID}")
CLUSTER_TYPE=$(echo "${CLUSTER_DETAILS}" | jq -r ".type")
CLUSTER_NAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".name")
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
export ARTIFACTORY_USERNAME=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".username")
export ARTIFACTORY_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")
# required in e2e test, therefore exporting variable
export CLUSTER_NAME

if [[ "${CLUSTER_TYPE}" == "fyre-ocp" ]]; then
    echo "Setting SKIP_INSTALL_GCLOUD=true, becuase CLUSTER_TYPE is ${CLUSTER_TYPE}"
    # shellcheck disable=SC2034
    SKIP_INSTALL_GCLOUD=true # used in setup.sh
fi

# shellcheck disable=SC1090
source "${WORKSPACE}/${APP_REPO_FOLDER}/ci/sps-scripts/setup.sh"
make generate
go install
make build

export SOURCE_DIRECTORY="${WORKSPACE}/${APP_REPO_FOLDER}"

if [ "${CLUSTER_TYPE}" == "fyre-ocp" ]; then
    echo "Fyre OCP Cluster detected"
    CLUSTER_SERVER=$(echo "${CLUSTER_DETAILS}" | jq -r ".server")
    CLUSTER_USERNAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".username")
    CLUSTER_PASSWORD=$(echo "${CLUSTER_DETAILS}" | jq -r ".password")
    # channel is part of the secret in Secrets Manager and should be e.g. "channel": "stable-4.18"
    CLUSTER_CHANNEL=$(echo "${CLUSTER_DETAILS}" | jq -r ".channel")
    mkdir -p bin
    cd bin
    echo "=== Installing oc cli ==="
    echo "trying to download oc from https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${CLUSTER_CHANNEL}/openshift-client-linux.tar.gz"
    curl -sk "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${CLUSTER_CHANNEL}/openshift-client-linux.tar.gz" -o openshift-client-linux.tar.gz
    ls -lah openshift-client-linux.tar.gz
    tar -xf openshift-client-linux.tar.gz
    rm -f openshift-client-linux.tar.gz README.md

    PATH=$(pwd):${PATH}
    export PATH

    # ensure that debug will not print cluster credentials to the log
    if [[ "${PIPELINE_DEBUG}" == 1 ]]; then
        set +x
    fi

    echo "Logging into ${CLUSTER_SERVER}"
    oc login --insecure-skip-tls-verify=true -u "${CLUSTER_USERNAME}" -p "${CLUSTER_PASSWORD}" --server="${CLUSTER_SERVER}"

    if [[ "${PIPELINE_DEBUG}" == 1 ]]; then
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

export PATH=${PATH}:/usr/local/go/bin:/usr/local/bin
go version

# fetching e2e test backend details
INSTANA_E2E_BACKEND_DETAILS=$(get_env instana-e2e-backend-details)
INSTANA_ENDPOINT_HOST=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".endpoint_host")
INSTANA_ENDPOINT_PORT=443
INSTANA_API_KEY=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".agent_key")
INSTANA_API_URL=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_url")
INSTANA_API_TOKEN=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_token")

export INSTANA_ENDPOINT_HOST INSTANA_ENDPOINT_PORT INSTANA_API_KEY INSTANA_API_URL INSTANA_API_TOKEN

echo "=== Claim cluster lock ==="

bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" claim "${CLUSTER_ID}"

echo "=== Running e2e tests ==="


if ! make e2e; then
    COMMIT_STATUS="success"
fi

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
    bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" release "${CLUSTER_ID}"
    echo "===== e2e.sh - end ====="
}

trap cleanup EXIT

echo