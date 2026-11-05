#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Shared async E2E runner for release-build pipeline.
# Parameterised via environment variables set in each stage's script block:
#   CLUSTER_ID  — one of: ocp-fyre-latest, gke-operator-lowest, ocp-fyre-lowest, gke-operator-latest
#   E2E_MODE    — "operator" or "helm"
#
# Behavior (EXIT finalizer attempts log/result uploads, including setup failures):
#   1. Read BOM from COS
#   2. Claim reslock
#   3. Run E2E tests (captures exit code, does not exit immediately on failure)
#   4. Release reslock
#   5. Upload timestamped log file
#   6. Retain each attempt; only successful attempts update the durable pass pointer
#   7. Post git commit status to operator repo at operator-sha
#   8. Exit non-zero if tests failed
#
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
echo "===== e2e-async.sh - start ====="
echo "CLUSTER_ID=${CLUSTER_ID}  E2E_MODE=${E2E_MODE}"

# ---------------------------------------------------------------------------
# Read pipeline environment
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region "us-south")
COS_STATE_PATH=$(get_env cos-state-path)
E2E_TIMESTAMP="$(date -u '+%Y%m%d%H%M%S')-$(cat /proc/sys/kernel/random/uuid)"
LOG_FILE="${WORKSPACE}/e2e-${CLUSTER_ID}-${E2E_TIMESTAMP}.log"
touch "${LOG_FILE}"
RESLOCK_CLAIMED=false
E2E_STATUS=failure
BOM_JSON='{}'

release_reslock() {
  if [[ "${RESLOCK_CLAIMED}" == true ]]; then
    if bash "${SOURCE_DIRECTORY}/ci/sps-scripts/reslock.sh" release "${CLUSTER_ID}"; then
      RESLOCK_CLAIMED=false
    else
      echo "ERROR: lock release failed for ${CLUSTER_ID}" >&2
      return 1
    fi
  fi
}

finalize_e2e() {
  local rc=$? status_file log_key pointer
  trap - EXIT TERM INT
  set +e
  release_reslock || rc=1
  if [[ ${rc} -ne 0 ]]; then E2E_STATUS=failure; fi
  echo "Attempt result: ${E2E_STATUS} (exit ${rc})" | tee -a "${LOG_FILE}"
  log_key="${COS_STATE_PATH}/e2e/${CLUSTER_ID}/test-${E2E_TIMESTAMP}-${E2E_STATUS}.log"
  status_file="${WORKSPACE}/status-${CLUSTER_ID}-${E2E_TIMESTAMP}.json"
  jq -n --argjson bom "${BOM_JSON}" --arg status "${E2E_STATUS}" --arg cluster "${CLUSTER_ID}" \
    --arg mode "${E2E_MODE}" --arg attempt "${E2E_TIMESTAMP}" --arg log "${log_key}" \
    --arg log_sha "$(sha256sum "${LOG_FILE}" | cut -d' ' -f1)" \
    '{status:$status, cluster:$cluster, mode:$mode, attempt:$attempt, log:$log, "log-sha256":$log_sha,
      "candidate-id":$bom["candidate-id"], sources:$bom.sources, images:$bom.images,
      checksums:$bom.checksums}' > "${status_file}"
  # Every attempt is retained. Failure never erases a prior pass for this candidate.
  if cos_put "${log_key}" "${LOG_FILE}" &&
      cos_put "${COS_STATE_PATH}/e2e/${CLUSTER_ID}/${E2E_TIMESTAMP}.json" "${status_file}"; then
    if [[ "${E2E_STATUS}" == success ]]; then
      pointer="${COS_STATE_PATH}/e2e/success/${CLUSTER_ID}.json"
      cos_put "${pointer}" "${status_file}" || rc=1
    fi
  else
    rc=1
  fi
  if [[ -n "${OPERATOR_SHA:-}" ]]; then
    GH_API_TOKEN=$(get_secret github-token)
    curl --silent --fail --show-error -X POST -H "Authorization: Bearer ${GH_API_TOKEN}" \
      -H 'Content-Type: application/json' \
      -d "$(jq -n --arg state "${E2E_STATUS}" --arg context "tekton/e2e-${CLUSTER_ID}" \
        --arg url "${PIPELINE_RUN_URL:-}" '{state:$state,context:$context,target_url:$url,
        description:"Release candidate E2E attempt"}')" \
      "https://api.github.com/repos/instana/instana-agent-operator/statuses/${OPERATOR_SHA}" || rc=1
  fi
  exit "${rc}"
}
trap finalize_e2e EXIT
trap 'exit 143' TERM
trap 'exit 130' INT

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
OPERATOR_MANIFEST=$(jq -er '.images."operator-manifest"' "${BOM_FILE}")
HELM_CHART_COS=$(jq -r '.artifacts."helm-chart"'   "${BOM_FILE}")
SOURCE_BRANCH=$(jq -r '."source-branch"'           "${BOM_FILE}")
BOM_JSON=$(cat "${BOM_FILE}")
rm -f "${BOM_FILE}"

