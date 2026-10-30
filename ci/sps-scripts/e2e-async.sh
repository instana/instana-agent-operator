#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Shared async E2E runner for release-build pipeline.
# Parameterised via environment variables set in each stage's script block:
#   CLUSTER_ID  — one of: ocp-fyre-latest, gke-operator-lowest, ocp-fyre-lowest, gke-operator-latest
#   E2E_MODE    — "operator" or "helm"
#
# Behaviour (never exits early — always uploads log + status file):
#   1. Read BOM from COS
#   2. Claim reslock
#   3. Run E2E tests (captures exit code, does not exit immediately on failure)
#   4. Release reslock
#   5. Upload timestamped log file
#   6. Write (overwrite) status pointer JSON
#   7. Post git commit status to operator repo at operator-sha
#   8. Exit non-zero if tests failed
#
set -uo pipefail   # intentionally no -e; we control exit manually
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== e2e-async.sh - start ====="
echo "CLUSTER_ID=${CLUSTER_ID}  E2E_MODE=${E2E_MODE}"

# ---------------------------------------------------------------------------
# Read pipeline environment
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region "us-south")
COS_STATE_PATH=$(get_env cos-state-path)

# ---------------------------------------------------------------------------
# Log in to IBM Cloud (async stages run in a fresh container; the login done
# in prepare.sh is not inherited)
# ---------------------------------------------------------------------------
IBMCLOUD_API_KEY=$(get_secret ibmcloud-api-key)
if ! ibmcloud login --apikey "${IBMCLOUD_API_KEY}" --no-region; then
  echo "ERROR: ibmcloud login failed" >&2
  exit 1
fi
unset IBMCLOUD_API_KEY

# ---------------------------------------------------------------------------
# Read BOM from COS
# ---------------------------------------------------------------------------
# Use a non-pre-existing path — ibmcloud cos get-object prompts interactively
# if the output file already exists (e.g. a path returned by mktemp).
BOM_FILE="${WORKSPACE}/bom-e2e-$$.json"
if ! ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${COS_STATE_PATH}/bom.json" \
  --region "${COS_REGION}" \
  --output json "${BOM_FILE}" || [[ ! -s "${BOM_FILE}" ]]; then
  echo "ERROR: failed to read BOM from COS" >&2
  exit 1
fi

VERSION=$(jq -r '.version'                         "${BOM_FILE}")
OPERATOR_SHA=$(jq -r '.sources."operator-sha"'     "${BOM_FILE}")
CHARTS_SHA=$(jq -r '.sources."charts-sha"'         "${BOM_FILE}")
OPERATOR_MANIFEST_DIGEST=$(jq -r '.images."operator-manifest-digest"' "${BOM_FILE}")
HELM_CHART_COS=$(jq -r '.artifacts."helm-chart"'   "${BOM_FILE}")
SOURCE_BRANCH=$(jq -r '."source-branch"'           "${BOM_FILE}")
BOM_JSON=$(cat "${BOM_FILE}")
rm -f "${BOM_FILE}"

echo "version=${VERSION}  operator-sha=${OPERATOR_SHA}"
echo "charts-sha=${CHARTS_SHA}"
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"

# ---------------------------------------------------------------------------
# Install dependencies
# ---------------------------------------------------------------------------
# The operator repo's installGolang.sh is the single source of truth for the Go
# version, also in helm mode (the charts setup.sh no longer installs Go).
if [[ -f "./installGolang.sh" ]]; then
  source ./installGolang.sh amd64
elif [[ -n "${PIPELINE_CONFIG_REPO_PATH:-}" && -f "${WORKSPACE}/${PIPELINE_CONFIG_REPO_PATH}/installGolang.sh" ]]; then
  source "${WORKSPACE}/${PIPELINE_CONFIG_REPO_PATH}/installGolang.sh" amd64
elif [[ -n "${APP_REPO_FOLDER:-}" && -f "${WORKSPACE}/${APP_REPO_FOLDER}/installGolang.sh" ]]; then
  source "${WORKSPACE}/${APP_REPO_FOLDER}/installGolang.sh" amd64
fi
export PATH=$PATH:/usr/local/go/bin

# ---------------------------------------------------------------------------
# Clone operator repo at frozen SHA (E2E tests live in the operator repo)
# ---------------------------------------------------------------------------
GH_TOKEN=$(get_secret github-token)
GH_TOKEN_FILE=$(mktemp)
echo "${GH_TOKEN}" > "${GH_TOKEN_FILE}"
chmod 600 "${GH_TOKEN_FILE}"

set +u
# shellcheck source=/dev/null
. "${ONE_PIPELINE_PATH}"/git/clone_repo_v2 \
  --repository "https://github.com/instana/instana-agent-operator" \
  --token-path "${GH_TOKEN_FILE}" \
  --branch "${SOURCE_BRANCH}" \
  --depth 1 \
  --directory operator-src \
  --use-lfs false \
  --force-exit false
