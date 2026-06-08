#!/usr/bin/env bash
#
# (c) Copyright IBM Corp. 2025
#
if [[ "$PIPELINE_DEBUG" == 1 ]]; then
	trap env EXIT
	env
	set -x
fi
cd $WORKSPACE/$APP_REPO_FOLDER
# Load environment variables - use commit hash, not branch name
GIT_COMMIT=$(load_repo app-repo commit)
BRANCH_NAME=$(load_repo app-repo branch)

echo "Building for commit ${GIT_COMMIT} on branch ${BRANCH_NAME}"

dnf -y install microdnf
./installGolang.sh amd64
export PATH=$PATH:/usr/local/go/bin

IMAGE_TAG=${GIT_COMMIT}
echo "Using IMAGE_TAG=${IMAGE_TAG}"
unset HISTFILE

# Authenticate with the private registry
ICR_REGISTRY_DOMAIN="icr.io"
echo "[INFO] Authenticating with the $ICR_REGISTRY_DOMAIN private Docker registry..."
export DOCKER_API_VERSION="1.41"
docker login -u iamapikey --password-stdin "$ICR_REGISTRY_DOMAIN" < /config/api-key

OPERATOR_IMAGE_NAME=icr.io/instana-agent-dev/instana-agent-operator

# Poll for the operator image for up to 10 minutes (parallel job should push it)
echo "Waiting for operator image ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG} to be available..."
MAX_WAIT_SECONDS=600
POLL_INTERVAL=10
ELAPSED=0

