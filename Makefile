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
CONTROLLER_GEN = ${GOBIN}/controller-gen
KUSTOMIZE = ${GOBIN}/kustomize
ENVTEST = ${GOBIN}/setup-envtest
GOLANGCI_LINT = ${GOBIN}/golangci-lint
OPERATOR_SDK = ${GOBIN}/operator-sdk
BUILDCTL =  ${GOBIN}/buildctl

# Current Operator version (override when executing Make target, e.g. like `make VERSION=2.0.0 bundle`)
VERSION ?= 0.0.1

# Previous version, will only be used for updating the "replaces" field in the ClusterServiceVersion when defined command-line
PREV_VERSION ?= 0.0.0

# Tool versions
CONTROLLER_GEN_VERSION ?= v0.19.0 # renovate: datasource=github-releases depName=kubernetes-sigs/controller-tools
KUSTOMIZE_VERSION ?= v4.5.5 # renovate: datasource=github-releases depName=kubernetes-sigs/kustomize
GOLANGCI_LINT_VERSION ?= v2.4.0 # renovate: datasource=github-releases depName=golangci/golangci-lint
# Buildkit versions - the image tag is the actual release version, CLI version is derived from it
BUILDKIT_IMAGE_TAG ?= v0.16.0 # renovate: datasource=github-releases depName=moby/buildkit
# Extract major.minor version for buildctl CLI (strip patch version)
BUILDCTL_VERSION = $(shell echo $(BUILDKIT_IMAGE_TAG) | sed -E 's/v([0-9]+\.[0-9]+)\.[0-9]+.*/v\1/')

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
IMG ?= icr.io/instana/instana-agent-operator:latest

# Image URL for the Instana Agent, as listed in the 'relatedImages' field in the CSV
AGENT_IMG ?= icr.io/instana/agent:latest

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.32

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

# Some commands work on Linux but not on MacOS and vice versa. Create variables for them so to run the proper command.
uname := $(shell uname)
ifeq ($(uname), Linux)
get_ip_addr := ip route get 1 | awk '{print $$(NF-2);exit}'
endif
ifeq ($(uname), Darwin)
get_ip_addr := ipconfig getifaddr en0
endif

# Detect if podman or docker is available locally
ifeq ($(shell command -v podman 2> /dev/null),)
    CONTAINER_CMD = docker
else
    CONTAINER_CMD = podman
endif

NAMESPACE ?= instana-agent
NAMESPACE_PREPULLER ?= instana-agent-image-prepuller

INSTANA_AGENT_CLUSTER_WIDE_RESOURCES := \
	"crd/agents.instana.io" \
	"crd/agentsremote.instana.io" \
	"clusterrole/leader-election-role" \
	"clusterrole/instana-agent-clusterrole" \
	"clusterrolebinding/leader-election-rolebinding" \
	"clusterrolebinding/instana-agent-clusterrolebinding"

all: build


##@ General

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

setup: ## sets git hooks path to .githook
	git config core.hooksPath .githooks

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=instana-agent-clusterrole webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...

lint: golangci-lint ## Run the golang-ci linter
	$(GOLANGCI_LINT) run --new-from-rev=HEAD --timeout 5m

EXCLUDED_TEST_DIRS = mocks e2e
EXCLUDE_PATTERN = $(shell echo $(EXCLUDED_TEST_DIRS) | sed 's/ /|/g')
PACKAGES = $(shell go list ./... | grep -vE "$(EXCLUDE_PATTERN)" | tr '\n' ' ')
KUBEBUILDER_ASSETS=$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)
test: manifests generate fmt vet lint envtest ## Run tests but ignore specific directories that match EXCLUDED_TEST_DIRS
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test $(PACKAGES) -coverprofile=coverage.out

.PHONY: e2e
e2e: ## Run end-to-end tests
	go test -timeout=45m -count=1 -failfast -v github.com/instana/instana-agent-operator/e2e

