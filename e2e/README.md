# Instana Agent Operator E2E Tests

This directory contains end-to-end tests for the Instana Agent Operator.

## Configuration

The e2e tests use environment variables for configuration. You can set these variables in a `.env` file in this directory or pass them directly to the test command.

### Environment Variables

#### Cluster Type

- `CLUSTER_TYPE`: The type of cluster to use for testing. Valid values are `kind` and `external`. Default: `external`.

#### Kind-specific Configuration (used only when CLUSTER_TYPE=kind)

- `KIND_CLUSTER_NAME`: The name of the kind cluster. Default: `instana-e2e`.
- `KIND_CONFIG`: The path to the kind cluster configuration file. Default: `e2e/kind-config-single-node.yaml`.
- `KIND_OPERATOR_IMAGE_NAME`: The name of the operator image to use for kind tests. Default: `instana-agent-operator`.
- `KIND_OPERATOR_IMAGE_TAG`: The tag of the operator image to use for kind tests. Default: `e2e`.

#### Common Configuration (used for both kind and external clusters)

- `INSTANA_API_KEY`: The Instana API key to use for testing. **Required**.
- `ARTIFACTORY_USERNAME`: The username for the Instana container registry. **Required**.
- `ARTIFACTORY_PASSWORD`: The password for the Instana container registry. **Required**.
- `ARTIFACTORY_HOST`: The host for the Instana container registry. Default: `delivery.instana.io`.
- `INSTANA_ENDPOINT_HOST`: The host for the Instana backend. Default: `ingress-red-saas.instana.io`.
- `OPERATOR_IMAGE_NAME`: The name of the operator image to use for external cluster tests. Default: `delivery.instana.io/int-docker-agent-local/instana-agent-operator/dev-build`.
- `OPERATOR_IMAGE_TAG`: The tag of the operator image to use for external cluster tests. If not set, it will use the git commit hash.

### .env File

You can create a `.env` file in this directory with the following content:

```
# Cluster type (external or kind)
CLUSTER_TYPE=external

# Kind-specific configuration (used only when CLUSTER_TYPE=kind)
KIND_CLUSTER_NAME=instana-e2e
KIND_CONFIG=e2e/kind-config-single-node.yaml
KIND_OPERATOR_IMAGE_NAME=instana-agent-operator
KIND_OPERATOR_IMAGE_TAG=e2e

# Common configuration (used for both kind and external clusters)
INSTANA_API_KEY=your-real-api-key
ARTIFACTORY_USERNAME=your-real-username
ARTIFACTORY_PASSWORD=your-real-password
ARTIFACTORY_HOST=delivery.instana.io
INSTANA_ENDPOINT_HOST=ingress-red-saas.instana.io
OPERATOR_IMAGE_NAME=your-real-operator-image
OPERATOR_IMAGE_TAG=your-real-tag
```

You can copy the `.env.template` file to `.env` and update the values:

```bash
cp .env.template .env
```

## Running Tests

### Setting Up the Environment

You can create a generic `.env` file from the template:

```bash
make e2e-env
```

This will create a `.env` file from the `.env.template` file. You should then edit the file to set the appropriate values for your environment.

### Kind Cluster Tests

To run tests on a kind cluster:

```bash
# Run all kind tests (standard and multinode)
make e2e-kind
```

This will:
1. Back up any existing `.env` file
2. Build the operator image
3. Create a kind cluster
4. Load the operator image into the cluster
5. Run the tests with `imagePullPolicy: IfNotPresent` to use the locally loaded image
6. Delete the kind cluster
7. Restore the original `.env` file

#### Image Pull Policy

For kind clusters, the operator is deployed with `imagePullPolicy: IfNotPresent` to ensure that the locally loaded image is used. This is done using the `deploy-kind` target in the Makefile, which modifies the deployment configuration before applying it.

### External Cluster Tests

To run tests on an external cluster:

```bash
# Make sure your kubeconfig is set up correctly
make e2e
```

This will:
1. Back up any existing `.env` file
2. Run the tests with CLUSTER_TYPE=external
3. Restore the original `.env` file

### Running Individual Tests

You can also run individual tests by setting the appropriate environment variables:

```bash
# For kind clusters
cd e2e && CLUSTER_TYPE=kind KIND_CLUSTER_NAME=instana-e2e KIND_CONFIG=e2e/kind-config-single-node.yaml KIND_OPERATOR_IMAGE_NAME=instana-agent-operator KIND_OPERATOR_IMAGE_TAG=e2e go test -v -tags=standard

# For external clusters
cd e2e && CLUSTER_TYPE=external go test -timeout=45m -count=1 -failfast -v github.com/instana/instana-agent-operator/e2e
```

## Test Tags

The e2e tests use tags to control which tests are run:

- `standard`: Run standard tests on a single-node cluster
- `multinode`: Run multinode tests on a multi-node cluster

## Adding New Tests

When adding new tests, follow these guidelines:

1. Use the `InstanaTestCfg` struct for configuration
2. Use the `CLUSTER_TYPE` environment variable to determine the cluster type
3. Use the appropriate environment variables for the cluster type
4. Add appropriate tags to the test file