#!/bin/bash

set -e

VERSION=$1
GITHUB_OAUTH_TOKEN=$2

if [[ -z ${VERSION} ]] || [[ -z ${GITHUB_OAUTH_TOKEN} ]]; then
  echo "Please ensure VERSION and GITHUB_OAUTH_TOKEN are set to continue"
  exit 1
fi

ROOT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" >/dev/null 2>&1 && pwd )/../.."
TARGET_DIR="$ROOT_DIR/target"
DOWNLOAD_DIR="$TARGET_DIR/downloads/$VERSION"
GITHUB_RELEASES_URL="https://api.github.com/repos/instana/instana-agent-operator/releases"

mkdir -p ${DOWNLOAD_DIR}

function github_curl() {
  curl -H "Authorization: token $GITHUB_OAUTH_TOKEN" \
       -H "Accept: application/vnd.github.v3.raw" \
       $@
}

printf "%s\n" "Checking if release v${VERSION} exists..."
GITHUB_RELEASE_RESPONSE=$(github_curl ${GITHUB_RELEASES_URL}/tags/v${VERSION})

GITHUB_RELEASE_ID=$(echo ${GITHUB_RELEASE_RESPONSE} | jq .id)
if [[ -z "${GITHUB_RELEASE_ID}" ]] || [[ ${GITHUB_RELEASE_ID} == "null" ]]; then
  printf "\n%s" "GitHub Release ${GITHUB_RELEASE_ID} does not exists. Cannot download assets."
  exit 0
fi

ASSET_URLS=$(echo ${GITHUB_RELEASE_RESPONSE} | jq -r ".assets | .[].browser_download_url")

# convert multiline string to array
SAVEIFS=$IFS              # Save current IFS
IFS=$'\n'                 # Change IFS to new line
ASSET_URLS=($ASSET_URLS)  # split to array
IFS=$SAVEIFS              # Restore IFS

function download_github_asset() {
  printf "%s\n" "Downloading asset ${1}"
  wget -q --auth-no-challenge \
    --header='Accept:application/octet-stream' \
    -P ${DOWNLOAD_DIR} \
    ${1}
}

printf "%s\n" "Downloading assets to ${DOWNLOAD_DIR}"
for (( i=0; i<${#ASSET_URLS[@]}; i++ ))
do
  download_github_asset ${ASSET_URLS[$i]}
done
