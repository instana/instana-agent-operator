#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Offline render/structural smoke test: no installation or registry writes.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
VERSION=$(jq -er '.version' "${SELECTED_BOM_FILE}")
TGZ=${PUBLIC_HELM_TGZ:-$(get_env repackaged-helm-tgz)}
YAML=${PUBLIC_OPERATOR_YAML:-$(get_env repackaged-operator-yaml)}
ZIP=${PUBLIC_OLM_ZIP:-$(get_env repackaged-olm-zip)}
OUT=$(mktemp -d "${WORKSPACE}/public-smoke-XXXXXX")
tar -xzf "${TGZ}" -C "${OUT}"
unzip -q "${ZIP}" -d "${OUT}/olm"
if grep -R -nE 'instana-agent-dev|OPERATOR_VERSION_PLACEHOLDER' "${OUT}" "${YAML}"; then
  echo 'ERROR: development reference/placeholder in public artifacts' >&2
  exit 1
fi
helm lint "${TGZ}"
helm template release-smoke "${TGZ}" > "${OUT}/rendered.yaml"
helm template release-smoke "${TGZ}" --set-json 'controllerManager.image=null' > "${OUT}/fallback.yaml"
python3 "${SCRIPT_DIR}/validate-public-artifacts.py" "${VERSION}" "${YAML}" "${OUT}" \
  "$(jq -er '."previous-version"' "${SELECTED_BOM_FILE}")"
echo "Public artifacts passed offline render and structural smoke tests"
