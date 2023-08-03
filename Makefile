# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2020-2022 Intel Corporation

export APP_NAME=intel-ethernet-operator

TLS_VERIFY ?= false

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
VERSION ?= 0.0.1

# Set default K8CLI tool to 'oc' if it's not defined. To change this you can export this in env. e.g., export K8CLI=kubectl
K8CLI ?= oc

TARGET_PLATFORM ?= OCP
# Set default IMGTOOL tool to 'podman' if it's not defined. To change this you can export this in env. e.g., export IMGTOOL=docker
IMGTOOL ?= podman

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "preview,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=preview,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="preview,fast,stable")
ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

# DEFAULT_CHANNEL defines the default channel used in the bundle.
# Add a new line here if you would like to change its default config. (E.g DEFAULT_CHANNEL = "stable")
# To re-generate a bundle for any other default channel without changing the default setup, you can:
# - use the DEFAULT_CHANNEL as arg of the bundle target (e.g make bundle DEFAULT_CHANNEL=stable)
# - use environment variables to overwrite this value (e.g export DEFAULT_CHANNEL="stable")
ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

# IMAGE_TAG_BASE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# intel.com/intel-ethernet-operator-bundle:$VERSION and intel.com/intel-ethernet-operator-catalog:$VERSION.
ifneq (, $(IMAGE_REGISTRY))
IMAGE_TAG_BASE = $(IMAGE_REGISTRY)/$(APP_NAME)
else
IMAGE_TAG_BASE = $(APP_NAME)
endif

# BUNDLE_IMG defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMG=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMG ?= $(IMAGE_TAG_BASE)-bundle:v$(VERSION)

# Versioned image tag
MANAGER_IMAGE ?= $(IMAGE_TAG_BASE)-manager
IMAGE_TAG_LATEST?=$(MANAGER_IMAGE):latest
IMAGE_TAG_VERSION=$(MANAGER_IMAGE):$(VERSION)

# Image URL to use all building/pushing image targets
IMG ?= $(IMAGE_TAG_VERSION)

export ETHERNET_MANAGER_IMAGE ?= $(IMAGE_TAG_VERSION)
# dependent images
export ETHERNET_DAEMON_IMAGE ?= $(IMAGE_TAG_BASE)-daemon:$(VERSION)
export ETHERNET_NODE_LABELER_IMAGE ?= $(IMAGE_TAG_BASE)-labeler:$(VERSION)

# FlowConfigDaemon image tag
FCDAEMON_NAME ?= $(IMAGE_TAG_BASE)-flowconfig-daemon
FCDAEMON_IMAGE_TAG_LATEST ?= $(FCDAEMON_NAME):latest
FCDAEMON_IMAGE_TAG_VERSION = $(FCDAEMON_NAME):$(VERSION)
# Image URL to use all building/pushing FlowConfigDaemon image targets
FCDAEMON_IMG?=$(FCDAEMON_IMAGE_TAG_VERSION)
FCDAEMON_DOCKERFILE = images/Dockerfile.FlowConfigDaemon

UFT_IMAGE ?= dcf-tool:v22.03
ifneq (, $(IMAGE_REGISTRY))
UFT_IMAGE_URL = $(IMAGE_REGISTRY)/$(UFT_IMAGE)
else
UFT_IMAGE_URL = $(UFT_IMAGE)
endif

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# To pass proxy for docker build from env invoke make with 'make docker-build HTTP_PROXY=$http_proxy HTTPS_PROXY=$https_proxy'
DOCKERARGS?=

# Podman specific args
ifeq ($(IMGTOOL),podman)
	DOCKERARGS += --format=docker
endif

# Add proxy args for Image builder if provided
ifdef HTTP_PROXY
	DOCKERARGS += --build-arg http_proxy=${HTTP_PROXY}
endif
ifdef HTTPS_PROXY
	DOCKERARGS += --build-arg https_proxy=${HTTPS_PROXY}
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build daemon flowconfig-daemon

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

.PHONY: help
help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

.PHONY: manifests
manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	FOLDER=. COPYRIGHT_FILE=COPYRIGHT ./copyright.sh