##@ Build

build: setup generate fmt vet ## Build manager binary.
	go build -o bin/manager *.go

run: export DEBUG_MODE=true
run: generate fmt vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config (run the "install" target to install CRDs into the cluster)
	go run ./

docker-build: test container-build ## Build docker image with the manager.

docker-push: ## Push the docker image with the manager.
	${CONTAINER_CMD} push ${IMG}

BUILDPLATFORM ?= linux/${ARCH}
BUILDTARGET ?= linux/${ARCH}
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

container-build: buildctl
	$(BUILDCTL) --addr=${CONTAINER_CMD}-container://buildkitd build \
	  --frontend=dockerfile.v0 \
	  --local context=. \
	  --local dockerfile=. \
	  --output type=oci,name=${IMG} \
	  --opt build-arg:VERSION=0.0.1 \
	  --opt build-arg:GIT_COMMIT=${GIT_COMMIT} \
	  --opt build-arg:BUILDPLATFORM=${BUILDPLATFORM} \
	  --opt build-arg:TARGETPLATFORM=${TARGETPLATFORM} \
	  --opt build-arg:DATE="$$(date)" | $(CONTAINER_CMD) load

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -k config/crd

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/crd

purge: ## Full purge of the agent in the cluster
	@echo "=== Removing finalizers from agent CR, if present ==="
	@echo "Checking if agent CR is present in namespace $(NAMESPACE)..."
	@if kubectl get agents.instana.io instana-agent -n $(NAMESPACE) >/dev/null 2>&1; then \
		echo "Found, removing finalizers..."; \
		kubectl patch agents.instana.io instana-agent -p '{"metadata":{"finalizers":null}}' --type=merge -n $(NAMESPACE); \
	else \
		echo "CR not present"; \
	fi
	@if kubectl get agentsremote.instana.io instana-agent-remote -n $(NAMESPACE) >/dev/null 2>&1; then \
		echo "Found, removing remote finalizers..."; \
		kubectl patch agentsremote.instana.io instana-agent-remote -p '{"metadata":{"finalizers":null}}' --type=merge -n $(NAMESPACE); \
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

deploy: manifests kustomize ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image instana/instana-agent-operator=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

scale-to-zero: ## Scales the operator to zero in the cluster to allow local testing against a cluster
	kubectl -n instana-agent scale --replicas=0 deployment.apps/instana-agent-operator && sleep 5 && kubectl get all -n instana-agent

deploy-minikube: manifests kustomize ## Convenience target to push the docker image to a local running Minikube cluster and deploy the Operator there.
	(eval $$(minikube docker-env) && docker rmi ${IMG} || true)
	docker save ${IMG} | (eval $$(minikube docker-env) && docker load)
	# Update correct Controller Manager image to be used
	cd config/manager && $(KUSTOMIZE) edit set image instana/instana-agent-operator=${IMG}
	# Make certain we don't try to pull images from somewhere else
	$(KUSTOMIZE) build config/default | sed -e 's|\(imagePullPolicy:\s*\)Always|\1Never|' | kubectl apply -f -

undeploy: ## Undeploy controller from the configured Kubernetes cluster in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

.PHONY: namespace
namespace: ## Generate namespace instana-agent for manual testing
	@echo "Creating namespace $(NAMESPACE) if it doesn't exist..."
	kubectl create namespace $(NAMESPACE) 2>/dev/null || true
	@echo "Detecting cluster type (OCP or standard K8s)..."
	@if command -v oc >/dev/null 2>&1 && oc get projects >/dev/null 2>&1; then \
		echo "OpenShift cluster detected, applying OCP-specific security policies..."; \
		oc adm policy add-scc-to-user privileged -z instana-agent -n $(NAMESPACE) || true; \
		oc adm policy add-scc-to-user anyuid -z instana-agent-remote -n $(NAMESPACE) || true; \
	else \
		echo "Standard Kubernetes cluster detected (GKE or other), no OCP-specific policies needed."; \
	fi