set -u
git -C "${WORKSPACE}/operator-src" fetch --depth 1 origin "${OPERATOR_SHA}" || exit 1
git -C "${WORKSPACE}/operator-src" checkout --detach "${OPERATOR_SHA}" || exit 1
rm -f "${GH_TOKEN_FILE}"

SOURCE_DIRECTORY="${WORKSPACE}/operator-src"

# ---------------------------------------------------------------------------
# Helm mode: the chart E2E tests live in the charts repo (GHE), not the operator
# repo. Clone it at the frozen SHA.
# ---------------------------------------------------------------------------
if [[ "${E2E_MODE}" == "helm" ]]; then
  GHE_TOKEN=$(get_secret ghe-token)
  GHE_TOKEN_FILE=$(mktemp)
  echo "${GHE_TOKEN}" > "${GHE_TOKEN_FILE}"
  chmod 600 "${GHE_TOKEN_FILE}"

  # github.ibm.com (GHE) requires GIT_AUTH_USER=x-access-token
  set +u
  GIT_AUTH_USER="x-access-token"
  # shellcheck source=/dev/null
  . "${ONE_PIPELINE_PATH}"/git/clone_repo_v2 \
    --repository "https://github.ibm.com/instana/instana-agent-charts" \
    --token-path "${GHE_TOKEN_FILE}" \
    --branch "$(jq -r '."charts-branch"' <<< "${BOM_JSON}")" \
    --depth 1 \
    --directory charts-src \
    --use-lfs false \
    --force-exit false
  unset GIT_AUTH_USER
  set -u
  git -C "${WORKSPACE}/charts-src" fetch --depth 1 origin "${CHARTS_SHA}" || exit 1
  git -C "${WORKSPACE}/charts-src" checkout --detach "${CHARTS_SHA}" || exit 1
  rm -f "${GHE_TOKEN_FILE}"
  CHARTS_DIRECTORY="${WORKSPACE}/charts-src"
else
  # Build operator binary (required by the operator e2e harness)
  cd "${SOURCE_DIRECTORY}"
  make generate
  go install
  make build
fi

# ---------------------------------------------------------------------------
# Authenticate with clusters
# ---------------------------------------------------------------------------
CLUSTER_DETAILS=$(get_secret "${CLUSTER_ID}")
CLUSTER_TYPE=$(echo "${CLUSTER_DETAILS}" | jq -r ".type")
CLUSTER_NAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".name")

ICR_USERNAME=iamapikey
ICR_PASSWORD=$(cat /config/api-key)
export ICR_USERNAME ICR_PASSWORD CLUSTER_NAME

# setup.sh reads SKIP_INSTALL_GCLOUD; it must always be defined (set -u)
if [[ "${CLUSTER_TYPE}" == "fyre-ocp" ]]; then
  SKIP_INSTALL_GCLOUD=true
else
  SKIP_INSTALL_GCLOUD=false
fi
# Helm mode uses the charts repo's setup (helm v4, Go, kubectl, gcloud);
# operator mode uses the operator repo's setup.
if [[ "${E2E_MODE}" == "helm" ]]; then
  # shellcheck disable=SC1090,SC1091
  source "${CHARTS_DIRECTORY}/ci/sps/setup.sh"
else
  # shellcheck disable=SC1090
  source "${SOURCE_DIRECTORY}/ci/sps-scripts/setup.sh"
fi

if [[ "${CLUSTER_TYPE}" == "fyre-ocp" ]]; then
  CLUSTER_SERVER=$(echo "${CLUSTER_DETAILS}"   | jq -r ".server")
  CLUSTER_USERNAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".username")
  CLUSTER_PASSWORD=$(echo "${CLUSTER_DETAILS}" | jq -r ".password")
  CLUSTER_CHANNEL=$(echo "${CLUSTER_DETAILS}"  | jq -r ".channel")
  mkdir -p "${SOURCE_DIRECTORY}/bin"
  cd "${SOURCE_DIRECTORY}/bin"
  curl -sk "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${CLUSTER_CHANNEL}/openshift-client-linux.tar.gz" \
    -o openshift-client-linux.tar.gz
  if ! tar -xf openshift-client-linux.tar.gz; then
    echo "ERROR: could not extract oc client for channel '${CLUSTER_CHANNEL}'" >&2
    exit 1
  fi
  rm -f openshift-client-linux.tar.gz README.md
  PATH=$(pwd):${PATH}; export PATH
  # Non-secret diagnostics to debug login failures (password is never printed)
  echo "oc binary: $(command -v oc)  channel=${CLUSTER_CHANNEL}"
  oc version --client 2>&1 | head -2
  echo "cluster=${CLUSTER_ID} server=${CLUSTER_SERVER} username=${CLUSTER_USERNAME} password-length=${#CLUSTER_PASSWORD}"
  if [[ "${CLUSTER_PASSWORD}" =~ [[:space:]] || "${CLUSTER_PASSWORD}" == "null" || -z "${CLUSTER_PASSWORD}" ]]; then
    echo "WARNING: password is empty, 'null' or contains whitespace" >&2
  fi
  if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then set +x; fi
  oc login --insecure-skip-tls-verify=true \
    -u "${CLUSTER_USERNAME}" -p "${CLUSTER_PASSWORD}" --server="${CLUSTER_SERVER}"
  if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then set -x; fi
