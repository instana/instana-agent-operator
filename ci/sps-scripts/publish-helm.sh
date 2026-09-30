#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
# Step: publish (inside publish-helm-and-release stage, runs after repackage-helm.sh)
# In sequence:
#   1. Push git tags on operator repo (v2.2.22) and charts repo (2.2.22)
#   2. Publish repackaged helm chart to github.com/instana/helm-charts, GCP bucket, TaaS Artifactory
#   3. Create public GitHub release on github.com/instana/instana-agent-operator
#      attaching olm zip, instana-agent-operator.yaml, and helm chart tarball
#
set -euo pipefail
if [[ "${PIPELINE_DEBUG:-0}" == 1 ]]; then
  trap env EXIT
  set -x
fi
echo "===== publish-helm.sh - start ====="

# ---------------------------------------------------------------------------
# Read BOM and promoted helm path
# ---------------------------------------------------------------------------
COS_BUCKET=$(get_env cos-bucket-name)
RELEASE_STATE_PATH=$(get_env release-state-path "pipeline-state/latest-release-candidate/bom.json")
HELM_TGZ=$(get_env repackaged-helm-tgz)

BOM_FILE=$(mktemp)
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${RELEASE_STATE_PATH}" \
  --output json "${BOM_FILE}"

VERSION=$(jq -r '.version'                             "${BOM_FILE}")
RELEASE_TAG=$(jq -r '."release-tag"'                   "${BOM_FILE}")
COS_STATE_PATH=$(jq -r '."cos-state-path"'             "${BOM_FILE}")
OPERATOR_SHA=$(jq -r '.sources."operator-sha"'         "${BOM_FILE}")
CHARTS_SHA=$(jq -r '.sources."charts-sha"'             "${BOM_FILE}")
OLM_BUNDLE_COS=$(jq -r '.artifacts."olm-bundle"'       "${BOM_FILE}")
OPERATOR_YAML_COS=$(jq -r '.artifacts."operator-yaml"' "${BOM_FILE}")
rm -f "${BOM_FILE}"

echo "version=${VERSION}  release-tag=${RELEASE_TAG}"
echo "operator-sha=${OPERATOR_SHA}  charts-sha=${CHARTS_SHA}"
echo "helm-tgz=${HELM_TGZ}"

# ---------------------------------------------------------------------------
# Download OLM zip and operator YAML from COS
# ---------------------------------------------------------------------------
OLM_ZIP="${WORKSPACE}/olm-${VERSION}.zip"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${OLM_BUNDLE_COS}" \
  --output json "${OLM_ZIP}"

OPERATOR_YAML="${WORKSPACE}/instana-agent-operator.yaml"
ibmcloud cos get-object \
  --bucket "${COS_BUCKET}" \
  --key "${OPERATOR_YAML_COS}" \
  --output json "${OPERATOR_YAML}"

# ---------------------------------------------------------------------------
# Credentials
# ---------------------------------------------------------------------------
GH_TOKEN=$(get_secret github-token)

PUBLIC_GITHUB_CREDENTIALS=$(get_secret github-instana-agent-build)
PUBLIC_GITHUB_USERNAME=$(echo "${PUBLIC_GITHUB_CREDENTIALS}" | jq -r ".username")
PUBLIC_GITHUB_EMAIL=$(echo "${PUBLIC_GITHUB_CREDENTIALS}" | jq -r ".email")
PUBLIC_GITHUB_TOKEN=$(echo "${PUBLIC_GITHUB_CREDENTIALS}" | jq -r ".token")

# ---------------------------------------------------------------------------
# 1. Push git tags
# ---------------------------------------------------------------------------
echo "=== Pushing git tags ==="

# Operator repo tag: v2.2.22
git -c "http.extraheader=Authorization: Bearer ${GH_TOKEN}" \
  clone --depth 1 \
  "https://x-access-token:${GH_TOKEN}@github.ibm.com/instana/instana-agent-operator.git" \
  op-tag-work
git -C op-tag-work config user.email "instana.ibm.github.enterprise@ibm.com"
git -C op-tag-work config user.name  "Instana-IBM-GitHub-Enterprise"
git -C op-tag-work fetch --depth 1 origin "${OPERATOR_SHA}"
git -C op-tag-work tag "${RELEASE_TAG}" "${OPERATOR_SHA}"
git -C op-tag-work push origin "${RELEASE_TAG}"
echo "Pushed operator tag ${RELEASE_TAG} at ${OPERATOR_SHA}"

# Charts repo tag: 2.2.22 (no 'v' prefix)
git clone --depth 1 \
  "https://x-access-token:${GH_TOKEN}@github.ibm.com/instana/instana-agent-charts.git" \
  charts-tag-work
git -C charts-tag-work config user.email "instana.ibm.github.enterprise@ibm.com"
git -C charts-tag-work config user.name  "Instana-IBM-GitHub-Enterprise"
git -C charts-tag-work fetch --depth 1 origin "${CHARTS_SHA}"
git -C charts-tag-work tag "${VERSION}" "${CHARTS_SHA}"
git -C charts-tag-work push origin "${VERSION}"
echo "Pushed charts tag ${VERSION} at ${CHARTS_SHA}"