.PHONY: create-cr
create-cr: ## Deploys CR from config/samples/instana_v1_instanaagent_demo.yaml (needs to be created in the workspace first)
	kubectl apply -f config/samples/instana_v1_instanaagent_demo.yaml

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

.PHONY: setup-ocp-mirror
setup-ocp-mirror: ## Setup ocp internal registry and define ImageContentSourcePolicy to pull from internal registry
	./ci/scripts/setup-ocp-mirror.sh

.PHONY: dev-run-cluster
dev-run-cluster: namespace install create-cr run ## Creates a full dev deployment on any cluster from scratch, also useful after purge

.PHONY: logs
logs: ## Tail operator logs
	kubectl logs -f deployment/instana-agent-controller-manager -n $(NAMESPACE)

##@ OLM

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: operator-sdk manifests kustomize ## Create the OLM bundle
	$(OPERATOR_SDK) generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image "instana/instana-agent-operator=$(IMG)"
	$(KUSTOMIZE) build config/manifests \
		| sed -e 's|\(replaces:.*v\)0.0.0|\1$(PREV_VERSION)|' \
		| sed -e 's|\(containerImage:[[:space:]]*\).*|\1$(IMG)|' \
		| sed -e 's|\(image:[[:space:]]*\).*instana-agent-operator:0.0.0|\1$(IMG)|' \
		| sed -e 's|\(image:[[:space:]]*\).*agent:latest|\1$(AGENT_IMG)|' \
		| $(OPERATOR_SDK) generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	./hack/patch-bundle.sh
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: buildctl ## Build the bundle image for OLM.
	#docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
	$(BUILDCTL) --addr=${CONTAINER_CMD}-container://buildkitd build --frontend gateway.v0 --opt source=docker/dockerfile --opt filename=./bundle.Dockerfile --local context=. --local dockerfile=. --output type=oci,name=${BUNDLE_IMG} | $(CONTAINER_CMD) load

controller-yaml: manifests kustomize ## Output the YAML for deployment, so it can be packaged with the release. Use `make --silent` to suppress other output.
	cd config/manager && $(KUSTOMIZE) edit set image "instana/instana-agent-operator=$(IMG)"
	$(KUSTOMIZE) build config/default

CONTROLLER_RUNTIME_VERSION := $(shell go list -m all | grep sigs.k8s.io/controller-runtime | awk '{print $$2}')

##@ Individual install targets to download binaries to ./bin-folder

