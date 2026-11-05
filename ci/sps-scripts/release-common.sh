#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Shared, side-effect-free definitions. Callers explicitly log in when needed.
set +x  # Never inherit credential-bearing xtrace from a pipeline wrapper.

# Log descriptions, never command lines: commands may contain credentials.
release_check_start() {
  RELEASE_CHECK="$1"
  RELEASE_CHECK_DETAIL="$2"
  echo "CHECK [${RELEASE_CHECK}]: ${RELEASE_CHECK_DETAIL}"
}

release_check_result() {
  local status="$1" rc="${2:-0}"
  echo "${status^^} [${RELEASE_CHECK}]: ${RELEASE_CHECK_DETAIL} (exit code ${rc})"
  if [[ -n "${RELEASE_CHECKS_FILE:-}" ]]; then
    jq -nc --arg check "${RELEASE_CHECK}" --arg detail "${RELEASE_CHECK_DETAIL}" \
      --arg status "${status}" --argjson rc "${rc}" \
      '{check:$check, status:$status, detail:$detail, "exit-code":$rc}' >> "${RELEASE_CHECKS_FILE}"
  fi
}

release_check_pass() {
  release_check_result passed
  RELEASE_CHECK=''
}

release_check_exit() {
  local rc=$?
  trap - EXIT
  if [[ ${rc} -ne 0 && -n "${RELEASE_CHECK:-}" ]]; then
    release_check_result failed "${rc}" >&2
    echo 'Stopped at the failed check; subsequent checks were not run.' >&2
  fi
  exit "${rc}"
}

cos_login() {
  local key
  key=$(get_secret ibmcloud-api-key)
  ibmcloud login --apikey "${key}" --no-region
}

cos_get() {
  local key="$1" out="$2"
  echo "COS download: cos://${COS_BUCKET}/${key} -> ${out}"
  rm -f "${out}"
  local rc=0
  ibmcloud cos get-object --bucket "${COS_BUCKET}" --key "${key}" \
    --region "${COS_REGION}" --output json "${out}" || rc=$?
  if [[ ${rc} -ne 0 ]]; then
    echo "ERROR: COS download failed: cos://${COS_BUCKET}/${key} (region ${COS_REGION}, exit code ${rc})." >&2
    echo 'Check the CLI error above for a missing object, access denial, or service failure.' >&2
    return "${rc}"
  fi
  echo "Downloaded: cos://${COS_BUCKET}/${key}"
}

cos_put() {
  ibmcloud cos put-object --bucket "${COS_BUCKET}" --key "$1" --body "$2" --region "${COS_REGION}" || return $?
  echo "COS upload: cos://${COS_BUCKET}/$1"
}

verify_tag() {
  local image="$1" expected="$2" actual
  actual=$(timeout 2m skopeo inspect --creds "iamapikey:$(cat /config/api-key)" \
    --format '{{.Digest}}' "docker://${image}")
  if [[ "${actual}" != "${expected}" ]]; then
    echo "ERROR: ${image} resolves to ${actual}, expected ${expected}" >&2
    return 1
  fi
  echo "Verified image tag: ${image} → ${actual}"
}

verify_sha256() {
  echo "SHA256 check: $1; expected $2"
  printf '%s  %s\n' "$2" "$1" | sha256sum --check
}