echo "version=${VERSION}  operator-sha=${OPERATOR_SHA}"
echo "charts-sha=${CHARTS_SHA}"
echo "operator-manifest-digest=${OPERATOR_MANIFEST_DIGEST}"

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

# Install Go from the frozen operator checkout, never from the caller's cwd.
bash "${WORKSPACE}/operator-src/installGolang.sh" amd64
export PATH="/usr/local/go/bin:${PATH}"
export GO_ALREADY_INSTALLED=true

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
set -euo pipefail

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
set -euo pipefail

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
  oc login --insecure-skip-tls-verify=true \
    -u "${CLUSTER_USERNAME}" -p "${CLUSTER_PASSWORD}" --server="${CLUSTER_SERVER}"
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

# Use the readable candidate tag; the BOM digest is checked around the suite.
export GIT_COMMIT="${OPERATOR_SHA}"
export OPERATOR_IMAGE_NAME="${OPERATOR_MANIFEST%:*}"
export OPERATOR_IMAGE_TAG="${OPERATOR_MANIFEST##*:}"

# ---------------------------------------------------------------------------
# Claim reslock
# ---------------------------------------------------------------------------
# reslock.sh builds its lock owner from these variables, which are only
# provided by the PR pipelines; set them for the release async stage.
export APP_REPO_NAME="${APP_REPO_NAME:-instana-agent-operator}"
export BRANCH="${BRANCH:-${SOURCE_BRANCH}}"
export PIPELINE_RUN_NAME="${PIPELINE_RUN_NAME:-release-${VERSION}}"
export BUILD_NUMBER="${BUILD_NUMBER:-${E2E_MODE}-${CLUSTER_ID}}"

E2E_EXIT_CODE=0

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
verify_tag "${OPERATOR_MANIFEST}" "${OPERATOR_MANIFEST_DIGEST}"

# ---------------------------------------------------------------------------
# Run E2E tests (capture output; never exit early on failure)
# ---------------------------------------------------------------------------
if [[ "${RESLOCK_CLAIMED}" != "true" ]]; then
  echo "E2E tests skipped: reslock for ${CLUSTER_ID} could not be claimed" | tee "${LOG_FILE}"
elif [[ "${E2E_MODE}" == "helm" ]]; then
  echo "=== Running E2E tests (mode=${E2E_MODE}) — output captured to ${LOG_FILE} ==="
  # For helm mode: pull the chart tarball built by the release pipeline from COS,
  # then run the charts repo e2e tests against it.
  # The file name must match instana-agent-*.tgz and must not pre-exist.
  HELM_TGZ="${WORKSPACE}/$(basename "${HELM_CHART_COS}")"
  rm -f "${HELM_TGZ}"
  timeout 5m ibmcloud cos get-object \
    --bucket "${COS_BUCKET}" \
    --key "${HELM_CHART_COS}" \
    --region "${COS_REGION}" \
    --output json "${HELM_TGZ}"
  verify_sha256 "${HELM_TGZ}" "$(jq -er '.checksums["helm-chart"]' <<< "${BOM_JSON}")"
  export INSTANA_AGENT_HELM_CHART_LOCATION="${HELM_TGZ}"
  echo "INSTANA_AGENT_HELM_CHART_LOCATION=${INSTANA_AGENT_HELM_CHART_LOCATION}"
  (cd "${CHARTS_DIRECTORY}" && timeout --signal=TERM --kill-after=60s 55m make e2e) \
    2>&1 | tee "${LOG_FILE}" || E2E_EXIT_CODE=$?
else
  echo "=== Running E2E tests (mode=${E2E_MODE}) — output captured to ${LOG_FILE} ==="
  timeout --signal=TERM --kill-after=60s 55m make e2e 2>&1 | tee "${LOG_FILE}" || E2E_EXIT_CODE=$?
fi

verify_tag "${OPERATOR_MANIFEST}" "${OPERATOR_MANIFEST_DIGEST}"
if [[ ${E2E_EXIT_CODE} -eq 0 ]]; then
  E2E_STATUS="success"
fi
echo "E2E result: ${E2E_STATUS} (exit code: ${E2E_EXIT_CODE})"

# ---------------------------------------------------------------------------
# Release reslock (always, even on failure)
# ---------------------------------------------------------------------------
release_reslock

# ---------------------------------------------------------------------------
# Fail the SPS stage if E2E tests failed
# ---------------------------------------------------------------------------
if [[ ${E2E_EXIT_CODE} -ne 0 ]]; then
  echo "ERROR: E2E tests failed for cluster ${CLUSTER_ID}"
  exit ${E2E_EXIT_CODE}
fi

echo "===== e2e-async.sh - end ====="
exit 0
