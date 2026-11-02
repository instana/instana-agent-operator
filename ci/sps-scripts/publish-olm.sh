#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
CREDS=$(get_secret github-instana-agent-build)
export GH_TOKEN
GH_TOKEN=$(jq -er '.token' <<< "${CREDS}")
python3 "${SCRIPT_DIR}/publish_olm.py" "${SELECTED_BOM_FILE}" "${PUBLIC_OLM_ZIP:-$(get_env repackaged-olm-zip)}" \
  "$(get_env olm-fork-owner instana)" "${WORKSPACE}/olm-prs.json"