elif [[ "${CLUSTER_TYPE}" == "gke" ]]; then
  CLUSTER_ZONE=$(echo "${CLUSTER_DETAILS}"    | jq -r ".zone")
  CLUSTER_PROJECT=$(echo "${CLUSTER_DETAILS}" | jq -r ".project")
  get_secret gcp-service-account > keyfile.json
  gcloud auth activate-service-account --key-file keyfile.json
  gcloud container clusters get-credentials "${CLUSTER_NAME}" \
    --zone "${CLUSTER_ZONE}" --project "${CLUSTER_PROJECT}"
else
  echo "ERROR: Unknown cluster type '${CLUSTER_TYPE}'"
  exit 1
fi

cd "${SOURCE_DIRECTORY}"
kubectl get nodes -o wide

# ---------------------------------------------------------------------------
# Fetch E2E backend credentials
# ---------------------------------------------------------------------------
INSTANA_E2E_BACKEND_DETAILS=$(get_secret instana-e2e-backend-details)
INSTANA_ENDPOINT_HOST=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".endpoint_host")
INSTANA_ENDPOINT_PORT=443
INSTANA_API_KEY=$(echo "${INSTANA_E2E_BACKEND_DETAILS}"   | jq -r ".agent_key")
INSTANA_API_URL=$(echo "${INSTANA_E2E_BACKEND_DETAILS}"   | jq -r ".api_url")
INSTANA_API_TOKEN=$(echo "${INSTANA_E2E_BACKEND_DETAILS}" | jq -r ".api_token")
# The charts e2e tests read INSTANA_AGENT_KEY; the operator e2e tests read INSTANA_API_KEY.
INSTANA_AGENT_KEY="${INSTANA_API_KEY}"
export INSTANA_ENDPOINT_HOST INSTANA_ENDPOINT_PORT INSTANA_API_KEY INSTANA_AGENT_KEY INSTANA_API_URL INSTANA_API_TOKEN

# Export the dev ICR image digest so E2E tests pick it up
export GIT_COMMIT="${OPERATOR_SHA}"

# ---------------------------------------------------------------------------
# Claim reslock
# ---------------------------------------------------------------------------
# reslock.sh builds its lock owner from these variables, which are only
# provided by the PR pipelines; set them for the release async stage.
export APP_REPO_NAME="${APP_REPO_NAME:-instana-agent-operator}"
export BRANCH="${BRANCH:-${SOURCE_BRANCH}}"
export PIPELINE_RUN_NAME="${PIPELINE_RUN_NAME:-release-${VERSION}}"
export BUILD_NUMBER="${BUILD_NUMBER:-${E2E_MODE}-${CLUSTER_ID}}"

RESLOCK_CLAIMED=false
E2E_STATUS="failure"
E2E_EXIT_CODE=0

release_reslock() {
  if [[ "${RESLOCK_CLAIMED}" == "true" ]]; then
    RESLOCK_CLAIMED=false
    echo "=== Releasing reslock for ${CLUSTER_ID} ==="
    bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" release "${CLUSTER_ID}" || \
      echo "WARNING: failed to release reslock for ${CLUSTER_ID}" >&2
  fi
}

# Safety net: release the lock on any exit path (error, cancellation, signal).
on_exit() {
  release_reslock
  if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then env; fi
}
trap on_exit EXIT

echo "=== Claiming reslock for ${CLUSTER_ID} ==="
if bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" claim "${CLUSTER_ID}"; then
  RESLOCK_CLAIMED=true
else
  echo "ERROR: failed to claim reslock for ${CLUSTER_ID}; E2E tests will not be started" >&2
  E2E_EXIT_CODE=1
fi

# ---------------------------------------------------------------------------
# Record run timestamp
# ---------------------------------------------------------------------------
E2E_TIMESTAMP=$(date -u '+%Y%m%d%H%M')

# ---------------------------------------------------------------------------
# Run E2E tests (capture output; never exit early on failure)
# ---------------------------------------------------------------------------
LOG_FILE="${WORKSPACE}/e2e-${CLUSTER_ID}-${E2E_TIMESTAMP}.log"

