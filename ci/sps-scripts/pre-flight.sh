#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail
# strip the leading "v" from the operator version for release:

PREFIX="v"
VERSION=$(cat "version/INSTANA_AGENT_OPERATOR_VERSION")
OPERATOR_DOCKER_VERSION=${VERSION#"$PREFIX"}

RED_HAT_REGISTRY_CREDENTIALS=$(get_env RED_HAT_REGISTRY)
RED_HAT_REGISTRY_USERNAME=$(echo "${RED_HAT_REGISTRY_CREDENTIALS}" | jq -r ".username")
RED_HAT_REGISTRY_PASSWORD=$(echo "${RED_HAT_REGISTRY_CREDENTIALS}" | jq -r ".password")
RED_HAT_API_TOKEN=$(echo "${RED_HAT_REGISTRY_CREDENTIALS}" | jq -r ".token")

# Run Preflight Image Scans for RH Marketplace

RED_HAT_PROJECT_ID=5e961c2c93604e02afa9ebdf
RED_HAT_REGISTRY="quay.io/redhat-isv-containers/${RED_HAT_PROJECT_ID}"
skopeo login -u ${RED_HAT_REGISTRY_USERNAME} -p ${RED_HAT_REGISTRY_PASSWORD} --authfile $(pwd)/auth.json quay.io
DOCKER_CFG_FILE="$(pwd)/auth.json"

# Download preflight tool if it doesn't exist
if [ ! -f "preflight-linux-amd64" ]; then
  echo "Downloading preflight tool..."
  PREFLIGHT_VERSION="1.14.1"
  curl -L -o preflight-linux-amd64 "https://github.com/redhat-openshift-ecosystem/openshift-preflight/releases/download/${PREFLIGHT_VERSION}/preflight-linux-amd64"
fi

chmod +x preflight-linux-amd64

./preflight-linux-amd64 check container --artifacts preflight-output "$RED_HAT_REGISTRY:$OPERATOR_DOCKER_VERSION" --certification-project-id=$RED_HAT_PROJECT_ID --docker-config $DOCKER_CFG_FILE --submit --pyxis-api-token=$RED_HAT_API_TOKEN