while [ $ELAPSED -lt $MAX_WAIT_SECONDS ]; do
	if OPERATOR_IMG_DIGEST=$(skopeo inspect --format "{{.Digest}}" docker://${OPERATOR_IMAGE_NAME}:${IMAGE_TAG} 2>/dev/null); then
		echo "OPERATOR_IMG_DIGEST=$OPERATOR_IMG_DIGEST"
		echo "Operator image found after ${ELAPSED} seconds"
		break
	fi
	
	echo "Image not yet available, waiting ${POLL_INTERVAL} seconds... (${ELAPSED}/${MAX_WAIT_SECONDS}s elapsed)"
	sleep $POLL_INTERVAL
	ELAPSED=$((ELAPSED + POLL_INTERVAL))
done

if [ -z "$OPERATOR_IMG_DIGEST" ]; then
	echo "ERROR: Operator image ${OPERATOR_IMAGE_NAME}:${IMAGE_TAG} not found after ${MAX_WAIT_SECONDS} seconds"
	echo "The parallel image build job may have failed or is taking longer than expected"
	exit 1
fi

mkdir -p target
mkdir -p bundle
mkdir -p ../docker-input

export PREFIX="v"
export VERSION="0.0.0"
export OLM_RELEASE_VERSION=${VERSION#"$PREFIX"}

# Load GitHub API token from secret
GH_API_TOKEN=$(get_env github-token)
export GH_API_TOKEN

# Get currently published version of the OLM bundle in the community operators project, so we can correctly set the 'replaces' field
# Uses jq to filter out non-release versions
export PREV_VERSION=$(curl --silent --fail --show-error -L -H "Authorization: Bearer ${GH_API_TOKEN}" https://api.github.com/repos/instana/instana-agent-operator/tags |
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

# Create bundle for public operator with image: icr.io/instana-agent-dev/instana-agent-operator:<version>
make IMG="${OPERATOR_IMAGE}" \
	VERSION="${OLM_RELEASE_VERSION}" \
	PREV_VERSION="${PREV_VERSION}" \
	AGENT_IMG="icr.io/instana/agent@${AGENT_IMG_DIGEST}" \
	bundle

echo ""
echo "=== Validating Operator Version Label in Bundle ==="

# Validate: No placeholders remain
echo "Checking for placeholders in bundle..."
if grep -r "OPERATOR_VERSION_PLACEHOLDER" bundle/manifests/ 2>/dev/null; then
	echo "ERROR: OPERATOR_VERSION_PLACEHOLDER found in bundle manifests"
	exit 1
fi
echo "✓ No placeholders found"

# Validate: Version label present in bundle
echo "Checking for operator-version label in bundle..."
if ! grep -r "operator-version: ${OLM_RELEASE_VERSION}" bundle/manifests/ 2>/dev/null; then
	echo "ERROR: operator-version label not found in bundle manifests"
	exit 1
fi
echo "✓ Version label found: operator-version: ${OLM_RELEASE_VERSION}"

# Validate: Version label in CSV specifically
CSV_FILE="bundle/manifests/instana-agent-operator.clusterserviceversion.yaml"
if [ ! -f "$CSV_FILE" ]; then
	echo "ERROR: CSV file not found at $CSV_FILE"
	exit 1
fi
if ! grep -q "operator-version: ${OLM_RELEASE_VERSION}" "$CSV_FILE"; then
	echo "ERROR: operator-version label not found in CSV"
	exit 1
fi
echo "✓ Version label found in CSV"

# Validate: Bundle structure
if [ ! -d "bundle/manifests" ] || [ ! -d "bundle/metadata" ]; then
	echo "ERROR: Bundle structure is incomplete"
	exit 1
fi
echo "✓ Bundle structure validated"

echo "=== Bundle validation complete ==="
echo ""

cp bundle.Dockerfile ../docker-input/
cp -R bundle ../docker-input/
pushd bundle
zip -r ../target/olm-${OLM_RELEASE_VERSION}.zip .
popd

# Create the YAML for installing the Agent Operator, which we want to package with the release
# Redirect stderr to temporary file to avoid contaminating YAML output, but preserve for debugging
STDERR_LOG=$(mktemp)
make --silent IMG="${OPERATOR_IMAGE_NAME}:${OLM_RELEASE_VERSION}" VERSION="${OLM_RELEASE_VERSION}" controller-yaml 2>"$STDERR_LOG" >target/instana-agent-operator.yaml

# Display stderr output for debugging, then clean up
if [ -s "$STDERR_LOG" ]; then
	echo "=== Tool installation/version check messages ==="
	cat "$STDERR_LOG"
	echo "================================================"
fi
rm -f "$STDERR_LOG"

echo ""
echo "=== Validating Operator Version Label in Controller YAML ==="

# Validate: No placeholders remain
echo "Checking for placeholders in controller YAML..."
if grep -q "OPERATOR_VERSION_PLACEHOLDER" target/instana-agent-operator.yaml; then
	echo "ERROR: OPERATOR_VERSION_PLACEHOLDER found in controller YAML"
	exit 1
fi
echo "✓ No placeholders found"

# Validate: Version label present (should appear twice: deployment metadata + pod template)
echo "Checking for operator-version label in controller YAML..."
LABEL_COUNT=$(grep -c "operator-version: ${OLM_RELEASE_VERSION}" target/instana-agent-operator.yaml || true)
if [ "$LABEL_COUNT" -ne 2 ]; then
	echo "ERROR: Expected 2 occurrences of operator-version label, found $LABEL_COUNT"
	exit 1
fi
echo "✓ Version label found in deployment metadata and pod template (2 occurrences)"

# Validate: Version label NOT in selector
echo "Checking that operator-version is not in selector..."
if grep -A 3 "selector:" target/instana-agent-operator.yaml | grep -q "operator-version"; then
	echo "ERROR: operator-version found in selector (should not be there - immutable field)"
	exit 1
fi
echo "✓ Selector does not contain operator-version (correct)"

# Validate: YAML is well-formed
echo "Validating YAML structure..."
# Ensure PyYAML is installed for validation
if ! python3 -c "import yaml" 2>/dev/null; then
	echo "Installing PyYAML for YAML validation..."
	pip3 install --quiet PyYAML || {
		echo "Error: Could not install PyYAML, skipping YAML structure validation"
		echo "✓ YAML structure validation skipped (PyYAML not available)"
		echo ""
		echo "=== Controller YAML validation complete ==="
		echo ""
		echo -e "===== DISPLAYING target/instana-agent-operator.yaml =====\n"
		cat target/instana-agent-operator.yaml
		exit 1
	}
fi
YAML_ERROR=$(mktemp)
if ! python3 -c "import yaml; list(yaml.safe_load_all(open('target/instana-agent-operator.yaml')))" 2>"$YAML_ERROR"; then
	echo "ERROR: Controller YAML is not well-formed"
	echo ""
	echo "=== YAML Validation Error Details ==="
	cat "$YAML_ERROR"
	echo "======================================"
	echo ""
	echo "=== First 50 lines of generated YAML ==="
	head -n 50 target/instana-agent-operator.yaml
	echo "========================================="
	echo ""
	echo "=== Last 50 lines of generated YAML ==="
	tail -n 50 target/instana-agent-operator.yaml
	echo "========================================"
	rm -f "$YAML_ERROR"
	exit 1
fi
rm -f "$YAML_ERROR"
echo "✓ YAML structure validated"

echo "=== Controller YAML validation complete ==="
echo ""

echo -e "===== DISPLAYING parts of target/instana-agent-operator.yaml =====\n"
echo "=== First 20 lines of generated YAML ==="
head -n 20 target/instana-agent-operator.yaml
echo "========================================="
echo ""
echo "=== Last 20 lines of generated YAML ==="
tail -n 20 target/instana-agent-operator.yaml
echo "========================================"
