#!/bin/bash
set -euo pipefail
echo "Building instana-agent helm chart..."

# Previously we build rancher charts and vanilla helm charts, but rancher
# switched their approaches and instead of allowing pushes of special charts
# is hosting their own overlay files now in https://github.com/rancher/partner-charts/tree/main-source/packages/instana/instana-agent/overlay

SOURCE_DIRECTORY=$(git rev-parse --show-toplevel)
TARGET_DIRECTORY=artefacts
NEW_CHART_VERSION=$(cat "${SOURCE_DIRECTORY}/versions/INSTANA_AGENT_CHART_VERSION")
HELM_APP_VERSION=$(cat "${SOURCE_DIRECTORY}/versions/INSTANA_AGENT_APP_VERSION")

cd "${SOURCE_DIRECTORY}"
rm -rf "${TARGET_DIRECTORY}"
mkdir -p "${TARGET_DIRECTORY}"

CHART_TARGET_DIRECTORY="${TARGET_DIRECTORY}/helm-charts"

echo "Copying changes from canonical to ${CHART_TARGET_DIRECTORY}"
rsync -avr "${SOURCE_DIRECTORY}/canonical/." "${CHART_TARGET_DIRECTORY}"

echo "Injecting versions in Chart.yaml"
yq eval -i ".appVersion = \"${HELM_APP_VERSION}\"" "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/Chart.yaml"
yq eval -i ".version = \"${NEW_CHART_VERSION}\"" "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/Chart.yaml"


echo "Downloading operator release yaml"
rm -rf operator-download
mkdir -p operator-download
pushd operator-download
curl -L https://github.com/instana/instana-agent-operator/releases/latest/download/instana-agent-operator.yaml | yq  -s '"operator_" + .kind + "_" + .metadata.name'
echo "Extracting CRD"
mkdir -p "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/crds"
# ensure to use lowercase letters for all filenames
for file in *; do
  mv "$file" "$(echo "$file" | tr '[:upper:]' '[:lower:]')"
done
mv ./*customresourcedefinition* "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/crds/operator_customresourcedefinition_agents_instana_io.yml"

# fetch current value of container image from the operator release and use it for default values.yaml
OPERARTOR_IMAGE=$(yq ".spec.template.spec.containers[0].image" "./operator_deployment_instana-agent-controller-manager.yml")
echo "OPERARTOR_IMAGE=${OPERARTOR_IMAGE}"
OPERARTOR_REPO=$(echo "${OPERARTOR_IMAGE}" | awk -F '[:]' '{print $1}')
echo "OPERARTOR_REPO=${OPERARTOR_REPO}"
OPERARTOR_TAG=$(echo "${OPERARTOR_IMAGE}" | awk -F '[:]' '{print $2}')
echo "OPERARTOR_TAG=${OPERARTOR_TAG}"
yq eval -i ".controllerManager.image.name = \"${OPERARTOR_REPO}\"" "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/values.yaml"
yq eval -i ".controllerManager.image.tag = \"${OPERARTOR_TAG}\"" "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/values.yaml"
sed -i "s/instana-agent-operator:latest/instana-agent-operator:${OPERARTOR_TAG}/g" "${SOURCE_DIRECTORY}/${CHART_TARGET_DIRECTORY}/templates/operator_deployment_instana-agent-controller-manager.yml"

popd
rm -rf operator-download

helm package "${CHART_TARGET_DIRECTORY}/." \
    --version "${NEW_CHART_VERSION}" \
    --app-version "${HELM_APP_VERSION}" \
    --destination "${TARGET_DIRECTORY}/"

helm lint ${TARGET_DIRECTORY}/instana-agent-*.tgz

echo "Bundled operator version: ${OPERARTOR_IMAGE}"