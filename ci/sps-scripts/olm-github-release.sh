#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
set -euo pipefail
# Get the git commit hash to use as the image tag
GIT_COMMIT=$(load_repo app-repo commit)
OPERATOR_IMAGE_MANIFEST_SHA=${GIT_COMMIT}

# Create a place to store our output for packaging up
mkdir -p target
TARGET_DIR=$(pwd)/target

# strip the leading "v" from the operator version for github artefacts and release:
PREFIX="v"
VERSION=$(cat "version/INSTANA_AGENT_OPERATOR_VERSION")
OLM_RELEASE_VERSION=${VERSION#"$PREFIX"}

# Get currently published version of the OLM bundle in the community operators project, so we can correctly set the 'replaces' field
# Uses jq to filter out non-release versions
PREV_VERSION=$(curl --silent --fail --show-error -L https://api.github.com/repos/instana/instana-agent-operator/tags \
| jq 'map(select(.name | test("^v[0-9]+.[0-9]+.[0-9]+$"))) | .[1].name' \
| sed 's/[^0-9]*\([0-9]\+\.[0-9]\+\.[0-9]\+\).*/\1/')

if [[ "x${PREV_VERSION}" = "x" ]]; then
echo "!! Could not determine previous released version. Fix either pipeline or tag history !!"
exit 1
fi

# For releases, we always use the version tag, not the commit SHA
echo "Using version ${OLM_RELEASE_VERSION} for Operator image"
OPERATOR_IMAGE="icr.io/instana/instana-agent-operator:${OLM_RELEASE_VERSION}"

# check that the operator image is really present before creating a release
echo "Verifying operator image exists: ${OPERATOR_IMAGE}"
skopeo inspect docker://${OPERATOR_IMAGE}
# Create bundle for public operator with image:  icr.io/instana/instana-agent-operator:<version>
make IMG="${OPERATOR_IMAGE}" \
VERSION="${OLM_RELEASE_VERSION}" \
PREV_VERSION="${PREV_VERSION}" \
AGENT_IMG="icr.io/instana/agent@${AGENT_IMG_DIGEST}" \
bundle

pushd bundle
zip -r ../target/olm-${OLM_RELEASE_VERSION}.zip .
popd

# Create the YAML for installing the Agent Operator, which we want to package with the release
make --silent IMG="icr.io/instana/instana-agent-operator:${OLM_RELEASE_VERSION}" controller-yaml > target/instana-agent-operator.yaml

echo "delivery.instana.io/rel-docker-agent-local/instana-agent-operator:${OLM_RELEASE_VERSION}" > target/images.txt
echo "icr.io/instana/instana-agent-operator:${OLM_RELEASE_VERSION}" >> target/images.txt

# Only include the latest tag when running on main branch
if [[ "${BRANCH}" == "main" ]]; then
echo "icr.io/instana/instana-agent-operator:latest" >> target/images.txt
fi

cat target/images.txt

# For public releases, also create the appropriate github release:RELEASE_REGEX='^v[0-9]+\.[0-9]+\.[0-9]+$'
if ! [[ $VERSION =~ $RELEASE_REGEX ]]; then
echo "---> **** Internal release, GitHub release creation skipped. ****"
exit 0
fi

echo "**** Public release, create github.com release $VERSION. ****"
./ci/scripts/create-github-release.sh $OLM_RELEASE_VERSION $GH_API_TOKEN $TARGET_DIR