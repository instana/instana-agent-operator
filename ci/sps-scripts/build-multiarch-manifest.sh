#!/usr/bin/env bash
set -euo pipefail
ARTIFACTORY_INTERNAL_CREDENTIALS=$(get_env artifactory_internal)
ARTIFACTORY_CREDENTIALS=$(get_env artifactory)
RED_HAT_REGISTRY_CREDENTIALS=$(get_env RED_HAT_REGISTRY)

ARTIFACTORY_USERNAME_INTERNAL=$(echo "${ARTIFACTORY_INTERNAL_CREDENTIALS}" | jq -r ".username")
ARTIFACTORY_PASSWORD_INTERNAL=$(echo "${ARTIFACTORY_INTERNAL_CREDENTIALS}" | jq -r ".password")

RED_HAT_REGISTRY_USERNAME=$(echo "${RED_HAT_REGISTRY_CREDENTIALS}" | jq -r ".username")
RED_HAT_REGISTRY_PASSWORD=$(echo "${RED_HAT_REGISTRY_CREDENTIALS}" | jq -r ".password")

ARTIFACTORY_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")
ARTIFACTORY_PASSWORD=$(echo "${ARTIFACTORY_CREDENTIALS}" | jq -r ".password")

DEV_BUILD_IMAGE=delivery.instana.io/int-docker-agent-local/instana-agent-operator/dev-build
ICR_REPOSITORY=icr.io/instana/instana-agent-operator
ARTIFACTORY_REPOSITORY="${ARTIFACTORY_CONTAINER_DOCKER_URL}/instana-agent-operator"
RED_HAT_REGISTRY="quay.io/redhat-isv-containers/5e961c2c93604e02afa9ebdf"

DIGEST=$(cat agent-operator-image-manifest-sha/digest)
echo ${DIGEST}
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_INTERNAL} \
--src-password ${ARTIFACTORY_PASSWORD_INTERNAL} \
--dest-username ${ARTIFACTORY_USERNAME} \
--dest-password ${ARTIFACTORY_PASSWORD} \
docker://${DEV_BUILD_IMAGE}@${DIGEST} \
docker://${DEV_BUILD_IMAGE}:main

# For non-public releases we are done:
export RELEASE_REGEX='^v[0-9]+\.[0-9]+\.[0-9]+$'
if ! [[ $VERSION =~ $RELEASE_REGEX ]]; then
echo "---> **** Internal release, publishing to icr.io & Red Hat container registry skipped. ****"
exit 0
fi

echo "---> **** Public release, publishing to icr.io & Red Hat container registry. ****"

# strip the leading "v" from the operator version for release:
export PREFIX="v"
export OPERATOR_DOCKER_VERSION=${VERSION#"$PREFIX"}

echo "---> Pushing multi-architectural manifest to icr.io with version tag"
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_INTERNAL} \
--src-password ${ARTIFACTORY_PASSWORD_INTERNAL} \
--dest-username ${ICR_USERNAME} \
--dest-password ${ICR_PASSWORD} \
docker://${DEV_BUILD_IMAGE}@${DIGEST} \
docker://${ICR_REPOSITORY}:${OPERATOR_DOCKER_VERSION}

echo "---> Pushing multi-architectural manifest to icr.io with the latest tag"
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_INTERNAL} \
--src-password ${ARTIFACTORY_PASSWORD_INTERNAL} \
--dest-username ${ICR_USERNAME} \
--dest-password ${ICR_PASSWORD} \
docker://${DEV_BUILD_IMAGE}@${DIGEST} \
docker://${ICR_REPOSITORY}:latest

echo "---> Pushing multi-architectural manifest to ${ARTIFACTORY_REPOSITORY}"
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_INTERNAL} \
--src-password ${ARTIFACTORY_PASSWORD_INTERNAL} \
--dest-username ${ARTIFACTORY_USERNAME} \
--dest-password ${ARTIFACTORY_PASSWORD} \
docker://${DEV_BUILD_IMAGE}@${DIGEST} \
docker://${ARTIFACTORY_REPOSITORY}:${OPERATOR_DOCKER_VERSION}

echo "---> pushing images to Red Hat Container Registry"
skopeo copy -a --preserve-digests \
--src-username ${ARTIFACTORY_USERNAME_INTERNAL} \
--src-password ${ARTIFACTORY_PASSWORD_INTERNAL} \
--dest-username ${RED_HAT_REGISTRY_USERNAME} \
--dest-password ${RED_HAT_REGISTRY_PASSWORD} \
docker://${DEV_BUILD_IMAGE}@${DIGEST} \
docker://${RED_HAT_REGISTRY}:${OPERATOR_DOCKER_VERSION}
