#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
VERSION=$(jq -er '.version' "${SELECTED_BOM_FILE}")
CREDS=$(get_secret redhat-registry)
AUTH_DIR=$(mktemp -d "${WORKSPACE}/preflight-auth-XXXXXX")
trap 'rm -f "${AUTH_DIR}/config.json"; rmdir "${AUTH_DIR}"' EXIT
printf '%s' "${CREDS}" | jq -r '.password' | \
  skopeo login --authfile "${AUTH_DIR}/config.json" --username "$(jq -er '.username' <<< "${CREDS}")" \
    --password-stdin quay.io
TOKEN=$(jq -er '.token | strings | select(length > 0)' <<< "${CREDS}")
mkdir -p "${WORKSPACE}/preflight-output"
RC=0
preflight check container "quay.io/redhat-isv-containers/5e961c2c93604e02afa9ebdf:${VERSION}" \
  --docker-config "${AUTH_DIR}/config.json" --artifacts "${WORKSPACE}/preflight-output" \
  --certification-component-id=5e961c2c93604e02afa9ebdf --submit --pyxis-api-token="${TOKEN}" || RC=$?
COS_BUCKET=$(get_env cos-bucket-name)
COS_REGION=$(get_env cos-region us-south)
STATE=$(jq -er '."cos-state-path"' "${SELECTED_BOM_FILE}")
ARTIFACT="${WORKSPACE}/preflight-$(date -u '+%Y%m%d%H%M%S')-$(cat /proc/sys/kernel/random/uuid).tgz"
tar -czf "${ARTIFACT}" -C "${WORKSPACE}/preflight-output" .
cos_put "${STATE}/promotion/$(basename "${ARTIFACT}")" "${ARTIFACT}"
exit "${RC}"
