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
ENVTEST_K8S_VERSION = 1.30

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

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


all: build


##@ General

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

setup: ## Basic project setup, e.g. installing GitHook for checking license headers
	cd .git/hooks && ln -fs ../../.githooks/* .

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code
	go vet ./...

lint: golangci-lint ## Run the golang-ci linter
	$(GOLANGCI_LINT) run --timeout 5m

test: manifests generate fmt vet lint envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) -p path)" go test ./... -coverprofile cover.out


##@ Build

build: setup generate fmt vet ## Build manager binary.
	go build -o bin/manager *.go

run: export DEBUG_MODE=true
run: generate fmt vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config (run the "install" target to install CRDs into the cluster)
	go run ./

docker-build: test ## Build docker image with the manager.
	docker build --build-arg VERSION=${VERSION} --build-arg GIT_COMMIT=${GIT_COMMIT} --build-arg DATE="$$(date)" -t ${IMG} .

docker-push: ## Push the docker image with the manager.
	docker push ${IMG}


##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	kubectl apply -k config/crd

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	kubectl delete -k config/crd

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
	$(call go-install-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0)

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
	$(call go-install-tool,$(GOLANGCI_LINT),github.com/golangci/golangci-lint/cmd/golangci-lint@v1.56.2)

OPERATOR_SDK = $(shell command -v operator-sdk 2>/dev/null || echo "operator-sdk")
# Test if operator-sdk is available on the system, otherwise download locally
ifneq ($(shell test -f $(OPERATOR_SDK) && echo -n yes),yes)
OPERATOR_SDK = $(shell pwd)/bin/operator-sdk
endif
operator-sdk: ## Download the Operator SDK binary locally if necessary.
	$(call curl-get-tool,$(OPERATOR_SDK),https://github.com/operator-framework/operator-sdk/releases/download/v1.16.0,operator-sdk_$${OS}_$${ARCH})


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
bundle-build: ## Build the bundle image for OLM.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

controller-yaml: manifests kustomize ## Output the YAML for deployment, so it can be packaged with the release. Use `make --silent` to suppress other output.
	cd config/manager && $(KUSTOMIZE) edit set image "instana/instana-agent-operator=$(IMG)"
	$(KUSTOMIZE) build config/default

get-mockgen:
	# commit 2c718f249a424ac6ce6e2afa28c3c17f95c51241 introduces --write_command_comment flag to enable us to keep the mocks without additional command comments
	which mockgen >> /dev/null 2>&1 || go install go.uber.org/mock/mockgen@2c718f249a424ac6ce6e2afa28c3c17f95c51241

MOCKGEN_ARGS = --write_package_comment=false --write_source_comment=false --write_command_comment=false --copyright_file=hack/boilerplate.go.txt
gen-mocks: get-mockgen
	mockgen --source ${GOPATH}/pkg/mod/sigs.k8s.io/controller-runtime@v0.17.2/pkg/client/interfaces.go --destination ./pkg/k8s/client/k8s_client_mock_test.go --package client ${MOCKGEN_ARGS}

	mockgen --source ./pkg/hash/hash.go --destination ./pkg/k8s/object/builders/agent/daemonset/hash_mock_test.go --package daemonset ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/transformations/pod_selector.go --destination ./pkg/k8s/object/builders/agent/daemonset/pod_selector_mock_test.go --package daemonset ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/ports/ports.go --destination ./pkg/k8s/object/builders/agent/daemonset/ports_mock_test.go --package daemonset ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/env/env_builder.go --destination ./pkg/k8s/object/builders/agent/daemonset/env_builder_mock_test.go --package daemonset ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/volume/volume_builder.go --destination ./pkg/k8s/object/builders/agent/daemonset/volume_builder_mock_test.go --package daemonset ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/operator/status/status.go --destination ./pkg/k8s/object/builders/agent/daemonset/status_mock_test.go --package daemonset ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/common/env/helpers_mock_test.go --package env ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/common/volume/helpers_mock_test.go --package volume ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/client/client.go --destination ./pkg/k8s/operator/operator_utils/instana_agent_client_mock_test.go --package operator_utils ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/builder/builder.go --destination ./pkg/k8s/operator/operator_utils/builder_mock_test.go --package operator_utils ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/operator/lifecycle/lifecycle.go --destination ./pkg/k8s/operator/operator_utils/lifecycle_mock_test.go --package operator_utils ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/k8s-sensor/rbac/helpers_mock_test.go --package rbac ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/k8s-sensor/serviceaccount/helpers_mock_test.go --package serviceaccount ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/agent/serviceaccount/helpers_mock_test.go --package serviceaccount ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/k8s-sensor/configmap/helpers_mock_test.go --package configmap ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/ports/ports.go --destination ./pkg/k8s/object/builders/common/ports/ports_mock_test.go --package ports ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/helpers/agent_interfaces.go --destination ./pkg/k8s/object/builders/common/ports/agent_interfaces_mock_test.go --package ports ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/agent/headless-service/helpers_mock_test.go --package headless_service ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/transformations/pod_selector.go --destination ./pkg/k8s/object/builders/agent/headless-service/pod_selector_mock_test.go --package headless_service ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/ports/ports.go --destination ./pkg/k8s/object/builders/agent/headless-service/ports_mock_test.go --package headless_service ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/transformations/pod_selector.go --destination ./pkg/k8s/object/builders/agent/service/pod_selector_mock_test.go --package service ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/ports/ports.go --destination ./pkg/k8s/object/builders/agent/service/ports_mock_test.go --package service ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/helpers/agent_interfaces.go --destination ./pkg/k8s/object/builders/agent/service/agent_interfaces_mock_test.go --package service ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/transformations/transformations.go --destination ./pkg/k8s/object/builders/common/builder/transformations_mock_test.go --package builder ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/common/builder/builder.go --destination ./pkg/k8s/object/builders/common/builder/builder_mock_test.go --package builder ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/agent/secrets/tls-secret/helpers_mock_test.go --package tls_secret ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/agent/secrets/containers-instana-io-secret/helpers_mock_test.go --package containers_instana_io_secret ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/builders/agent/secrets/containers-instana-io-secret/docker_config_json.go --destination ./pkg/k8s/object/builders/agent/secrets/containers-instana-io-secret/docker_config_json_mock_test.go --package containers_instana_io_secret ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/operator/status/status.go --destination ./pkg/k8s/object/builders/agent/configmap/status_mock_test.go --package configmap ${MOCKGEN_ARGS}

	mockgen --source ./pkg/k8s/object/builders/common/helpers/helpers.go --destination ./pkg/k8s/object/builders/k8s-sensor/poddisruptionbudget/helpers_mock_test.go --package poddisruptionbudget ${MOCKGEN_ARGS}
	mockgen --source ./pkg/k8s/object/transformations/pod_selector.go --destination ./pkg/k8s/object/builders/k8s-sensor/poddisruptionbudget/pod_selector_mock_test.go --package poddisruptionbudget ${MOCKGEN_ARGS}




