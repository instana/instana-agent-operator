#!/usr/bin/env bash
# (c) Copyright IBM Corp. 2026
# Resumable publication. Existing bytes/tags are accepted only if identical.
set -euo pipefail
SCRIPT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
source "${SCRIPT_DIR}/release-common.sh"
BOM_FILE="${SELECTED_BOM_FILE:?}"
VERSION=$(jq -er '.version' "${BOM_FILE}")
HELM_TGZ=${PUBLIC_HELM_TGZ:-$(get_env repackaged-helm-tgz)}
OLM_ZIP=${PUBLIC_OLM_ZIP:-$(get_env repackaged-olm-zip)}
OPERATOR_YAML=${PUBLIC_OPERATOR_YAML:-$(get_env repackaged-operator-yaml)}
export GH_TOKEN GHE_TOKEN
GH_TOKEN=$(get_secret github-token)
GHE_TOKEN=$(get_secret ghe-token)
python3 "${SCRIPT_DIR}/release_api.py" tags "${BOM_FILE}"

PUB_DIR=$(mktemp -d "${WORKSPACE}/publish-XXXXXX")
chmod 700 "${PUB_DIR}"
get_secret gcp-bucket-write-agents-io-service-key > "${PUB_DIR}/gcp-key.json"
trap 'rm -f "${PUB_DIR}/gcp-key.json" "${PUB_DIR}/askpass"' EXIT
gcloud auth activate-service-account --key-file "${PUB_DIR}/gcp-key.json"

# Read the index first with its object generation; conditional upload detects
# concurrent writers rather than silently dropping their new versions.
INDEX=gs://agents.instana.io/helm/index.yaml
GENERATION=$(gsutil stat "${INDEX}" | awk '/Generation:/ {print $2}')
[[ "${GENERATION}" =~ ^[0-9]+$ ]]
gsutil cp "${INDEX}#${GENERATION}" "${PUB_DIR}/index-current.yaml"
DEST="gs://agents.instana.io/helm/$(basename "${HELM_TGZ}")"
if gsutil -q stat "${DEST}"; then
  gsutil cp "${DEST}" "${PUB_DIR}/existing.tgz"
  cmp "${HELM_TGZ}" "${PUB_DIR}/existing.tgz"
else
  gsutil -h 'x-goog-if-generation-match:0' cp "${HELM_TGZ}" "${DEST}"
fi
mkdir "${PUB_DIR}/repo"
cp "${HELM_TGZ}" "${PUB_DIR}/repo/"
helm repo index "${PUB_DIR}/repo" --url https://agents.instana.io/helm/ --merge "${PUB_DIR}/index-current.yaml"
gsutil -h "x-goog-if-generation-match:${GENERATION}" cp "${PUB_DIR}/repo/index.yaml" "${INDEX}"

TAAS_CREDS=$(get_secret taas-artifactory-read-writer-creds)
TAAS_AUTH="$(jq -er '.username' <<< "${TAAS_CREDS}"):$(jq -er '.password' <<< "${TAAS_CREDS}")"
URL="https://na.artifactory.swg-devops.com/artifactory/instana-team-release-agent-helm-local/instana-agent-${VERSION}.tgz"
CODE=$(curl --silent --show-error -u "${TAAS_AUTH}" -o "${PUB_DIR}/taas.tgz" -w '%{http_code}' "${URL}")
case "${CODE}" in
  200) cmp "${HELM_TGZ}" "${PUB_DIR}/taas.tgz" ;;
  404) curl --fail --silent --show-error -u "${TAAS_AUTH}" -T "${HELM_TGZ}" "${URL}" ;;
  *) echo "ERROR: Artifactory read failed: HTTP ${CODE}" >&2; exit 1 ;;
esac

# Credentials stay out of git URLs, persisted configs and process tracing.
cat > "${PUB_DIR}/askpass" <<'SH'
#!/usr/bin/env bash
case "$1" in *Username*) echo x-access-token ;; *) echo "${GH_TOKEN}" ;; esac
SH
chmod 700 "${PUB_DIR}/askpass"
export GIT_ASKPASS="${PUB_DIR}/askpass" GIT_TERMINAL_PROMPT=0
git clone --depth 1 https://github.com/instana/helm-charts.git "${PUB_DIR}/charts"
CHART_DIR="${PUB_DIR}/charts/instana-agent"
# Only our disposable clone is modified.
rm -rf "${CHART_DIR}"
mkdir "${CHART_DIR}"
tar -xzf "${HELM_TGZ}" -C "${CHART_DIR}" --strip-components=1
git -C "${PUB_DIR}/charts" config user.email "instanacd@instana.com"
git -C "${PUB_DIR}/charts" config user.name "Instana CD"
git -C "${PUB_DIR}/charts" add .
if ! git -C "${PUB_DIR}/charts" diff --cached --quiet; then
  git -C "${PUB_DIR}/charts" commit -s -m "chore: publish Instana agent chart ${VERSION}"
  git -C "${PUB_DIR}/charts" push origin HEAD:main
fi
# All assets are uploaded/verified while draft. Reruns never replace public bytes.
python3 "${SCRIPT_DIR}/release_api.py" release "${BOM_FILE}" "${OLM_ZIP}" "${OPERATOR_YAML}" "${HELM_TGZ}"
