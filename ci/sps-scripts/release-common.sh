#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Shared, side-effect-free definitions. Callers explicitly log in when needed.
set +x  # Never inherit credential-bearing xtrace from a pipeline wrapper.

cos_login() {
  local key
  key=$(get_secret ibmcloud-api-key)
  ibmcloud login --apikey "${key}" --no-region
}

cos_get() {
  local key="$1" out="$2"
  rm -f "${out}"
  ibmcloud cos get-object --bucket "${COS_BUCKET}" --key "${key}" \
    --region "${COS_REGION}" --output json "${out}"
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
  printf '%s  %s\n' "$2" "$1" | sha256sum --check
}