.PHONY: controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	@if [ -f $(CONTROLLER_GEN) ]; then \
		echo "Controller-gen binary found in $(CONTROLLER_GEN)"; \
		version=$$($(CONTROLLER_GEN) --version | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown"); \
		if [ "$$version" = "$(CONTROLLER_GEN_VERSION)" ]; then \
			echo "Controller-gen version $(CONTROLLER_GEN_VERSION) is already installed"; \
		else \
			echo "Updating controller-gen from $$version to $(CONTROLLER_GEN_VERSION)"; \
			go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION); \
		fi \
	else \
		echo "Installing controller-gen $(CONTROLLER_GEN_VERSION)"; \
		go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_GEN_VERSION); \
	fi

.PHONY: kustomize
kustomize: ## Download kustomize locally if necessary.
	@if [ -f $(KUSTOMIZE) ]; then \
		echo "Kustomize binary found in $(KUSTOMIZE)"; \
		version=$$($(KUSTOMIZE) version | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown"); \
		if [ "$$version" = "$(KUSTOMIZE_VERSION)" ]; then \
			echo "Kustomize version $(KUSTOMIZE_VERSION) is already installed"; \
		else \
			echo "Updating kustomize from $$version to $(KUSTOMIZE_VERSION)"; \
			go install sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION); \
		fi \
	else \
		echo "Installing kustomize $(KUSTOMIZE_VERSION)"; \
		go install sigs.k8s.io/kustomize/kustomize/v4@$(KUSTOMIZE_VERSION); \
	fi

.PHONY: envtest
envtest: ## Download envtest-setup locally if necessary.
	@if [ -f $(ENVTEST) ]; then \
		echo "Envtest binary found in $(ENVTEST)"; \
	else \
		echo "Installing envtest"; \
		go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest; \
	fi

.PHONY: golanci-lint
golangci-lint: ## Download the golangci-lint linter locally if necessary.
	@if [ -f $(GOLANGCI_LINT) ]; then \
		echo "Golangci-lint binary found in $(GOLANGCI_LINT)"; \
		version=$$($(GOLANGCI_LINT) --version | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown"); \
		if [ "$$version" = "$(GOLANGCI_LINT_VERSION)" ]; then \
			echo "Golangci-lint version $(GOLANGCI_LINT_VERSION) is already installed"; \
		else \
			echo "Updating golangci-lint from $$version to $(GOLANGCI_LINT_VERSION)"; \
			go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
		fi \
	else \
		echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION); \
	fi

.PHONY: operator-sdk
operator-sdk: ## Download the Operator SDK binary locally if necessary.
	@if [ -f $(OPERATOR_SDK) ]; then \
		echo "Operator SDK binary found in $(OPERATOR_SDK)"; \
	else \
		echo "DOwnload Operator SDK for $(OS)/$(ARCH) to $(OPERATOR_SDK)"; \
		curl -Lo $(OPERATOR_SDK) https://github.com/operator-framework/operator-sdk/releases/download/v1.16.0/operator-sdk_${OS}_${ARCH}; \
		chmod +x $(OPERATOR_SDK); \
	fi


.PHONY: buildctl
BUILDKITD_CONTAINER_NAME = buildkitd
buildctl: ## Download the buildctl binary locally if necessary and prepare the container for running builds.
	@if [ -f $(BUILDCTL) ]; then \
		echo "Buildctl binary found in $(BUILDCTL)"; \
		version=$$($(BUILDCTL) -v | grep -o 'v[0-9]\+\.[0-9]\+\.[0-9]\+' || echo "unknown"); \
		if [ "$$version" = "$(BUILDCTL_VERSION)" ] || [ "$$version" = "v0.0.0" ]; then \
			echo "Buildctl version $(BUILDCTL_VERSION) is already installed"; \
		else \
			echo "Updating buildctl from $$version to $(BUILDCTL_VERSION)"; \
			go install github.com/moby/buildkit/cmd/buildctl@$(BUILDCTL_VERSION); \
		fi \
	else \
		echo "Installing buildctl $(BUILDCTL_VERSION)"; \
		go install github.com/moby/buildkit/cmd/buildctl@$(BUILDCTL_VERSION); \
	fi
	@if [ "`$(CONTAINER_CMD) ps -a -q -f name=$(BUILDKITD_CONTAINER_NAME)`" ]; then \
		echo "Ensuring buildkit container is using the correct version $(BUILDKIT_IMAGE_TAG)"; \
		$(CONTAINER_CMD) rm -f $(BUILDKITD_CONTAINER_NAME) 2>/dev/null || true; \
		echo "Starting buildkit container with version $(BUILDKIT_IMAGE_TAG)"; \
		$(CONTAINER_CMD) run -d --name buildkitd --privileged docker.io/moby/buildkit:$(BUILDKIT_IMAGE_TAG); \
		echo "Allowing 5 seconds to bootup"; \
		sleep 5; \
	else \
		echo "$(BUILDKITD_CONTAINER_NAME) container is not present, launching it now"; \
		$(CONTAINER_CMD) run -d --name buildkitd --privileged docker.io/moby/buildkit:$(BUILDKIT_IMAGE_TAG); \
		echo "Allowing 5 seconds to bootup"; \
		sleep 5; \
	fi
