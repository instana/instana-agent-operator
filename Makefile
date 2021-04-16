# Current Operator version
VERSION ?= 0.0.1

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= instana-agent-operator-bundle:$(VERSION)

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# Image URL to use all building/pushing image targets
IMG ?= instana-agent-operator:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


all: build

##@ General

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

setup: ## Basic project setup, e.g. installing GitHook for license headers
	cd .git/hooks && ln -fs ../../.githooks/* .

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	##@ python3 ./hack/customize_crds.py

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...


vet: ## Run go vet against code
	go vet ./...

ENVTEST_ASSETS_DIR = $(shell pwd)/testbin
test: generate fmt vet manifests ## Run tests.
	mkdir -p $(ENVTEST_ASSETS_DIR)
	test -f $(ENVTEST_ASSETS_DIR)/setup-envtest.sh || curl -sSLo $(ENVTEST_ASSETS_DIR)/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.6.3/hack/setup-envtest.sh
	source $(ENVTEST_ASSETS_DIR)/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./... -coverprofile cover.out


##@ Build

build: setup generate fmt vet ## Build manager binary.
	go build -o bin/manager *.go

run: export RUN_LOCAL=true
run: export POD_NAMESPACE=instana-agent
run: export DEBUG_OPERATOR=true
run: generate fmt vet manifests ## Run against the configured Kubernetes cluster in ~/.kube/config
	go run ./

docker-build: test ## Build docker image with the manager.
	docker build -t ${IMG} .

docker-push: ## Push the docker image with the manager.
	docker push ${IMG}


##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller in the configured Kubernetes cluster in ~/.kube/config
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the configured Kubernetes cluster in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -


CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v3@v3.8.7)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go get $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef

# Generate bundle manifests and metadata, then validate generated files.
.PHONY: bundle
bundle: manifests kustomize
	operator-sdk generate kustomize manifests -q
	cd config/olm/$(PLATFORM) && $(KUSTOMIZE) edit set image "docker.io/instana/instana-agent-operator:snapshot"="$(IMG)"
	$(KUSTOMIZE) build config/olm/$(PLATFORM) | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
	rm -rf ./config/olm/$(PLATFORM)/$(VERSION)
	mkdir -p ./config/olm/$(PLATFORM)/$(VERSION)
	mv ./bundle/* ./config/olm/$(PLATFORM)/$(VERSION)
	mv ./config/olm/$(PLATFORM)/$(VERSION)/manifests/instana-agent.clusterserviceversion.yaml ./config/olm/$(PLATFORM)/$(VERSION)/manifests/instana-agent.v$(VERSION).clusterserviceversion.yaml
	mv ./bundle.Dockerfile ./config/olm/$(PLATFORM)/bundle-$(VERSION).Dockerfile

.PHONY: bundle-build ## Build the bundle image.
bundle-build:
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMG) .
