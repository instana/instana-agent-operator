#
# (c) Copyright IBM Corp. 2025
#

# Detect operating system and architecture
OS := $(shell uname -s | tr '[:upper:]' '[:lower:]')
ARCH := $(shell uname -m)
ifeq ($(ARCH),x86_64)
	ARCH := amd64
endif
ifeq ($(ARCH),aarch64)
	ARCH := arm64
endif

# Tools installation directory and shortcuts
export GOBIN=$(shell pwd)/bin
CONTROLLER_GEN_BIN = ${GOBIN}/controller-gen
KUSTOMIZE_BIN = ${GOBIN}/kustomize
ENVTEST_BIN = ${GOBIN}/setup-envtest
GOLANGCI_LINT_BIN = ${GOBIN}/golangci-lint
OPERATOR_SDK_BIN = ${GOBIN}/operator-sdk
BUILDCTL_BIN =  ${GOBIN}/buildctl
MOCKGEN_BIN = ${GOBIN}/mockgen

# Detect if podman or docker is available locally
ifeq ($(shell command -v podman 2> /dev/null),)
    CONTAINER_CMD = docker
else
    CONTAINER_CMD = podman
endif

# Current Operator version (override when executing Make target, e.g. like `make bundle VERSION=2.0.0`)
VERSION ?= 0.0.1

# Previous version, will only be used for updating the "replaces" field in the ClusterServiceVersion when defined command-line
PREV_VERSION ?= 0.0.0

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
CHANNELS ?= "stable"
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
DEFAULT_CHANNEL ?= "stable"
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?=  icr.io/instana/instana-agent-operator:latest

# Image URL for the Instana Agent, as listed in the 'relatedImages' field in the CSV
AGENT_IMG ?= icr.io/instana/agent:latest

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

NAMESPACE ?= instana-agent
NAMESPACE_PREPULLER ?= instana-agent-image-prepuller

INSTANA_AGENT_CLUSTER_WIDE_RESOURCES := \
	"crd/agents.instana.io" \
	"clusterrole/leader-election-role" \
	"clusterrole/instana-agent-clusterrole" \
	"clusterrolebinding/leader-election-rolebinding" \
	"clusterrolebinding/instana-agent-clusterrolebinding"

##@ General targets

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

install: install-githooks install-tools generate-all ## Prepare the whole project with all dependencies and requirements to be run
	go install

install-githooks: ## Sets/Enables .githooks as the hooks path
	git config core.hooksPath .githooks

install-tools: ## Installs tools all tools needed by the project in the ./bin-folder
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0
	go install sigs.k8s.io/kustomize/kustomize/v4@v4.5.5
	go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest
	go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@v2.1.6
	go install github.com/moby/buildkit/cmd/buildctl@v0.16
	go install go.uber.org/mock/mockgen@74a29c6e6c2cbb8ccee94db061c1604ff33fd188

