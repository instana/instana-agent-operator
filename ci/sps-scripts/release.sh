#!/usr/bin/env bash
set -euo pipefail
echo "===== release.sh - start ====="
echo "==== Step 1: Creating release page in IBM GitHub Enterprise ===="

SOURCE_DIRECTORY=$(git rev-parse --show-toplevel)
NEW_CHART_VERSION=$(cat "${SOURCE_DIRECTORY}/versions/INSTANA_AGENT_CHART_VERSION")

GITHUB_API_URL="https://github.ibm.com/api/v3"
GITHUB_TOKEN=$(get_env release-github-enterprise-token)
REPO_OWNER="instana"
REPO_NAME="instana-agent-charts"
TAG_NAME="${NEW_CHART_VERSION}"
RELEASE_NAME="${NEW_CHART_VERSION}"
TARBALL_PATH="$(ls artefacts/instana-agent-*.tgz)"
TARGET_COMMIT=$(git rev-parse HEAD)

# see: https://docs.github.com/en/rest/releases/releases?apiVersion=2022-11-28
# create the release
RELEASE_RESPONSE=$(curl -s -L -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -d "{\"tag_name\":\"${TAG_NAME}\",\"target_commitish\":\"${TARGET_COMMIT}\", \"name\":\"${RELEASE_NAME}\",\"generate_release_notes\": true,\"draft\":false,\"prerelease\":false}" \
  "${GITHUB_API_URL}/repos/${REPO_OWNER}/${REPO_NAME}/releases")

# extract the upload url from the response
UPLOAD_URL=$(echo "${RELEASE_RESPONSE}" | jq -r ".upload_url")
UPLOAD_URL=${UPLOAD_URL%\{?name,label\}}

# upload helm chart
FILENAME=$(basename "${TARBALL_PATH}")
curl -s -L -X POST \
  -H "Accept: application/vnd.github+json" \
  -H "Authorization: Bearer ${GITHUB_TOKEN}" \
  -H "X-GitHub-Api-Version: 2022-11-28" \
  -H "Content-Type: application/octet-stream" \
  --data-binary @"${TARBALL_PATH}" \
  "${UPLOAD_URL}?name=${FILENAME}"

echo
echo "Release created and file uploaded successfully!"

echo "==== Step 2: Publish new helm chart to public GitHub repo ===="

PUBLIC_GITHUB_CREDENTIALS=$(get_env github-instana-agent-build)
PUBLIC_GITHUB_USERNAME=$(echo "${PUBLIC_GITHUB_CREDENTIALS}" | jq -r ".username")
PUBLIC_GITHUB_EMAIL=$(echo "${PUBLIC_GITHUB_CREDENTIALS}" | jq -r ".email")
PUBLIC_GITHUB_TOKEN=$(echo "${PUBLIC_GITHUB_CREDENTIALS}" | jq -r ".token")

rm -rf helm-charts
git clone --depth=1 "https://${PUBLIC_GITHUB_TOKEN}@github.com/instana/helm-charts.git" helm-charts
rm -rf "${SOURCE_DIRECTORY}/helm-charts/instana-agent"
mkdir "${SOURCE_DIRECTORY}/helm-charts/instana-agent"
tar -xzvf "${SOURCE_DIRECTORY}"/artefacts/instana-agent-*.tgz -C "${SOURCE_DIRECTORY}/helm-charts/instana-agent" --strip-components=1
cd "${SOURCE_DIRECTORY}/helm-charts"
if git diff --quiet; then
  echo "No changes detected. Exiting."
else
  echo "Changes detected, continue to commit and push them"
  git config --global "user.email" "${PUBLIC_GITHUB_EMAIL}"
  git config --global "user.name" "${PUBLIC_GITHUB_USERNAME}"
  git add .
  git commit -m "Instana-agent chart version ${NEW_CHART_VERSION}"
  git push origin main
fi

echo "==== Step 3: Push update to GCP bucket: gs://agents.instana.io/helm/index.yaml ===="
cd "${SOURCE_DIRECTORY}"
echo "Authenticating with gcloud"
get_env gcp-service-account > keyfile.json
gcloud auth activate-service-account --key-file keyfile.json
BUCKET="agents.instana.io"
echo "Retrieving current helm repository index from bucket ${BUCKET}"
gsutil cp "gs://${BUCKET}/helm/index.yaml" index-current.yaml

echo "Updating the repository index"
rm -rf repository-packaged-charts
mkdir repository-packaged-charts

cp artefacts/instana-agent-*.tgz repository-packaged-charts/
helm repo index repository-packaged-charts/ --url "https://agents.instana.io/helm/" --merge index-current.yaml

echo "Uploading the repository in gs://${BUCKET}/helm/ ... "
gsutil cp repository-packaged-charts/* "gs://${BUCKET}/helm/"

echo "==== Step 4: Push update to Artifactory https://delivery.instana.io/artifactory/rel-helm-agent-local/instana-agent-${NEW_CHART_VERSION}.tgz ===="
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
DELIVERY_RELEASE_ARTIFACTORY_USERNAME=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".username")
DELIVERY_RELEASE_ARTIFACTORY_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")
helm repo add helm-agent https://delivery.instana.io/artifactory/rel-helm-agent-local --username "$DELIVERY_RELEASE_ARTIFACTORY_USERNAME" --password "$DELIVERY_RELEASE_ARTIFACTORY_PASSWORD"
helm repo update
helm search repo helm-agent
helm push rel-helm-agent-local "${SOURCE_DIRECTORY}"/artefacts/instana-agent-"${NEW_CHART_VERSION}".tgz
helm repo update
helm search repo helm-agent

echo "==== Step 5: Bump versions/INSTANA_AGENT_CHART_VERSION with "[skip ci]" commit message and push to GitHub Enterprise ===="
cd "${SOURCE_DIRECTORY}"
NEW_VERSION=$(awk -F. '{$NF = $NF + 1;} 1' OFS=. versions/INSTANA_AGENT_CHART_VERSION) && echo "${NEW_VERSION}" > versions/INSTANA_AGENT_CHART_VERSION
git config --global "user.email" "instana.ibm.github.enterprise@ibm.com"
git config --global "user.name" "Instana-IBM-GitHub-Enterprise"
git add versions/INSTANA_AGENT_CHART_VERSION
git commit -m "[skip ci] Bump the instana-agent chart version to ${NEW_VERSION}"
git push origin "${BRANCH}"

echo "===== release.sh - end ====="