# ---------------------------------------------------------------------------
# 2a. Publish helm chart → github.com/instana/helm-charts
# ---------------------------------------------------------------------------
echo "=== Publishing helm chart to github.com/instana/helm-charts ==="
rm -rf helm-charts-public
git clone --depth 1 \
  "https://x-access-token:${PUBLIC_GITHUB_TOKEN}@github.com/instana/helm-charts.git" \
  helm-charts-public

rm -rf helm-charts-public/instana-agent
mkdir helm-charts-public/instana-agent
tar -xzf "${HELM_TGZ}" -C helm-charts-public/instana-agent --strip-components=1

cd helm-charts-public
git config user.email "${PUBLIC_GITHUB_EMAIL}"
git config user.name  "${PUBLIC_GITHUB_USERNAME}"
if git diff --quiet; then
  echo "No changes to helm-charts repo — skipping commit"
else
  git add .
  git commit -m "Instana-agent operator chart version ${VERSION}"
  git push origin main
  echo "Pushed helm chart to helm-charts repo"
fi
cd "${WORKSPACE}"

# ---------------------------------------------------------------------------
# 2b. Publish helm chart → GCP bucket gs://agents.instana.io/helm/
# ---------------------------------------------------------------------------
echo "=== Publishing helm chart to GCP bucket ==="
get_secret gcp-bucket-write-agents-io-service-key > keyfile.json
gcloud auth activate-service-account --key-file keyfile.json
BUCKET="agents.instana.io"

gsutil cp "gs://${BUCKET}/helm/index.yaml" index-current.yaml

rm -rf repo-charts
mkdir repo-charts
cp "${HELM_TGZ}" repo-charts/
helm repo index repo-charts/ --url "https://agents.instana.io/helm/" --merge index-current.yaml
gsutil cp repo-charts/* "gs://${BUCKET}/helm/"
echo "Pushed helm chart to GCP bucket"

# ---------------------------------------------------------------------------
# 2c. Publish helm chart → TaaS Artifactory
# ---------------------------------------------------------------------------
echo "=== Publishing helm chart to TaaS Artifactory ==="
TAAS_CREDS=$(get_secret taas-artifactory-read-writer-creds)
TAAS_USER=$(echo "${TAAS_CREDS}" | jq -r ".username")
TAAS_PASS=$(echo "${TAAS_CREDS}" | jq -r ".password")

curl_fail_with_body() {
  curl -o - -w "\n%{http_code}\n" "$@" | \
    awk '{l[NR]=$0} END {for(i=1;i<=NR-1;i++) print l[i]}; END{if($0<200||$0>299) exit 1}'
}

curl_fail_with_body \
  -u "${TAAS_USER}:${TAAS_PASS}" \
  -T "${HELM_TGZ}" \
  "https://na.artifactory.swg-devops.com/artifactory/instana-team-release-agent-helm-local/instana-agent-${VERSION}.tgz"
echo "Pushed helm chart to TaaS Artifactory"

# ---------------------------------------------------------------------------
# 3. Create public GitHub release on github.com/instana/instana-agent-operator
#    Tags must already exist (pushed in step 1).
# ---------------------------------------------------------------------------
echo "=== Creating public GitHub release ${RELEASE_TAG} ==="

HELM_TGZ_BASENAME=$(basename "${HELM_TGZ}")
OLM_ZIP_BASENAME="olm-${VERSION}.zip"
OPERATOR_YAML_BASENAME="instana-agent-operator.yaml"

RELEASE_RESPONSE=$(curl --silent --fail --show-error -X POST \
  -H "Authorization: Bearer ${PUBLIC_GITHUB_TOKEN}" \
  -H "Accept: application/vnd.github+json" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d "{\"tag_name\":\"${RELEASE_TAG}\",\"target_commitish\":\"${OPERATOR_SHA}\",\"name\":\"${RELEASE_TAG}\",\"draft\":false,\"prerelease\":false,\"generate_release_notes\":true}" \
  "https://api.github.com/repos/instana/instana-agent-operator/releases")

UPLOAD_URL=$(echo "${RELEASE_RESPONSE}" | jq -r ".upload_url" | sed 's/{.*}//')

upload_asset() {
  local file="$1"
  local name="$2"
  echo "Uploading ${name}..."
  curl --silent --fail --show-error -X POST \
    -H "Authorization: Bearer ${PUBLIC_GITHUB_TOKEN}" \
    -H "Accept: application/vnd.github+json" \
    -H "X-GitHub-Api-Version: 2022-11-28" \
    -H "Content-Type: application/octet-stream" \
    --data-binary @"${file}" \
    "${UPLOAD_URL}?name=${name}"
  echo "Uploaded ${name}"
}

upload_asset "${OLM_ZIP}"      "${OLM_ZIP_BASENAME}"
upload_asset "${OPERATOR_YAML}" "${OPERATOR_YAML_BASENAME}"
upload_asset "${HELM_TGZ}"     "${HELM_TGZ_BASENAME}"

echo "Public GitHub release created: https://github.com/instana/instana-agent-operator/releases/tag/${RELEASE_TAG}"

echo "===== publish-helm.sh - end ====="
