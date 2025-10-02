#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail
# note: PIPELINE_CONFIG_REPO_PATH will point to config, not to the app folder with the current branch, use APP_REPO_FOLDER instead
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
    trap env EXIT
    env
    set -x
fi
echo "===== e2e-kind.sh - start ====="
echo "CLUSTER_ID=${CLUSTER_ID}, TASK_NAME=${TASK_NAME}"
cd "${WORKSPACE}/${APP_REPO_FOLDER}"
pwd
if [[ $(get_env run-"${CLUSTER_ID}") == "false" ]]; then
    echo "skipping tests due to run-${CLUSTER_ID} being false"
    exit 0
fi

CLUSTER_NAME=kind-pipeline
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
ARTIFACTORY_USERNAME=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".username")
ARTIFACTORY_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")

export ARTIFACTORY_USERNAME ARTIFACTORY_PASSWORD
# required in e2e test, therefore exporting variable
export CLUSTER_NAME

echo "Setting SKIP_INSTALL_GCLOUD=true"
# shellcheck disable=SC2034
SKIP_INSTALL_GCLOUD=true # used in setup.sh
# shellcheck disable=SC1090
source "${WORKSPACE}/${APP_REPO_FOLDER}/ci/sps-scripts/setup.sh"
make generate
go install

export SOURCE_DIRECTORY="${WORKSPACE}/${APP_REPO_FOLDER}"

cd "${SOURCE_DIRECTORY}"

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


# Initialize COMMIT_STATUS with default value
COMMIT_STATUS="failure"
E2E_EXIT_CODE=0

echo "=== Running e2e-kind tests ==="

# Run the e2e-kind tests
if ! make e2e-kind; then
    echo "E2E-kind tests failed"
    E2E_EXIT_CODE=1
else
    COMMIT_STATUS="success"
fi

# Ensure that cluster is released after a successful claim, even if tests fail
cleanup() {
    set-commit-status \
        --repository "$(load_repo app-repo url)" \
        --commit-sha "$(load_repo app-repo commit)" \
        --state "${COMMIT_STATUS}" \
        --description "Kubernetes e2e test" \
        --context "tekton/e2e-${CLUSTER_ID}" \
        --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
    if [[ "$(get_env pipeline_namespace)" == *"pr"* ]]; then
        set-commit-status \
            --repository "$(load_repo app-repo url)" \
            --commit-sha "$(load_repo app-repo commit)" \
            --state "${COMMIT_STATUS}" \
            --description "Kubernetes e2e test" \
            --context "tekton/e2e-${CLUSTER_ID}" \
            --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/unit-test\?}"
    else
        set-commit-status \
            --repository "$(load_repo app-repo url)" \
            --commit-sha "$(load_repo app-repo commit)" \
            --state "${COMMIT_STATUS}" \
            --description "Kubernetes e2e test" \
            --context "tekton/e2e-${CLUSTER_ID}" \
            --target-url "${PIPELINE_RUN_URL//\?/\/$TASK_NAME\/deploy\?}"
    fi
    echo "===== e2e.sh - end ====="

    # Exit with the same code as the e2e tests
    if [ $E2E_EXIT_CODE -ne 0 ]; then
        echo "Exiting with failure due to e2e test failure"
        exit $E2E_EXIT_CODE
    fi
}

trap cleanup EXIT

echo