uninstall-tools: ## Uninstalls tools in the ./bin-folder.
	rm -rf ./bin/*

##@ Testing and QA targets

test-all: lint test e2e ## Runs all validation and tests

.PHONY: e2e
e2e: ## Run end-to-end tests
	go test -timeout=30m -count=1 -failfast -v github.com/instana/instana-agent-operator/e2e

EXCLUDED_TEST_DIRS = mocks e2e
EXCLUDE_PATTERN = $(shell echo $(EXCLUDED_TEST_DIRS) | sed 's/ /|/g')
PACKAGES = $(shell go list ./... | grep -vE "$(EXCLUDE_PATTERN)" | tr '\n' ' ')
ENVTEST_K8S_VERSION = 1.32
KUBEBUILDER_ASSETS=$(shell $(ENVTEST_BIN) use $(ENVTEST_K8S_VERSION) -p path)
test: ## Run tests but ignore specific directories that match EXCLUDED_TEST_DIRS
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test $(PACKAGES) -coverprofile=coverage.out

lint: ## Run linter
	$(GOLANGCI_LINT_BIN) run --new-from-rev=HEAD --timeout 5m

##@ Build/Run targets

build: ## Build manager binary.
	go build -o bin/manager *.go

run: export DEBUG_MODE=true
run: ## Run against the configured Kubernetes cluster in ~/.kube/config (run the "install" target to install CRDs into the cluster)
	go run ./

##@ Generation targets

generate-all: generate-manifests generate-deepcopies generate-mocks ## Generate all

generate-manifests: ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN_BIN) crd rbac:roleName=instana-agent-clusterrole webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate-deepcopies: ## Generate code that uses DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN_BIN) object paths="./..."

CONTROLLER_RUNTIME_VERSION := $(shell go list -m all | grep sigs.k8s.io/controller-runtime | awk '{print $$2}')
generate-mocks: ## Generate mocks for tests
	${MOCKGEN_BIN} --source $(shell go env GOPATH)/pkg/mod/sigs.k8s.io/controller-runtime@$(CONTROLLER_RUNTIME_VERSION)/pkg/client/interfaces.go --destination ./mocks/k8s_client_mock.go --package mocks
	${MOCKGEN_BIN} --source ./pkg/hash/hash.go --destination ./mocks/hash_mock.go --package mocks
	${MOCKGEN_BIN} --source ./pkg/k8s/client/client.go --destination ./mocks/instana_agent_client_mock.go --package mocks
	${MOCKGEN_BIN} --source ./pkg/k8s/object/transformations/pod_selector.go --destination ./mocks/pod_selector_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/object/transformations/transformations.go --destination ./mocks/transformations_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/object/builders/common/ports/ports_builder.go --destination ./mocks/ports_builder_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/object/builders/common/env/env_builder.go --destination ./mocks/env_builder_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/object/builders/common/volume/volume_builder.go --destination ./mocks/volume_builder_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./mocks/helpers_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/object/builders/common/builder/builder.go --destination ./mocks/builder_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/json_or_die/json.go --destination ./mocks/json_or_die_marshaler_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/operator/status/agent_status_manager.go --destination ./mocks/agent_status_manager_mock.go --package mocks 
	${MOCKGEN_BIN} --source ./pkg/k8s/operator/lifecycle/dependent_lifecycle_manager.go --destination ./mocks/dependent_lifecycle_manager_mock.go --package mocks

##@ Containerization targets

container-push: ## Push the docker image with the manager.
	${CONTAINER_CMD} push ${IMG}

BUILDPLATFORM ?= linux/${ARCH}
BUILDTARGET ?= linux/${ARCH}
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)
container-build: ## Build docker image with the manager.
	$(BUILDCTL_BIN) --addr=${CONTAINER_CMD}-container://buildkitd build \
		--frontend=dockerfile.v0 \
		--local context=. \
		--local dockerfile=. \
		--opt build-arg:VERSION=0.0.1 \
		--opt build-arg:GIT_COMMIT=${GIT_COMMIT}  \
		--opt build-arg:DATE="$$(date)" \
		--opt build-arg:BUILDPLATFORM=${BUILDPLATFORM} \
		--opt build-arg:TARGETPLATFORM=${TARGETPLATFORM} \
		--opt filename=Containerfile \
		--output type=oci,name=${IMG} | $(CONTAINER_CMD) load

##@ Kubernetes targets

INSTANA_AGENT_YAML?=config/samples/instana_v1_instanaagent_demo.yaml
kubectl-apply: ## Apply all .yaml-files needed to run the operator in a kubernetes environment (INSTANA_AGENT_YAML environment variable defines path to Instana Agent .yaml-file)
	kubectl apply -k config/crd
	kubectl apply -f config/samples/instana_agent_namespace.yaml
	@if [ -f ${INSTANA_AGENT_YAML} ]; then \
		kubectl apply -f ${INSTANA_AGENT_YAML}; \
	else \
		echo "Warning: Instana Agent .yaml-file wasn't there! No Agent will be added..."; \
	fi
	cd config/manager && $(KUSTOMIZE_BIN) edit set image instana/instana-agent-operator=${IMG}
	$(KUSTOMIZE_BIN) build config/default | kubectl apply -f -

kubectl-delete: ## Remove any traces of the operator that are found in the kubernetes environment
	@echo "=== Removing finalizers from agent CR, if present ==="
	@echo "Checking if agent CR is present in namespace $(NAMESPACE)..."
	@if kubectl get agents.instana.io instana-agent -n $(NAMESPACE) >/dev/null 2>&1; then \
		echo "Found, removing finalizers..."; \
		kubectl patch agents.instana.io instana-agent -p '{"metadata":{"finalizers":null}}' --type=merge -n $(NAMESPACE); \
	else \
		echo "CR not present"; \
	fi
	@echo "=== Cleaning up cluster wide resources, if present ==="
	@for resource in $(INSTANA_AGENT_CLUSTER_WIDE_RESOURCES); do \
		resource_type=$$(echo $$resource | cut -d'/' -f1); \
		resource_name=$$(echo $$resource | cut -d'/' -f2); \
		if kubectl get $$resource_type $$resource_name > /dev/null 2>&1; then \
			echo "Deleting $$resource..."; \
			kubectl delete $$resource_type $$resource_name; \
		else \
			echo "Resource $$resource does not exist, skipping..."; \
		fi; \
	done
	@echo "Cleanup complete!"
	@echo "=== Removing instana-agent namespace, if present ==="
	kubectl delete ns $(NAMESPACE) --wait || true

kubectl-scale-to-zero: ## Scales the operator to zero in the cluster to allow local testing against a cluster
	kubectl -n instana-agent scale --replicas=0 deployment.apps/instana-agent-operator && sleep 5 && kubectl get all -n instana-agent

deploy-minikube: ## Convenience target to push the docker image to a local running Minikube cluster and deploy the Operator there.
	(eval $$(minikube docker-env) && docker rmi ${IMG} || true)
	docker save ${IMG} | (eval $$(minikube docker-env) && docker load)
	# Update correct Controller Manager image to be used
	cd config/manager && $(KUSTOMIZE_BIN) edit set image instana/instana-agent-operator=${IMG}
	# Make certain we don't try to pull images from somewhere else
	$(KUSTOMIZE_BIN) build config/default | sed -e 's|\(imagePullPolicy:\s*\)Always|\1Never|' | kubectl apply -f -

.PHONY: create-pull-secret
create-pull-secret: ## Creates image pull secret for delivery.instana.io from your local docker config
	@echo "Filtering Docker config for delivery.instana.io settings, ensure to login locally first..."
	@mkdir -p .tmp
	@jq '{auths: {"delivery.instana.io": .auths["delivery.instana.io"]}}' ${HOME}/.docker/config.json > .tmp/filtered-docker-config.json
	@echo "Checking if secret delivery-instana-io-pull-secret exists in namespace $(NAMESPACE)..."
	@if kubectl get secret delivery-instana-io-pull-secret -n $(NAMESPACE) >/dev/null 2>&1; then \
		echo "Updating existing secret delivery-instana-io-pull-secret..."; \
		kubectl delete secret delivery-instana-io-pull-secret -n $(NAMESPACE); \
		kubectl create secret generic delivery-instana-io-pull-secret \
			--from-file=.dockerconfigjson=.tmp/filtered-docker-config.json \
			--type=kubernetes.io/dockerconfigjson \
			-n $(NAMESPACE); \
	else \
		echo "Creating new secret delivery-instana-io-pull-secret..."; \
		kubectl create secret generic delivery-instana-io-pull-secret \
			--from-file=.dockerconfigjson=.tmp/filtered-docker-config.json \
			--type=kubernetes.io/dockerconfigjson \
			-n $(NAMESPACE); \
	fi
	@echo "Patching serviceaccount..."
	@kubectl patch serviceaccount instana-agent-operator \
		-p '{"imagePullSecrets": [{"name": "delivery-instana-io-pull-secret"}]}' \
		-n instana-agent
	@rm -rf .tmp
	@echo "Restarting operator deployment..."
	@kubectl delete pods -l app.kubernetes.io/name=instana-agent-operator -n $(NAMESPACE)

.PHONY: pre-pull-images
pre-pull-images: ## Pre-pulls images on the target cluster (useful in slow network situations to run tests reliably)
	@if [ "$(INSTANA_API_KEY)" == "" ]; then \
		echo "env variable INSTANA_API_KEY is undefined but should contain the agent download key"; \
		exit 1; \
	fi
	kubectl apply -f ci/scripts/instana-agent-image-prepuller-ns.yaml || true
	@echo "Creating Docker registry secret..."
	@echo "Checking if secret containers-instana-io-pull-secret exists in namespace $(NAMESPACE_PREPULLER)..."
	@if kubectl get secret containers-instana-io-pull-secret -n $(NAMESPACE_PREPULLER) >/dev/null 2>&1; then \
		echo "Updating existing secret containers-instana-io-pull-secret..."; \
		kubectl delete secret containers-instana-io-pull-secret -n $(NAMESPACE_PREPULLER); \
	fi
	@kubectl create secret docker-registry containers-instana-io-pull-secret \
		--docker-server=containers.instana.io \
		--docker-username="_" \
		--docker-password=$${INSTANA_API_KEY} \
		-n $(NAMESPACE_PREPULLER)
	@echo "Start instana-agent-image-prepuller daemonset..."
	@echo "Checking if daemonset instana-agent-image-prepuller exists in namespace $(NAMESPACE_PREPULLER)..."
	@if kubectl get ds instana-agent-image-prepuller -n $(NAMESPACE_PREPULLER) >/dev/null 2>&1; then \
		echo "Updating existing secret containers-instana-io-pull-secret..."; \
		kubectl delete ds instana-agent-image-prepuller -n $(NAMESPACE_PREPULLER); \
		kubectl delete pods -n $(NAMESPACE_PREPULLER) -l name=instana-agent-image-prepuller --force --grace-period=0; \
	fi
	@kubectl apply -f ci/scripts/instana-agent-image-prepuller.yaml -n $(NAMESPACE_PREPULLER)
	@echo "Waiting for the instana-agent-prepuller daemonset"
	@kubectl rollout status ds/instana-agent-image-prepuller -n $(NAMESPACE_PREPULLER) --timeout=1800s
	@echo "Cleaning up instana-agent-prepuller namespace"
	kubectl delete ds instana-agent-image-prepuller -n $(NAMESPACE_PREPULLER)
	kubectl delete pods -n $(NAMESPACE_PREPULLER) -l name=instana-agent-image-prepuller --force --grace-period=0 || true
	kubectl delete ns $(NAMESPACE_PREPULLER)

BUILDKITD_CONTAINER_NAME = buildkitd
# Test if buildctl is available in the GOPATH, if not, set to local and download if needed
buildctl: ## Download the buildctl cli locally if necessary.
	@if [ "`podman ps -a -q -f name=$(BUILDKITD_CONTAINER_NAME)`" ]; then \
		if [ "`podman ps -aq -f status=exited -f name=$(BUILDKITD_CONTAINER_NAME)`" ]; then \
			echo "Starting buildkitd container $(BUILDKITD_CONTAINER_NAME)"; \
			$(CONTAINER_CMD) start $(BUILDKITD_CONTAINER_NAME) || true; \
			echo "Allowing 5 seconds to bootup"; \
			sleep 5; \
		else \
			echo "Buildkit daemon is already running, skip container creation"; \
		fi \
	else \
		echo "$(BUILDKITD_CONTAINER_NAME) container is not present, launching it now"; \
		$(CONTAINER_CMD) run -d --name buildkitd --privileged docker.io/moby/buildkit:v0.16.0; \
		echo "Allowing 5 seconds to bootup"; \
		sleep 5; \
	fi

##@ OpenShift Container Platform specific targets

.PHONY: namespace
oc-namespace: ## Generate namespace instana-agent on OCP for manual testing
	oc new-project instana-agent || true
	oc adm policy add-scc-to-user privileged -z instana-agent -n instana-agent

.PHONY: setup-ocp-mirror
setup-ocp-mirror: ## Setup ocp internal registry and define ImageContentSourcePolicy to pull from internal registry
	./ci/scripts/setup-ocp-mirror.sh

.PHONY: dev-run-ocp
dev-run-ocp: oc-namespace install kubectl-apply run ## Creates a full dev deployment on OCP from scratch, also useful after purge

.PHONY: logs
logs: ## Tail operator logs
	kubectl logs -f deployment/instana-agent-controller-manager -n $(NAMESPACE)

##@ OLM

.PHONY: bundle
bundle: ## Generate bundle manifests and metadata, then validate generated files for an OLM bundle.
	@if [ -f $(OPERATOR_SDK_BIN) ]; then \
		echo "operator-sdk already installed at: $(OPERATOR_SDK_BIN)"; \
	else \
		curl -Lo $(OPERATOR_SDK_BIN) https://github.com/operator-framework/operator-sdk/releases/download/v1.23.0/operator-sdk_${OS}_${ARCH}; \
		chmod +x $(OPERATOR_SDK_BIN); \
	fi
	$(OPERATOR_SDK_BIN) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE_BIN) edit set image "instana/instana-agent-operator=$(IMG)"
	$(KUSTOMIZE_BIN) build config/manifests \
		| sed -e 's|\(replaces:.*v\)0.0.0|\1$(PREV_VERSION)|' \
		| sed -e 's|\(containerImage:[[:space:]]*\).*|\1$(IMG)|' \
		| sed -e 's|\(image:[[:space:]]*\).*instana-agent-operator:0.0.0|\1$(IMG)|' \
		| sed -e 's|\(image:[[:space:]]*\).*agent:latest|\1$(AGENT_IMG)|' \
		| $(OPERATOR_SDK_BIN) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	./hack/patch-bundle.sh
	$(OPERATOR_SDK_BIN) bundle validate ./bundle

.PHONY: bundle-build
BUNDLE_IMG?=instana-agent-operator-bundle:$(VERSION)
bundle-build: ## Build the bundle image for OLM.
	$(BUILDCTL_BIN) --addr=${CONTAINER_CMD}-container://buildkitd build \
		--frontend gateway.v0 \
		--local context=. \
		--local dockerfile=. \
		--opt source=docker/dockerfile \
		--opt filename=./bundle.Dockerfile \
		--output type=oci,name=${BUNDLE_IMG} | $(CONTAINER_CMD) load

controller-yaml: ## Output the YAML for deployment, so it can be packaged with the release. Use `make --silent` to suppress other output.
	cd config/manager && $(KUSTOMIZE_BIN) edit set image "instana/instana-agent-operator=$(IMG)"
	$(KUSTOMIZE_BIN) build config/default
	