if [[ "${RESLOCK_CLAIMED}" != "true" ]]; then
  echo "E2E tests skipped: reslock for ${CLUSTER_ID} could not be claimed" | tee "${LOG_FILE}"
elif [[ "${E2E_MODE}" == "helm" ]]; then
  echo "=== Running E2E tests (mode=${E2E_MODE}) — output captured to ${LOG_FILE} ==="
  # For helm mode: pull the chart tarball built by the release pipeline from COS,
  # then run the charts repo e2e tests against it.
  # The file name must match instana-agent-*.tgz and must not pre-exist.
  HELM_TGZ="${WORKSPACE}/$(basename "${HELM_CHART_COS}")"
  ibmcloud cos get-object \
    --bucket "${COS_BUCKET}" \
    --key "${HELM_CHART_COS}" \
    --region "${COS_REGION}" \
    --output json "${HELM_TGZ}"
  export INSTANA_AGENT_HELM_CHART_LOCATION="${HELM_TGZ}"
  echo "INSTANA_AGENT_HELM_CHART_LOCATION=${INSTANA_AGENT_HELM_CHART_LOCATION}"
  (cd "${CHARTS_DIRECTORY}" && make e2e) 2>&1 | tee "${LOG_FILE}" || E2E_EXIT_CODE=$?
else
  echo "=== Running E2E tests (mode=${E2E_MODE}) — output captured to ${LOG_FILE} ==="
  make e2e 2>&1 | tee "${LOG_FILE}" || E2E_EXIT_CODE=$?
fi

if [[ ${E2E_EXIT_CODE} -eq 0 ]]; then
  E2E_STATUS="success"
fi
echo "E2E result: ${E2E_STATUS} (exit code: ${E2E_EXIT_CODE})"

# ---------------------------------------------------------------------------
# Release reslock (always, even on failure)
# ---------------------------------------------------------------------------
release_reslock

# ---------------------------------------------------------------------------
# Upload log file to COS (always uploaded)
# ---------------------------------------------------------------------------
LOG_COS_KEY="${COS_STATE_PATH}/e2e/${CLUSTER_ID}/test-${E2E_TIMESTAMP}-${E2E_STATUS}.log"
ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${LOG_COS_KEY}" \
  --body "${LOG_FILE}" \
  --region "${COS_REGION}"
echo "COS upload: cos://${COS_BUCKET}/${LOG_COS_KEY}"

# ---------------------------------------------------------------------------
# Write (overwrite) the status pointer file
# ---------------------------------------------------------------------------
STATUS_JSON=$(jq -n \
  --arg status    "${E2E_STATUS}" \
  --arg cluster   "${CLUSTER_ID}" \
  --arg timestamp "$(date -u '+%Y-%m-%dT%H:%M:%SZ')" \
  --arg log       "${LOG_COS_KEY}" \
  '{
    "status":    $status,
    "cluster":   $cluster,
    "timestamp": $timestamp,
    "log":       $log
  }')

STATUS_FILE=$(mktemp)
echo "${STATUS_JSON}" > "${STATUS_FILE}"

STATUS_COS_KEY="${COS_STATE_PATH}/e2e/${CLUSTER_ID}.json"
ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${STATUS_COS_KEY}" \
  --body "${STATUS_FILE}" \
  --region "${COS_REGION}"
echo "COS upload: cos://${COS_BUCKET}/${STATUS_COS_KEY}"
rm -f "${STATUS_FILE}"

# ---------------------------------------------------------------------------
# Post git commit status to operator repo
# ---------------------------------------------------------------------------
GH_API_TOKEN=$(get_secret github-token)
COMMIT_STATE="${E2E_STATUS}"    # "success" or "failure"

curl --silent --fail --show-error -X POST \
  -H "Authorization: Bearer ${GH_API_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{\"state\":\"${COMMIT_STATE}\",\"context\":\"tekton/e2e-${CLUSTER_ID}\",\"description\":\"E2E test (${E2E_MODE})\",\"target_url\":\"${PIPELINE_RUN_URL:-}\"}" \
  "https://api.github.com/repos/instana/instana-agent-operator/statuses/${OPERATOR_SHA}"

echo "Posted commit status '${COMMIT_STATE}' to operator repo at ${OPERATOR_SHA}"

# ---------------------------------------------------------------------------
# Fail the SPS stage if E2E tests failed
# ---------------------------------------------------------------------------
if [[ ${E2E_EXIT_CODE} -ne 0 ]]; then
  echo "ERROR: E2E tests failed for cluster ${CLUSTER_ID}"
  exit ${E2E_EXIT_CODE}
fi

echo "===== e2e-async.sh - end ====="
