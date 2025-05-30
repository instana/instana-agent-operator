# Current Operator version (override when executing Make target, e.g. like `make VERSION=2.0.0 bundle`)
VERSION ?= 0.0.1

# Previous version, will only be used for updating the "replaces" field in the ClusterServiceVersion when defined command-line
PREV_VERSION ?= 0.0.0

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= instana-agent-operator-bundle:$(VERSION)

# Include the latest Git commit SHA, gets injected in code via Docker build (just like VERSION)
GIT_COMMIT ?= $(shell git rev-parse --short HEAD)

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

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd"
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.32

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

GOPATH=$(shell go env GOPATH)

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
	"clusterrole/leader-election-role" \
	"clusterrole/instana-agent-clusterrole" \
	"clusterrolebinding/leader-election-rolebinding" \
	"clusterrolebinding/instana-agent-clusterrolebinding"

all: build


##@ General

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

setup: ## Basic project setup, e.g. installing GitHook for checking license headers
	cd .git/hooks && ln -fs ../../.githooks/* .

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=instana-agent-clusterrole webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...

lint: golangci-lint ## Run the golang-ci linter
	$(GOLANGCI_LINT) run --timeout 5m

EXCLUDED_TEST_DIRS = mocks e2e
EXCLUDE_PATTERN = $(shell echo $(EXCLUDED_TEST_DIRS) | sed 's/ /|/g')
PACKAGES = $(shell go list ./... | grep -vE "$(EXCLUDE_PATTERN)" | tr '\n' ' ')
KUBEBUILDER_ASSETS=$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)
test: gen-mocks manifests generate fmt vet lint envtest ## Run tests but ignore specific directories that match EXCLUDED_TEST_DIRS
	KUBEBUILDER_ASSETS="$(KUBEBUILDER_ASSETS)" go test $(PACKAGES) -coverprofile=coverage.out

.PHONY: e2e
e2e: ## Run end-to-end tests
	go test -timeout=30m -count=1 -failfast -v github.com/instana/instana-agent-operator/e2e

##@ Build

build: gen-mocks setup generate fmt vet ## Build manager binary.
	go build -o bin/manager *.go

run: export DEBUG_MODE=true
run: gen-mocks generate fmt vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config (run the "install" target to install CRDs into the cluster)
	go run ./

docker-build: test container-build ## Build docker image with the manager.

docker-push: ## Push the docker image with the manager.
	${CONTAINER_CMD} push ${IMG}

container-build: buildctl
	$(BUILDCTL) --addr=${CONTAINER_CMD}-container://buildkitd build --frontend=dockerfile.v0 --local context=. --local dockerfile=. --output type=oci,name=${IMG} --opt build-arg:VERSION=0.0.1 --opt build-arg:GIT_COMMIT=${GIT_COMMIT}  --opt build-arg:DATE="$$(date)" | $(CONTAINER_CMD) load

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


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.18.0)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-install-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.5)

ENVTEST = $(shell pwd)/bin/setup-envtest
envtest: ## Download envtest-setup locally if necessary.
	$(call go-install-tool,$(ENVTEST),sigs.k8s.io/controller-runtime/tools/setup-envtest@latest)

GOLANGCI_LINT = $(shell go env GOPATH)/bin/golangci-lint
# Test if golangci-lint is available in the GOPATH, if not, set to local and download if needed
ifneq ($(shell test -f $(GOLANGCI_LINT) && echo -n yes),yes)
GOLANGCI_LINT = $(shell pwd)/bin/golangci-lint
endif
golangci-lint: ## Download the golangci-lint linter locally if necessary.
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.64.4)

OPERATOR_SDK = $(shell command -v operator-sdk 2>/dev/null || echo "operator-sdk")
# Test if operator-sdk is available on the system, otherwise download locally
ifneq ($(shell test -f $(OPERATOR_SDK) && echo -n yes),yes)
OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
endif
operator-sdk: ## Download the Operator SDK binary locally if necessary.
	$(call curl-get-tool,$(OPERATOR_SDK),https://github.com/operator-framework/operator-sdk/releases/download/v1.16.0,operator-sdk_$${OS}_$${ARCH})

BUILDCTL = $(shell pwd)/bin/buildctl
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
	$(call go-install-tool,$(BUILDCTL),github.com/moby/buildkit/cmd/buildctl@v0.16)

# go-install-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-install-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin GO111MODULE=on go install $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# curl-get-tool will download the package $3 from $2 and install it to $1.
# The package name can use $${OS} and $${ARCH} to fetch the specific version (double $$ for escaping)
define curl-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
ARCH=`case $$(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $$(uname -m) ;; esac` ;\
OS=$$(uname | awk '{print tolower($$0)}') ;\
echo "Downloading $(2)/$(3)" ;\
curl -LO $(2)/$(3) ;\
curl -LO $(2)/checksums.txt ;\
grep $(3) checksums.txt | sha256sum -c - ;\
chmod +x $(3) ;\
mkdir -p $$(dirname $(1)) ;\
mv $(3) $(1) ;\
rm -rf $$TMP_DIR ;\
}
endef

.PHONY: namespace
namespace: ## Generate namespace instana-agent on OCP for manual testing
	oc new-project instana-agent || true
	oc adm policy add-scc-to-user privileged -z instana-agent -n instana-agent

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

.PHONY: dev-run-ocp
dev-run-ocp: namespace install create-cr run ## Creates a full dev deployment on OCP from scratch, also useful after purge

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

get-mockgen:
	go install go.uber.org/mock/mockgen@74a29c6e6c2cbb8ccee94db061c1604ff33fd188

gen-mocks: get-mockgen
	${GOBIN}/mockgen --source ${GOPATH}/pkg/mod/sigs.k8s.io/controller-runtime@v0.20.4/pkg/client/interfaces.go --destination ./mocks/k8s_client_mock.go --package mocks
	${GOBIN}/mockgen --source ./pkg/hash/hash.go --destination ./mocks/hash_mock.go --package mocks
	${GOBIN}/mockgen --source ./pkg/k8s/client/client.go --destination ./mocks/instana_agent_client_mock.go --package mocks
	${GOBIN}/mockgen --source ./pkg/k8s/object/transformations/pod_selector.go --destination ./mocks/pod_selector_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/transformations/transformations.go --destination ./mocks/transformations_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/ports/ports_builder.go --destination ./mocks/ports_builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/env/env_builder.go --destination ./mocks/env_builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/volume/volume_builder.go --destination ./mocks/volume_builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./mocks/helpers_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/builder/builder.go --destination ./mocks/builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/json_or_die/json.go --destination ./mocks/json_or_die_marshaler_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/operator/status/agent_status_manager.go --destination ./mocks/agent_status_manager_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/operator/lifecycle/dependent_lifecycle_manager.go --destination ./mocks/dependent_lifecycle_manager_mock.go --package mocks
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/ports/remote_ports_builder.go --destination ./mocks/remote_ports_builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/env/remote_env_builder.go --destination ./mocks/remote_env_builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/volume/remote_volume_builder.go --destination ./mocks/remote_volume_builder_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/object/builders/common/helpers/remote_helpers.go --destination ./mocks/remote_helpers_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/operator/status/remote_agent_status_manager.go --destination ./mocks/remote_agent_status_manager_mock.go --package mocks 
	${GOBIN}/mockgen --source ./pkg/k8s/operator/lifecycle/remote_dependent_lifecycle_manager.go --destination ./mocks/remote_dependent_lifecycle_manager_mock.go --package mocks