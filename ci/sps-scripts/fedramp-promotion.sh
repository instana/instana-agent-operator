#!/bin/bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail

ARTIFACTORY_USERNAME_AND_PASSWORD_RELEASE=$(get_env ARTIFACTORY_USERNAME_AND_PASSWORD_RELEASE)
ARTIFACTORY_USERNAME_RELEASE=$(echo "${ARTIFACTORY_USERNAME_AND_PASSWORD_RELEASE}" | jq -r ".username")
ARTIFACTORY_PASSWORD_RELEASE=$(echo "${ARTIFACTORY_USERNAME_AND_PASSWORD_RELEASE}" | jq -r ".password")

ARTIFACTORY_GENERIC_FEDRAMP_URL=$(get_env ARTIFACTORY_GENERIC_FEDRAMP_URL)
ARTIFACTORY_CONTAINER_FEDRAMP_DOCKER_URL=$(get_env ARTIFACTORY_CONTAINER_FEDRAMP_DOCKER_URL)
ARTIFACTORY_CONTAINER_DOCKER_URL=$(get_env ARTIFACTORY_CONTAINER_DOCKER_URL)

RELEASE_ARTIFACTORY_REPOSITORY="${ARTIFACTORY_CONTAINER_DOCKER_URL}/instana-agent-operator"
FEDRAMP_ARTIFACTORY_REPOSITORY="${ARTIFACTORY_CONTAINER_FEDRAMP_DOCKER_URL}/instana-agent-operator"

DOMAIN=$(echo $ARTIFACTORY_GENERIC_FEDRAMP_URL | cut -d'/' -f1)
REPO_PATH=$(echo $ARTIFACTORY_GENERIC_FEDRAMP_URL | cut -d'/' -f2-)
FEDRAMP_GENERIC="$DOMAIN/artifactory/$REPO_PATH"
FEDRAMP_GENERIC_REPOSITORY="${FEDRAMP_GENERIC}/instana-agent-operator"

export ARTIFACT_VERSION
$WORKSPACE/$APP_REPO_FOLDER/ci/scripts/get-latest-fedramp-release-version.sh


echo "==== Step 5: Bump version/FEDRAMP_VERSION with "[skip ci]" commit message and push to GitHub Enterprise ===="
cd "${SOURCE_DIRECTORY}"
NEW_VERSION=$(awk -F. '{$NF = $NF + 1;} 1' OFS=. version/FEDRAMP_VERSION) && echo "${NEW_VERSION}" > version/FEDRAMP_VERSION
git config --global "user.email" "instana.ibm.github.enterprise@ibm.com"
git config --global "user.name" "Instana-IBM-GitHub-Enterprise"
git add version/FEDRAMP_VERSION
git commit -m "[skip ci] Bump the fedramp version to ${NEW_VERSION}"
git push origin "${BRANCH}"

FEDRAMP_VERSION=$(cat version/FEDRAMP_VERSION)

# Check if ARTIFACT_VERSION is set
if [ -z "$ARTIFACT_VERSION" ]; then
echo "Error: ARTIFACT_VERSION is not set. Failed to get the latest FedRAMP release version."
exit 1
fi

# Check if FEDRAMP_VERSION is set
if [ -z "$FEDRAMP_VERSION" ]; then
echo "Error: FEDRAMP_VERSION is not set. Failed to read version from fedramp-version/number."
exit 1
fi

# strip the leading "v" from the operator version for release:
export PREFIX="v"
export OPERATOR_DOCKER_VERSION=${ARTIFACT_VERSION#"$PREFIX"}
COMBINED_VERSION=${OPERATOR_DOCKER_VERSION}.fedramp-${FEDRAMP_VERSION}

# Validate OPERATOR_DOCKER_VERSION before using it
if [ -z "$OPERATOR_DOCKER_VERSION" ]; then
echo "Error: OPERATOR_DOCKER_VERSION is not set. Cannot proceed with artifact promotion."
exit 1
fi

# Validate COMBINED_VERSION before using it
if [ -z "$COMBINED_VERSION" ]; then
echo "Error: COMBINED_VERSION is not set. Cannot proceed with artifact promotion."
exit 1
fi

echo "Using OPERATOR_DOCKER_VERSION=$OPERATOR_DOCKER_VERSION and COMBINED_VERSION=$COMBINED_VERSION"

echo "---> Pushing multi-architectural manifest to ${FEDRAMP_ARTIFACTORY_REPOSITORY}"
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_RELEASE} \
--src-password ${ARTIFACTORY_PASSWORD_RELEASE} \
--dest-username ${ARTIFACTORY_USERNAME_RELEASE} \
--dest-password ${ARTIFACTORY_PASSWORD_RELEASE} \
docker://${RELEASE_ARTIFACTORY_REPOSITORY}:${OPERATOR_DOCKER_VERSION} \
docker://${FEDRAMP_ARTIFACTORY_REPOSITORY}:${COMBINED_VERSION}

echo "---> Pushing multi-architectural manifest to ${FEDRAMP_ARTIFACTORY_REPOSITORY} with latest tag"
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_RELEASE} \
--src-password ${ARTIFACTORY_PASSWORD_RELEASE} \
--dest-username ${ARTIFACTORY_USERNAME_RELEASE} \
--dest-password ${ARTIFACTORY_PASSWORD_RELEASE} \
docker://${RELEASE_ARTIFACTORY_REPOSITORY}:${OPERATOR_DOCKER_VERSION} \
docker://${FEDRAMP_ARTIFACTORY_REPOSITORY}:latest

echo "Pull the released manifest and replace icr.io image with FedRAMP artifactory image"
VERSION=${OPERATOR_DOCKER_VERSION} COMBINED_VERSION=${COMBINED_VERSION} $WORKSPACE/$APP_REPO_FOLDER/ci/scripts/fedramp-manifest-converter.sh

MANIFEST_FILE_FEDRAMP="instana-agent-operator.yaml"
echo "Verify the size of the instana-agent-operator.yaml before pushing"
if [ -s $MANIFEST_FILE_FEDRAMP ]; then
echo "File exists and it's not empty"
else
echo "Download failed or file is empty. Aborting"
exit 1
fi

echo "---> Pushing operator YAML to ${FEDRAMP_GENERIC_REPOSITORY} with $COMBINED_VERSION"
status_code=$(curl --silent --output /dev/stderr --write-out "%{http_code}" -u "${ARTIFACTORY_USERNAME_RELEASE}:${ARTIFACTORY_PASSWORD_RELEASE}" -X PUT "https://$FEDRAMP_GENERIC_REPOSITORY/$COMBINED_VERSION/instana-agent-operator.yaml" -T "${MANIFEST_FILE_FEDRAMP}")
echo # curl doesn't output a line break on error
if test "$status_code" -ne 201; then
echo -e "[\e[1m\e[91mERROR\e[39m\e[21m] curl returned $status_code, exiting..."
exit 1
fi


echo "---> Pushing operator YAML to ${FEDRAMP_GENERIC_REPOSITORY} with latest"
status_code=$(curl --silent --output /dev/stderr --write-out "%{http_code}" -u "${ARTIFACTORY_USERNAME_RELEASE}:${ARTIFACTORY_PASSWORD_RELEASE}" -X PUT "https://$FEDRAMP_GENERIC_REPOSITORY/latest/instana-agent-operator.yaml" -T "${MANIFEST_FILE_FEDRAMP}")
echo # curl doesn't output a line break on error
if test "$status_code" -ne 201; then
echo -e "[\e[1m\e[91mERROR\e[39m\e[21m] curl returned $status_code, exiting..."
exit 1
fi