#!/bin/bash
#
# (c) Copyright IBM Corp. 2021, 2024
# (c) Copyright Instana Inc.
#


set -e

VERSION=$1
GITHUB_OAUTH_TOKEN=$2
TARGET_DIR=$3

if [[ -z ${VERSION} ]] || [[ -z ${GITHUB_OAUTH_TOKEN} ]]; then
  echo "Please ensure VERSION and GITHUB_OAUTH_TOKEN are set so a GitHub Release can be created"
  exit 1
fi

OPERATOR_RESOURCE_FILENAME="instana-agent-operator.yaml"
GITHUB_RELEASES_URL="https://api.github.com/repos/instana/instana-agent-operator/releases"

printf "%s\n" "Checking if release v${VERSION} exists..."
GITHUB_RELEASE_RESPONSE=$(curl -X GET \
  -H "Authorization: token $GITHUB_OAUTH_TOKEN" \
  ${GITHUB_RELEASES_URL}/tags/v"${VERSION}")

GITHUB_RELEASE_ID=$(echo "${GITHUB_RELEASE_RESPONSE}" | jq .id)
if [[ -z "${GITHUB_RELEASE_ID}" ]] || [[ ${GITHUB_RELEASE_ID} == "null" ]]; then
  printf "\n%s" "Creating GitHub Release..."
  GITHUB_RELEASE_RESPONSE=$(curl -X POST \
    -H "Authorization: token $GITHUB_OAUTH_TOKEN" \
    -H 'Content-Type: application/json' \
    -d "{ \"tag_name\": \"v${VERSION}\", \"target_commitish\": \"main\", \"name\": \"v${VERSION}\" }" \
    ${GITHUB_RELEASES_URL})

  GITHUB_RELEASE_ID=$(echo "${GITHUB_RELEASE_RESPONSE}" | jq .id)
  if [[ -z "${GITHUB_RELEASE_ID}" ]] || [[ ${GITHUB_RELEASE_ID} == "null" ]]; then
    echo "Unable to determine GitHub Release id. Please check on https://github.com/instana/instana-agent-operator/releases if it was created"
    exit 0
  fi
fi

upload_github_asset() {
  local asset_file=$1
  local asset_filename=$2
  if [[ ! -f ${asset_file} ]]; then
    echo "${asset_file} not found. Unable to upload asset to Github Release ${GITHUB_RELEASE_ID}"
  else
    printf "\n%s" "Uploading ${asset_file} to Github Release ${GITHUB_RELEASE_ID}..."
    curl -X POST \
      -H "Authorization: token $GITHUB_OAUTH_TOKEN" \
      -H 'Content-Type: text/x-yaml' \
      --data-binary @"${asset_file}" \
      https://uploads.github.com/repos/instana/instana-agent-operator/releases/"${GITHUB_RELEASE_ID}"/assets?name="${asset_filename}"
  fi
}

upload_github_asset "${TARGET_DIR}/${OPERATOR_RESOURCE_FILENAME}" "${OPERATOR_RESOURCE_FILENAME}"
upload_github_asset "${TARGET_DIR}/olm-$VERSION.zip" "olm-$VERSION.zip"
upload_github_asset "${TARGET_DIR}/images.txt" "images.txt"
