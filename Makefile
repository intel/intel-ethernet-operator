# SPDX-License-Identifier: Apache-2.0
# Copyright (c) 2021 Intel Corporation

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

# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# To pass proxy for docker build from env invoke make with 'make docker-build HTTP_PROXY=$http_proxy HTTPS_PROXY=$https_proxy'
DOCKERARGS?=
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

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	FOLDER=. COPYRIGHT_FILE=COPYRIGHT ./copyright.sh

# Generate flowconfig-daemon deployment assets
.PHONY: flowconfig-manifests
flowconfig-manifests: manifests kustomize
	cd config/flowconfig-daemon && $(KUSTOMIZE) edit set image daemon-image=${FCDAEMON_IMG}
	$(KUSTOMIZE) build config/flowconfig-daemon -o assets/flowconfig-daemon/daemon.yaml
	FOLDER=. COPYRIGHT_FILE=COPYRIGHT ./copyright.sh

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: ## Run go fmt against code.
	go fmt ./...

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests generate fmt vet ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); ETHERNET_NAMESPACE=default go test ./... -coverprofile cover.out -v

test_daemon: manifests generate fmt vet ## Run tests only for the fwddp_daemon.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.7.2/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); cd pkg/fwddp-daemon && ETHERNET_NAMESPACE=default go test ./... -v

##@ Build

build: generate fmt vet ## Build manager binary.
	go build -o bin/manager main.go

# Build flowconfig-daemon binary
flowconfig-daemon: generate fmt vet
	go build -o bin/flowconfig-daemon cmd/flowconfig-daemon/main.go
daemon: generate fmt vet
	go build -o bin/fwddp-daemon cmd/fwddp-daemon/main.go

run: manifests generate fmt vet ## Run a controller from your host.
	go run ./main.go

docker-build: test ## Build docker image with the manager.
	$(IMGTOOL) build -t ${IMAGE_TAG_VERSION} ${DOCKERARGS} .
	$(IMGTOOL) image tag ${IMAGE_TAG_VERSION} ${IMAGE_TAG_LATEST}

docker-build-manager:
	$(IMGTOOL) build --file Dockerfile --build-arg=VERSION=$(VERSION) --tag ${ETHERNET_MANAGER_IMAGE} ${DOCKERARGS} .
	$(IMGTOOL) image tag ${ETHERNET_MANAGER_IMAGE} ${IMAGE_TAG_LATEST}

docker-build-daemon:
	$(IMGTOOL) build --file Dockerfile.daemon --build-arg=VERSION=$(VERSION) --tag ${ETHERNET_DAEMON_IMAGE} ${DOCKERARGS} .

docker-build-labeler:
	$(IMGTOOL) build --file Dockerfile.labeler --build-arg=VERSION=$(VERSION) --tag ${ETHERNET_NODE_LABELER_IMAGE} ${DOCKERARGS} .

docker-push-manager:
	$(call push_image,${ETHERNET_MANAGER_IMAGE})

docker-push-daemon:
	$(call push_image,${ETHERNET_DAEMON_IMAGE})

docker-push-labeler:
	$(call push_image,${ETHERNET_NODE_LABELER_IMAGE})

# Build FlowConfigDaemon docker image
.PHONY: docker-build-flowconfig
docker-build-flowconfig:
	$(IMGTOOL) build . -f ${FCDAEMON_DOCKERFILE} -t ${FCDAEMON_IMG} $(DOCKERARGS)
	$(IMGTOOL) image tag ${FCDAEMON_IMG} ${FCDAEMON_IMAGE_TAG_LATEST}

docker-push-flowconfig:
	$(call push_image,${FCDAEMON_IMG})

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(K8CLI) apply -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | $(K8CLI) delete -f -


deploy: manifests flowconfig-manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=${ETHERNET_MANAGER_IMAGE}
	$(KUSTOMIZE) build config/default | envsubst | $(K8CLI) apply -f -
	$(K8CLI) apply -f config/flowconfig-daemon/add_flowconfigdaemon.yaml
	$(K8CLI) apply -f config/flowconfig-daemon/sriov_nad.yaml

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | $(K8CLI) delete -f -

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

.PHONY: bundle
bundle: manifests kustomize ## Generate bundle manifests and metadata, then validate generated files.
	operator-sdk generate kustomize manifests -q
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(ETHERNET_MANAGER_IMAGE)
	$(KUSTOMIZE) build config/manifests | envsubst | operator-sdk generate bundle -q --overwrite --version $(VERSION) $(BUNDLE_METADATA_OPTS)
	cp config/metadata/dependencies.yaml bundle/metadata/dependencies.yaml
	operator-sdk bundle validate ./bundle
	FOLDER=. COPYRIGHT_FILE=COPYRIGHT ./copyright.sh
	cat COPYRIGHT bundle.Dockerfile >bundle.tmp
	printf "\nCOPY COPYRIGHT /licenses/LICENSE\n" >> bundle.tmp
	mv bundle.tmp bundle.Dockerfile

.PHONY: bundle-build
bundle-build: bundle ## Build the bundle image.
	$(IMGTOOL) build -f bundle.Dockerfile -t $(BUNDLE_IMG) .

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(call push_image,${BUNDLE_IMG})

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.15.1/$${OS}-$${ARCH}-opm ;\
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

build_all: docker-build-manager docker-build-daemon docker-build-labeler docker-build-flowconfig bundle-build

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
	--go_opt=Mpkg/flowconfig/rpc/v1/flow/flow.proto=github.com/otcshare/intel-ethernet-operator/apis/flowconfig/v1/flow \
	--go-grpc_out=. --go-grpc_opt=paths=source_relative pkg/flowconfig/rpc/v1/flow/flow.proto
