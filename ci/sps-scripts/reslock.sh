#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Helper script to calculate CI Server specific metadata to be used for locking.
# When invoked, it generates a unique lock owner name from concourse meta-data and
# fetched the most recent go binary from github to execute the lock or release command. 
set -euo pipefail
echo "===== reslock.sh - start ====="
RESLOCK_COMMAND=$1
RESLOCK_RESOURCE_NAME=$2

echo "${RESLOCK_COMMAND} lock ${RESLOCK_RESOURCE_NAME}"

export RESLOCK_GITHUB_REPO_OWNER=instana
export RESLOCK_LOCK_OWNER="${APP_REPO_NAME}/${BRANCH}/${PIPELINE_RUN_NAME}/${BUILD_NUMBER}"
RESLOCK_GITHUB_TOKEN=$(get_secret reslock-github-token)
export RESLOCK_GITHUB_TOKEN

echo "RESLOCK_GITHUB_REPO_OWNER=${RESLOCK_GITHUB_REPO_OWNER}"
echo "RESLOCK_LOCK_OWNER=${RESLOCK_LOCK_OWNER}"

RESLOCK_RUNNER=$(mktemp)
trap 'rm -f "${RESLOCK_RUNNER}"' EXIT

# run.sh (fetched below) calls `curl -sfL` without --show-error, so a failing
# GitHub API call there exits silently. Probe the same endpoint to surface why.
# Never use `bash -x` here: it would print the token.
reslock_diagnose() {
  local probe_headers probe_body
  probe_headers=$(mktemp)
  probe_body=$(mktemp)
  echo "--- reslock diagnostics (${RESLOCK_COMMAND} ${RESLOCK_RESOURCE_NAME}) ---" >&2
  echo "date (UTC): $(date -u '+%Y-%m-%dT%H:%M:%SZ')" >&2
  for url in \
    'https://github.ibm.com/api/v3/repos/instana/reslock/releases/latest' \
    'https://github.ibm.com/api/v3/rate_limit'; do
    local code
    code=$(curl --silent --location --max-time 30 \
      -H "Accept: application/vnd.github+json" \
      -H "Authorization: Bearer ${RESLOCK_GITHUB_TOKEN}" \
      -D "${probe_headers}" -o "${probe_body}" -w '%{http_code}' "${url}") || code="curl-failed($?)"
    echo "GET ${url} -> HTTP ${code}" >&2
    grep -iE '^(x-ratelimit-|retry-after|x-github-sso|x-accepted-oauth-scopes)' "${probe_headers}" >&2 || true
    if [[ "${code}" != 200 ]]; then
      head -c 500 "${probe_body}" >&2 || true
      echo >&2
    fi
  done
  rm -f "${probe_headers}" "${probe_body}"
  echo "--- end reslock diagnostics ---" >&2
}

echo "Fetching reslock runner script"
if ! curl --fail --silent --show-error --retry 3 \
  -H "Authorization: Bearer ${RESLOCK_GITHUB_TOKEN}" \
  'https://raw.github.ibm.com/instana/reslock/main/run.sh' > "${RESLOCK_RUNNER}"; then
  echo "ERROR: could not download reslock run.sh" >&2
  exit 1
fi
test -s "${RESLOCK_RUNNER}" || { echo "ERROR: downloaded reslock run.sh is empty" >&2; exit 1; }
bash -n "${RESLOCK_RUNNER}" || { echo "ERROR: downloaded reslock run.sh has syntax errors" >&2; exit 1; }

if [ "${RESLOCK_COMMAND}" == "claim" ]; then
    RESLOCK_ARGS=(claim k8s-clusters "${RESLOCK_RESOURCE_NAME}" -t 70m -w)
else
    RESLOCK_ARGS=("${RESLOCK_COMMAND}" k8s-clusters "${RESLOCK_RESOURCE_NAME}")
fi

echo "Running reslock ${RESLOCK_ARGS[*]}"
RESLOCK_RC=0
bash "${RESLOCK_RUNNER}" "${RESLOCK_ARGS[@]}" || RESLOCK_RC=$?
if [[ ${RESLOCK_RC} -ne 0 ]]; then
  echo "ERROR: reslock ${RESLOCK_ARGS[*]} failed with exit code ${RESLOCK_RC}" >&2
  reslock_diagnose
  exit "${RESLOCK_RC}"
fi
echo "===== reslock.sh - end ====="