# Generate flowconfig-daemon deployment assets
.PHONY: flowconfig-manifests
flowconfig-manifests: kustomize
	cd config/flowconfig-daemon && $(KUSTOMIZE) edit set image daemon-image=${FCDAEMON_IMG} && $(KUSTOMIZE) edit set image dcf-tool=${UFT_IMAGE_URL}
	mkdir -p assets/flowconfig-daemon
	$(KUSTOMIZE) build config/flowconfig-daemon -o assets/flowconfig-daemon/daemon.yaml
	FOLDER=. COPYRIGHT_FILE=COPYRIGHT ./copyright.sh

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.22

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.9.2
KUSTOMIZE_INSTALL_SCRIPT ?= "https://raw.githubusercontent.com/kubernetes-sigs/kustomize/master/hack/install_kustomize.sh"
.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { curl -s $(KUSTOMIZE_INSTALL_SCRIPT) | bash -s -- $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest


.PHONY: test
test: manifests flowconfig-manifests generate fmt vet envtest ## Run tests.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" ETHERNET_NAMESPACE=default go test ./... -timeout 30m -coverprofile cover.out

.PHONY: test
test_daemon: manifests generate fmt vet envtest ## Run tests only for the fwddp_daemon.
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" ETHERNET_NAMESPACE=default go test pkg/fwddp-daemon -coverprofile cover.out

.PHONY: build
build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

# Build flowconfig-daemon binary
.PHONY: flowconfig-daemon
flowconfig-daemon: generate fmt vet
	go build -o bin/flowconfig-daemon cmd/flowconfig-daemon/main.go

.PHONY: daemon
daemon: generate fmt vet
	go build -o bin/fwddp-daemon cmd/fwddp-daemon/main.go

.PHONY: run
run: manifests flowconfig-manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	$(IMGTOOL) build -t ${IMAGE_TAG_VERSION} ${DOCKERARGS} .
	$(IMGTOOL) image tag ${IMAGE_TAG_VERSION} ${IMAGE_TAG_LATEST}

.PHONY: docker-build-manager
docker-build-manager: flowconfig-manifests
	$(IMGTOOL) build --file Dockerfile --build-arg=VERSION=$(VERSION) --tag ${ETHERNET_MANAGER_IMAGE} ${DOCKERARGS} .
	$(IMGTOOL) image tag ${ETHERNET_MANAGER_IMAGE} ${IMAGE_TAG_LATEST}

.PHONY: docker-build-daemon
docker-build-daemon:
	$(IMGTOOL) build --file Dockerfile.daemon --build-arg=VERSION=$(VERSION) --tag ${ETHERNET_DAEMON_IMAGE} ${DOCKERARGS} .

.PHONY: docker-build-labeler
docker-build-labeler:
	$(IMGTOOL) build --file Dockerfile.labeler --build-arg=VERSION=$(VERSION) --tag ${ETHERNET_NODE_LABELER_IMAGE} ${DOCKERARGS} .

.PHONY: docker-push-manager
docker-push-manager:
	$(call push_image,${ETHERNET_MANAGER_IMAGE})

.PHONY: docker-push-daemon
docker-push-daemon:
	$(call push_image,${ETHERNET_DAEMON_IMAGE})

.PHONY: docker-push-labeler
docker-push-labeler:
	$(call push_image,${ETHERNET_NODE_LABELER_IMAGE})

# Build FlowConfigDaemon docker image
.PHONY: docker-build-flowconfig
docker-build-flowconfig:
	$(IMGTOOL) build . -f ${FCDAEMON_DOCKERFILE} -t ${FCDAEMON_IMG} $(DOCKERARGS)
	$(IMGTOOL) image tag ${FCDAEMON_IMG} ${FCDAEMON_IMAGE_TAG_LATEST}

.PHONY: docker-push-flowconfig
docker-push-flowconfig:
	$(call push_image,${FCDAEMON_IMG})

ifndef ignore-not-found
  ignore-not-found = false
endif

##@ Deployment
.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(K8CLI) apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(K8CLI) delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests flowconfig-manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${ETHERNET_MANAGER_IMAGE}
	$(KUSTOMIZE) build config/default | envsubst | $(K8CLI) apply -f -
	$(K8CLI) apply -f config/flowconfig-daemon/add_flowconfigdaemon.yaml
	$(K8CLI) apply -f config/flowconfig-daemon/sriov_nad.yaml

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | $(K8CLI) delete -f -

.PHONY: bundle
bundle: manifests kustomize flowconfig-manifests ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(ETHERNET_MANAGER_IMAGE)
	$(KUSTOMIZE) build config/manifests | envsubst | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	operator-sdk bundle validate ./bundle
	FOLDER=. COPYRIGHT_FILE=COPYRIGHT ./copyright.sh
	cat COPYRIGHT bundle.Dockerfile >bundle.tmp
	printf "\nCOPY COPYRIGHT /licenses/LICENSE\n" >> bundle.tmp
	mv bundle.tmp bundle.Dockerfile
.PHONY: bundle-build
bundle-build: bundle ## Build the bundle image.
	$(IMGTOOL) build -f bundle.Dockerfile -t $(BUNDLE_IMG) ${DOCKERARGS} .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(call push_image,${BUNDLE_IMG})

.PHONY: opm
OPM_VERSION = v1.15.1
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/${OPM_VERSION}/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMGS=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMGS ?= $(BUNDLE_IMG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(IMAGE_TAG_BASE)-catalog:v$(VERSION)

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMGS to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool $(IMGTOOL) --mode semver --tag $(CATALOG_IMG) --bundles $(BUNDLE_IMGS) $(if ifeq $(TLS_VERIFY) false, --skip-tls) $(FROM_INDEX_OPT)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(call push_image,${CATALOG_IMG})

.PHONY: build-all
build_all: docker-build-manager docker-build-daemon docker-build-labeler docker-build-flowconfig bundle-build

.PHONY: push_all
push_all: docker-push-manager docker-push-daemon docker-push-labeler docker-push-flowconfig bundle-push

define push_image
	$(if $(filter $(IMGTOOL), podman), $(IMGTOOL) push $(1) --tls-verify=$(TLS_VERIFY), $(IMGTOOL) push $(1))
endef

# The gen-proto target below is ony required to rebuild the protobuf for UFT
#
# Install protoc compiler:
# http://google.github.io/proto-lens/installing-protoc.html
#
# PROTOC_ZIP=protoc-3.14.0-linux-x86_64.zip
# curl -OL https://github.com/protocolbuffers/protobuf/releases/download/v3.14.0/$PROTOC_ZIP
# sudo unzip -o $PROTOC_ZIP -d /usr/local bin/protoc
# sudo unzip -o $PROTOC_ZIP -d /usr/local 'include/*'
# rm -f $PROTOC_ZIP
#
# Install protoc-gen-go and protoc-gen-go-grpc plugins
# https://grpc.io/docs/languages/go/quickstart/
#
# $ go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.26
# $ go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@v1.1
#
# From <https://grpc.io/docs/languages/go/quickstart/>
.PHONY: gen-proto
gen-proto:
	protoc --go_out=. --go_opt=paths=source_relative \
	--go_opt=Mpkg/flowconfig/rpc/v1/flow/flow.proto=github.com/intel-collab/applications.orchestration.operators.intel-ethernet-operator/apis/flowconfig/v1/flow \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/flowconfig/rpc/v1/flow/flow.proto
