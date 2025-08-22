#!/usr/bin/env bash
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
	trap env EXIT
	env
	set -x
fi
export GIT_COMMIT="$(get_env branch || echo "latest")"
export ARTIFACTORY_INTERNAL_USERNAME=$(get_env ARTIFACTORY_INTERNAL_USERNAME)
export ARTIFACTORY_INTERNAL_PASSWORD=$(get_env ARTIFACTORY_INTERNAL_PASSWORD)

dnf -y install microdnf
./installGolang.sh amd64
export PATH=$PATH:/usr/local/go/bin

IMAGE_TAG=${GIT_COMMIT}
echo "Using IMAGE_TAG=${IMAGE_TAG}"
unset HISTFILE
skopeo login -u ${ARTIFACTORY_INTERNAL_USERNAME} -p ${ARTIFACTORY_INTERNAL_PASSWORD} delivery.instana.io

OPERATOR_IMAGE_NAME=delivery.instana.io/int-docker-agent-local/instana-agent-operator/dev-build
OPERATOR_IMG_DIGEST=$(skopeo inspect --format "{{.Digest}}" docker://${OPERATOR_IMAGE_NAME}:${IMAGE_TAG})
echo "OPERATOR_IMG_DIGEST=$OPERATOR_IMG_DIGEST"

mkdir -p target
mkdir -p bundle
mkdir -p ../docker-input

export PREFIX="v"
export VERSION="0.0.0"
export OLM_RELEASE_VERSION=${VERSION#"$PREFIX"}

# Get currently published version of the OLM bundle in the community operators project, so we can correctly set the 'replaces' field
# Uses jq to filter out non-release versions
export PREV_VERSION=$(curl --silent --fail --show-error -L https://api.github.com/repos/instana/instana-agent-operator/tags |
	jq 'map(select(.name | test("^v[0-9]+.[0-9]+.[0-9]+$"))) | .[1].name' |
	sed 's/[^0-9]*\([0-9]\+\.[0-9]\+\.[0-9]\+\).*/\1/')

if [[ "x${PREV_VERSION}" = "x" ]]; then
	echo "!! Could not determine previous released version. Fix either pipeline or tag history !!"
	exit 1
fi

echo "Operator manifest SHA found, using digest ${OPERATOR_IMG_DIGEST} for Operator image"
export OPERATOR_IMAGE="${OPERATOR_IMAGE_NAME}@${OPERATOR_IMG_DIGEST}"

# Get the digest for the agent image
AGENT_IMAGE_NAME="icr.io/instana/agent"
AGENT_IMAGE_TAG="latest"
echo "Fetching digest for ${AGENT_IMAGE_NAME}:${AGENT_IMAGE_TAG}"
export AGENT_IMG_DIGEST=$(skopeo inspect --format "{{.Digest}}" docker://${AGENT_IMAGE_NAME}:${AGENT_IMAGE_TAG})
echo "AGENT_IMG_DIGEST=$AGENT_IMG_DIGEST"

# Create bundle for public operator with image: delivery.instana.io/int-docker-agent-local/instana-agent-operator/dev-build:<version>
make IMG="${OPERATOR_IMAGE}" \
	VERSION="${OLM_RELEASE_VERSION}" \
	PREV_VERSION="${PREV_VERSION}" \
	AGENT_IMG="icr.io/instana/agent@${AGENT_IMG_DIGEST}" \
	bundle

cp bundle.Dockerfile ../docker-input/
cp -R bundle ../docker-input/
pushd bundle
zip -r ../target/olm-${OLM_RELEASE_VERSION}.zip .
popd

# Create the YAML for installing the Agent Operator, which we want to package with the release
make --silent IMG="${OPERATOR_IMAGE_NAME}:${OLM_RELEASE_VERSION}" controller-yaml >target/instana-agent-operator.yaml

echo -e "===== DISPLAYING target/instana-agent-operator.yaml =====\n"
cat target/instana-agent-operator.yaml
