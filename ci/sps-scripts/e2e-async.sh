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
COS_STATE_PATH=$(get_env cos-state-path)

# ---------------------------------------------------------------------------
# Read BOM from COS
# ---------------------------------------------------------------------------
BOM_FILE=$(mktemp)
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "pipeline-state/latest-release-candidate/bom.json" \
  --output json "${BOM_FILE}"

VERSION=$(jq -r '.version'                         "${BOM_FILE}")
OPERATOR_SHA=$(jq -r '.sources."operator-sha"'     "${BOM_FILE}")
CHARTS_SHA=$(jq -r '.sources."charts-sha"'         "${BOM_FILE}")
OPERATOR_MANIFEST_DIGEST=$(jq -r '.images."operator-manifest-digest"' "${BOM_FILE}")
HELM_CHART_COS=$(jq -r '.artifacts."helm-chart"'   "${BOM_FILE}")
SOURCE_BRANCH=$(jq -r '."source-branch"'           "${BOM_FILE}")

echo "version=${VERSION}  operator-sha=${OPERATOR_SHA}"
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"

# ---------------------------------------------------------------------------
# Install dependencies
# ---------------------------------------------------------------------------
source "${WORKSPACE}/${APP_REPO_FOLDER}/installGolang.sh" amd64
export PATH=$PATH:/usr/local/go/bin

# ---------------------------------------------------------------------------
# Clone operator repo at frozen SHA (E2E tests live in the operator repo)
# ---------------------------------------------------------------------------
GH_TOKEN=$(get_secret github-token)
clone_repo_v2 \
  --repo "https://github.ibm.com/instana/instana-agent-operator" \
  --token-path <(echo "${GH_TOKEN}") \
  --branch "${SOURCE_BRANCH}" \
  --depth 1 \
  --directory operator-src
git -C operator-src checkout "${OPERATOR_SHA}"

SOURCE_DIRECTORY="${WORKSPACE}/operator-src"

# Build operator binary (required by e2e harness)
cd "${SOURCE_DIRECTORY}"
make generate
go install
make build

# ---------------------------------------------------------------------------
# Authenticate with clusters
# ---------------------------------------------------------------------------
CLUSTER_DETAILS=$(get_secret "${CLUSTER_ID}")
CLUSTER_TYPE=$(echo "${CLUSTER_DETAILS}" | jq -r ".type")
CLUSTER_NAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".name")

ICR_USERNAME=iamapikey
ICR_PASSWORD=$(cat /config/api-key)
export ICR_USERNAME ICR_PASSWORD CLUSTER_NAME

if [[ "${CLUSTER_TYPE}" == "fyre-ocp" ]]; then
  SKIP_INSTALL_GCLOUD=true
fi
# shellcheck disable=SC1090
source "${SOURCE_DIRECTORY}/ci/sps-scripts/setup.sh"

if [[ "${CLUSTER_TYPE}" == "fyre-ocp" ]]; then
  CLUSTER_SERVER=$(echo "${CLUSTER_DETAILS}"   | jq -r ".server")
  CLUSTER_USERNAME=$(echo "${CLUSTER_DETAILS}" | jq -r ".username")
  CLUSTER_PASSWORD=$(echo "${CLUSTER_DETAILS}" | jq -r ".password")
  CLUSTER_CHANNEL=$(echo "${CLUSTER_DETAILS}"  | jq -r ".channel")
  mkdir -p "${SOURCE_DIRECTORY}/bin"
  cd "${SOURCE_DIRECTORY}/bin"
  curl -sk "https://mirror.openshift.com/pub/openshift-v4/clients/ocp/${CLUSTER_CHANNEL}/openshift-client-linux.tar.gz" \
    -o openshift-client-linux.tar.gz
  tar -xf openshift-client-linux.tar.gz
  rm -f openshift-client-linux.tar.gz README.md
  PATH=$(pwd):${PATH}; export PATH
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
export INSTANA_ENDPOINT_HOST INSTANA_ENDPOINT_PORT INSTANA_API_KEY INSTANA_API_URL INSTANA_API_TOKEN

# Export the dev ICR image digest so E2E tests pick it up
export GIT_COMMIT="${OPERATOR_SHA}"

# ---------------------------------------------------------------------------
# Claim reslock
# ---------------------------------------------------------------------------
echo "=== Claiming reslock for ${CLUSTER_ID} ==="
bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" claim "${CLUSTER_ID}"
RESLOCK_CLAIMED=true

# ---------------------------------------------------------------------------
# Record run timestamp
# ---------------------------------------------------------------------------
E2E_TIMESTAMP=$(date -u '+%Y%m%d%H%M')

# ---------------------------------------------------------------------------
# Run E2E tests (capture output; never exit early on failure)
# ---------------------------------------------------------------------------
LOG_FILE="${WORKSPACE}/e2e-${CLUSTER_ID}-${E2E_TIMESTAMP}.log"
E2E_STATUS="failure"
E2E_EXIT_CODE=0

echo "=== Running E2E tests (mode=${E2E_MODE}) — output captured to ${LOG_FILE} ==="
if [[ "${E2E_MODE}" == "helm" ]]; then
  # For helm mode: pull chart tarball from COS first, then run helm e2e target
  HELM_TGZ=$(mktemp --suffix=".tgz")
  ibmcloud cos get-object \
    --bucket "${COS_BUCKET}" \
    --key "${HELM_CHART_COS}" \
    --output json "${HELM_TGZ}"
  export HELM_CHART_PATH="${HELM_TGZ}"
  make e2e-helm 2>&1 | tee "${LOG_FILE}" || E2E_EXIT_CODE=$?
else
  make e2e 2>&1 | tee "${LOG_FILE}" || E2E_EXIT_CODE=$?
fi

if [[ ${E2E_EXIT_CODE} -eq 0 ]]; then
  E2E_STATUS="success"
fi
echo "E2E result: ${E2E_STATUS} (exit code: ${E2E_EXIT_CODE})"

# ---------------------------------------------------------------------------
# Release reslock (always, even on failure)
# ---------------------------------------------------------------------------
echo "=== Releasing reslock for ${CLUSTER_ID} ==="
bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" release "${CLUSTER_ID}" || true
RESLOCK_CLAIMED=false

# ---------------------------------------------------------------------------
# Upload log file to COS (always uploaded)
# ---------------------------------------------------------------------------
LOG_COS_KEY="${COS_STATE_PATH}/e2e/${CLUSTER_ID}/test-${E2E_TIMESTAMP}-${E2E_STATUS}.log"
ibmcloud cos put-object \
  --bucket "${COS_BUCKET}" \
  --key "${LOG_COS_KEY}" \
  --body "${LOG_FILE}"
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
  --body "${STATUS_FILE}"
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
  "https://api.github.ibm.com/repos/instana/instana-agent-operator/statuses/${OPERATOR_SHA}"

echo "Posted commit status '${COMMIT_STATE}' to operator repo at ${OPERATOR_SHA}"

# ---------------------------------------------------------------------------
# Fail the SPS stage if E2E tests failed
# ---------------------------------------------------------------------------
if [[ ${E2E_EXIT_CODE} -ne 0 ]]; then
  echo "ERROR: E2E tests failed for cluster ${CLUSTER_ID}"
  exit ${E2E_EXIT_CODE}
fi

echo "===== e2e-async.sh - end ====="
