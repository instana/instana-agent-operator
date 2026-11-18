#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
BOM_FILE="${SELECTED_BOM_FILE:?Run through run-promotion.sh}"
VERSION=$(jq -er '.version' "${BOM_FILE}")
DIGEST=$(jq -er '.images["operator-manifest-digest"]' "${BOM_FILE}")
SRC="icr.io/instana-agent-dev/instana-agent-operator@${DIGEST}"
ICR_KEY=$(cat /config/api-key)

copy_version() {
  local dst="$1" creds="$2" existing
  # List tags first: a transport/authentication failure is not an absent image.
  if skopeo list-tags --creds "${creds}" "docker://${dst%:*}" | \
      jq -e --arg tag "${dst##*:}" '.Tags | index($tag) != null' >/dev/null; then
    existing=$(skopeo inspect --creds "${creds}" --format '{{.Digest}}' "docker://${dst}")
    [[ "${existing}" == "${DIGEST}" ]] || { echo "ERROR: conflicting image ${dst}" >&2; return 1; }
  else
    # Check repository access separately, rather than treating any error as absent.
    skopeo list-tags --creds "${creds}" "docker://${dst%:*}" >/dev/null
    skopeo copy --all --preserve-digests --src-creds "iamapikey:${ICR_KEY}" --dest-creds "${creds}" \
      "docker://${SRC}" "docker://${dst}"
  fi
  [[ $(skopeo inspect --creds "${creds}" --format '{{.Digest}}' "docker://${dst}") == "${DIGEST}" ]]
}

case "${PROMOTION_PHASE:?}" in
  redhat)
    creds=$(get_secret redhat-registry)
    copy_version "quay.io/redhat-isv-containers/5e961c2c93604e02afa9ebdf:${VERSION}" \
      "$(jq -er '.username' <<< "${creds}"):$(jq -er '.password' <<< "${creds}")"
    ;;
  versioned)
    copy_version "icr.io/instana/instana-agent-operator:${VERSION}" "iamapikey:${ICR_KEY}"
    rel_creds=$(get_secret icr-rel-api-key)
    rel_username=$(jq -er '.username | strings | select(length > 0)' <<< "${rel_creds}")
    rel_password=$(jq -er '.password | strings | select(length > 0)' <<< "${rel_creds}")
    copy_version "icr.io/instana-rel/rel-docker-agent-local/instana-agent-operator:${VERSION}" \
      "${rel_username}:${rel_password}"
    ;;
  latest)
    # Public version labels guard against accidentally downgrading :latest.
    current=$(skopeo inspect --creds "iamapikey:${ICR_KEY}" \
      docker://icr.io/instana/instana-agent-operator:latest | jq -er '.Labels.version')
    python3 - "${current}" "${VERSION}" <<'PY'
import sys
from pathlib import Path
sys.path.insert(0, str(Path(__import__('os').environ['RELEASE_SCRIPTS'])))
from release_api import semver
old, new = map(semver, sys.argv[1:])
if old is None or new is None or old > new:
    raise SystemExit('Refusing an invalid or downgrading latest update')
PY
    skopeo copy --all --preserve-digests --src-creds "iamapikey:${ICR_KEY}" \
      --dest-creds "iamapikey:${ICR_KEY}" "docker://${SRC}" docker://icr.io/instana/instana-agent-operator:latest
    ;;
  *) exit 1 ;;
esac
