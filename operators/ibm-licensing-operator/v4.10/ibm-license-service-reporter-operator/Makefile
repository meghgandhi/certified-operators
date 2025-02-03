#
# Licensed Materials - Property of IBM
# (C) Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#

# VERSION defines the project version for the bundle.
# Update this value when you upgrade the version of your project.
# To re-generate a bundle for another specific version without changing the standard setup, you can:
# - use the VERSION as arg of the bundle target (e.g make bundle VERSION=0.0.2)
# - use environment variables to overwrite this value (e.g export VERSION=0.0.2)
CSV_VERSION ?= $(shell cat ./version/version.go | grep "Version =" | awk '{ print $$3}' | tr -d '"')
COMMIT_VERSION ?= $(shell git describe --exact-match 2> /dev/null || \
                 git describe --match=$(git rev-parse --short=8 HEAD) --always --abbrev=8)

CHANNELS ?= v4.2
DEFAULT_CHANNEL ?= v4.2

# CHANNELS define the bundle channels used in the bundle.
# Add a new line here if you would like to change its default config. (E.g CHANNELS = "candidate,fast,stable")
# To re-generate a bundle for other specific channels without changing the standard setup, you can:
# - use the CHANNELS as arg of the bundle target (e.g make bundle CHANNELS=candidate,fast,stable)
# - use environment variables to overwrite this value (e.g export CHANNELS="candidate,fast,stable")
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

REGISTRY_BASE ?= docker-na-public.artifactory.swg-devops.com
REGISTRY_INTEGRATION ?= ${REGISTRY_BASE}/hyc-cloud-private-integration-docker-local/ibmcom
REGISTRY_EDGE ?= ${REGISTRY_BASE}/hyc-cloud-private-edge-docker-local/ibmcom
REGISTRY_SCRATCH ?= ${REGISTRY_BASE}/hyc-cloud-private-scratch-docker-local/ibmcom
IMAGE_TAG ?= $(COMMIT_VERSION)

# Identify brach name to tag dev images
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(GIT_BRANCH),HEAD)
	GIT_BRANCH=$(TRAVIS_BRANCH)
endif

GIT_BRANCH_ORIG := $(GIT_BRANCH)

# Renovate generates branch names with special characters (e.g., "/").
# This replaces all special characters other than '-' and '.' with '-' to ensure the valid Docker tag is used.
GIT_BRANCH:=$(shell echo $(GIT_BRANCH_ORIG) | sed 's/[^a-zA-Z0-9.-]/-/g')

echo "Sanitized branch name that will be used as image tag: ${GIT_BRANCH}"

# OPERATOR_IMAGE defines the docker.io namespace and part of the image name for remote images.
# This variable is used to construct full image tags for bundle and catalog images.
#
# For example, running 'make bundle-build bundle-push catalog-build catalog-push' will build and push both
# $OPERATOR_IMAGE-bundle:$VERSION and $OPERATOR_IMAGE-catalog:$VERSION.
OPERATOR_IMAGE ?= $(REGISTRY_INTEGRATION)/ibm-license-service-reporter-operator
OPERATOR_IMAGE_DEV ?= $(REGISTRY_SCRATCH)/ibm-license-service-reporter-operator
OPERATOR_IMAGE_ICR ?= icr.io/cpopen/ibm-license-service-reporter-operator

# BUNDLE_IMAGE defines the image:tag used for the bundle.
# You can use it as an arg. (E.g make bundle-build BUNDLE_IMAGE=<some-registry>/<project-name-bundle>:<tag>)
BUNDLE_IMAGE ?= $(OPERATOR_IMAGE)-bundle
BUNDLE_IMAGE_DEV ?= $(OPERATOR_IMAGE_DEV)-bundle

# BUNDLE_GEN_FLAGS are the flags passed to the operator-sdk generate bundle command
BUNDLE_GEN_FLAGS ?= -q --overwrite --extra-service-accounts ibm-license-service-reporter --version $(CSV_VERSION) $(BUNDLE_METADATA_OPTS)

# USE_IMAGE_DIGESTS defines if images are resolved via tags or digests
# You can enable this value if you would like to use SHA Based Digests
# To enable set flag to true
USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

# Image URL to use all building/pushing image targets
# TAGGED_IMAGE ?= $(OPERATOR_IMAGE):$(COMMIT_VERSION)
# TAGGED_IMAGE_DEV ?= $(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)
# TAGGED_IMAGE_ICR ?= $(OPERATOR_IMAGE_ICR):$(IMAGE_TAG) # Official bundle must refer to icr.io operator mage
# ENVTEST_K8S_VERSION refers to the version of kubebuilder assets to be downloaded by envtest binary.
ENVTEST_K8S_VERSION = 1.25.0

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

ARCH := $(shell uname -m)
LOCAL_ARCH := "amd64"
ifeq ($(ARCH),x86_64)
    LOCAL_ARCH="amd64"
else ifeq ($(ARCH),ppc64le)
    LOCAL_ARCH="ppc64le"
else ifeq ($(ARCH),s390x)
    LOCAL_ARCH="s390x"
else ifeq ($(ARCH),arm64)
    LOCAL_ARCH="arm64"
else
    $(error "This system's ARCH $(ARCH) isn't recognized/supported")
endif

OS := $(shell uname)
ifeq ($(OS),Linux)
    LOCAL_OS ?= linux
    XARGS_FLAGS="-r"
	STRIP_FLAGS=
else ifeq ($(OS),Darwin)
    LOCAL_OS ?= darwin
    XARGS_FLAGS=
	STRIP_FLAGS="-x"
else
    $(error "This system's OS $(OS) isn't recognized/supported")
endif


# Setup DOCKER_BUILD_OPTS after all includes complete
# Variables for redhat ubi certification required labels
IMAGE_NAME=$(OPERATOR_IMAGE):$(IMAGE_TAG)
IMAGE_NAME_DEV=$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)
IMAGE_DISPLAY_NAME=IBM License Service Reporter Operator
IMAGE_MAINTAINER=talk2sam@us.ibm.com
IMAGE_VENDOR=IBM
IMAGE_DESCRIPTION=Operator used to install IBM License Service Reporter.
IMAGE_SUMMARY=$(IMAGE_DESCRIPTION)
IMAGE_OPENSHIFT_TAGS=licensing
$(eval WORKING_CHANGES := $(shell git status --porcelain))
$(eval BUILD_DATE := $(shell date +%Y/%m/%d@%H:%M:%S))
$(eval GIT_COMMIT := $(shell git rev-parse --short HEAD))
$(eval VCS_REF := $(GIT_COMMIT))
IMAGE_RELEASE=$(VCS_REF)
IMAGE_VERSION=$(IMAGE_TAG)
IMAGE_BUILDDATE=$(BUILD_DATE)
GIT_REMOTE_URL = $(shell git config --get remote.origin.url)
POSTGRES_USER=postgres
REPORTER_USER=license-administrator

$(eval DOCKER_BUILD_OPTS := --build-arg "IMAGE_NAME=$(IMAGE_NAME)" --build-arg "IMAGE_DISPLAY_NAME=$(IMAGE_DISPLAY_NAME)" --build-arg "IMAGE_MAINTAINER=$(IMAGE_MAINTAINER)" --build-arg "IMAGE_VENDOR=$(IMAGE_VENDOR)" --build-arg "IMAGE_VERSION=$(IMAGE_VERSION)" --build-arg "VERSION=$(CSV_VERSION)" --build-arg "IMAGE_RELEASE=$(IMAGE_RELEASE)"  --build-arg "IMAGE_BUILDDATE=$(IMAGE_BUILDDATE)" --build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" --build-arg "IMAGE_SUMMARY=$(IMAGE_SUMMARY)" --build-arg "IMAGE_OPENSHIFT_TAGS=$(IMAGE_OPENSHIFT_TAGS)" --build-arg "VCS_REF=$(VCS_REF)" --build-arg "VCS_URL=$(GIT_REMOTE_URL)" --build-arg "IMAGE_NAME_ARCH=$(IMAGE_NAME)-$(LOCAL_ARCH)" --build-arg "POSTGRES_USER=$(POSTGRES_USER)" --build-arg "REPORTER_USER=$(REPORTER_USER)")
$(eval DOCKER_BUILD_OPTS_DEV := --build-arg "IMAGE_NAME=$(IMAGE_NAME_DEV)" --build-arg "IMAGE_DISPLAY_NAME=$(IMAGE_DISPLAY_NAME)" --build-arg "IMAGE_MAINTAINER=$(IMAGE_MAINTAINER)" --build-arg "IMAGE_VENDOR=$(IMAGE_VENDOR)" --build-arg "IMAGE_VERSION=$(IMAGE_VERSION)" --build-arg "VERSION=$(CSV_VERSION)" --build-arg "IMAGE_RELEASE=$(IMAGE_RELEASE)"  --build-arg "IMAGE_BUILDDATE=$(IMAGE_BUILDDATE)" --build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" --build-arg "IMAGE_SUMMARY=$(IMAGE_SUMMARY)" --build-arg "IMAGE_OPENSHIFT_TAGS=$(IMAGE_OPENSHIFT_TAGS)" --build-arg "VCS_REF=$(VCS_REF)" --build-arg "VCS_URL=$(GIT_REMOTE_URL)" --build-arg "IMAGE_NAME_ARCH=$(IMAGE_NAME_DEV)-$(LOCAL_ARCH)" --build-arg "POSTGRES_USER=$(POSTGRES_USER)" --build-arg "REPORTER_USER=$(REPORTER_USER)")


TARGETOS ?= linux
TARGETARCH ?= amd64

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

.PHONY: all
all: build

.PHONY: all-dev
all-dev: docker-login build-amd64 docker-buildx-with-push-dev bundle-build-dev bundle-push-dev catalog-build-dev catalog-push-dev

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
manifests: controller-gen yq ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(YQ) -i '.metadata.annotations."olm.skipRange" = ">=1.0.0 <$(CSV_VERSION)"' ./config/manifests/bases/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(YQ) -i '.metadata.annotations.containerImage = "$(OPERATOR_IMAGE_ICR):$(CSV_VERSION)"' ./config/manifests/bases/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases

.PHONY: generate
generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."
	$(MAKE) generate-yaml-no-olm

.PHONY: fmt
fmt: ## Run go fmt against code.
	go fmt ./...

.PHONY: vet
vet: ## Run go vet against code.
	go vet ./...

.PHONY: test
test: manifests generate fmt vet envtest ## Run tests.
	@echo $(POSTGRES_USER) > /tmp/POSTGRES_USER
	@echo $(REPORTER_USER) > /tmp/REPORTER_USER
	KUBEBUILDER_ASSETS="$(shell $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path)" go test ./... -coverprofile cover.out

##@ Build

.PHONY: build
build: generate fmt vet ## Build manager binary. TODO export container images
	go build -o bin/manager-$(LOCAL_OS)-$(LOCAL_ARCH) main.go

.PHONY: build-target
build-target: generate fmt vet ## Build manager binary.
	CGO_ENABLED=0 GOOS=$(TARGETOS) GOARCH=$(TARGETARCH) go build -a -o bin/manager-$(TARGETOS)-$(TARGETARCH) main.go

.PHONY: run
run: manifests generate fmt vet ## Run a controller from your host. Adjust namespace variable according to your environment, e.g. NAMESPACE=lsr-ns make run
	IBM_POSTGRESQL_IMAGE=docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-scratch-docker-local/ibmcom/ibm-postgresql:master \
	IBM_LICENSE_SERVICE_REPORTER_IMAGE=docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-scratch-docker-local/ibmcom/ibm-license-service-reporter:master \
	IBM_LICENSE_SERVICE_REPORTER_UI_IMAGE=docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-scratch-docker-local/ibmcom/ibm-license-service-reporter-ui:master \
	IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE=docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-integration-docker-local/ibmcom/ibm-license-service-reporter-oauth2-proxy:master \
	NAMESPACE=${NAMESPACE} OPERATOR_NAMESPACE=${NAMESPACE} go run ./main.go

.PHONY: docker-login
docker-login:
	docker login -u $(ARTIFACTORY_USERNAME) -p $(ARTIFACTORY_TOKEN) $(REGISTRY_BASE)

# If you wish built the manager image targeting other platforms you can use the --platform flag.
# (i.e. docker build --platform linux/arm64 ). However, you must enable docker buildKit for it.
# More info: https://docs.docker.com/develop/develop-images/build_enhancements/
.PHONY: docker-build
docker-build: test ## Build docker image with the manager.
	docker build $(DOCKER_BUILD_OPTS) --build-arg "TARGETOS=$(TARGETOS)" --build-arg "TARGETARCH=$(TARGETARCH)" -t $(OPERATOR_IMAGE):$(IMAGE_TAG) -t $(OPERATOR_IMAGE):$(CSV_VERSION) .

.PHONY: docker-build-dev
docker-build-dev: test build ## Build docker image with the manager.
	docker build $(DOCKER_BUILD_OPTS_DEV) --build-arg "TARGETOS=$(TARGETOS)" --build-arg "TARGETARCH=$(TARGETARCH)" -t $(OPERATOR_IMAGE_DEV):$(IMAGE_TAG) -t $(OPERATOR_IMAGE_DEV):$(GIT_BRANCH) .

.PHONY: docker-push
docker-push: ## Push docker image with the manager. Use only with image passed as IMG parameter.
	docker push $(IMG)

.PHONY: build-amd64
build-amd64: test
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o bin/manager-linux-amd64 main.go

# PLATFORMS defines the target platforms for  the manager image be build to provide support to multiple
# architectures. (i.e. make docker-buildx IMG=myregistry/mypoperator:0.0.1). To use this option you need to:
# - able to use docker buildx . More info: https://docs.docker.com/build/buildx/
# - have enable BuildKit, More info: https://docs.docker.com/develop/develop-images/build_enhancements/
# - be able to push the image for your registry (i.e. if you do not inform a valid value via IMG=<myregistry/image:<tag>> than the export will fail)
# To properly provided solutions that supports more than one platform you should use this option.
PLATFORMS ?= linux/amd64
.PHONY: docker-buildx-with-push
docker-buildx-with-push: ## Build and push docker image for the manager for cross-platform support
	- docker buildx create --name project-v3-builder
	- docker buildx use project-v3-builder
	docker buildx build $(DOCKER_BUILD_OPTS) --push --platform $(PLATFORMS) --tag $(OPERATOR_IMAGE):$(IMAGE_TAG) --tag $(OPERATOR_IMAGE):$(CSV_VERSION) --progress plain .
	- docker buildx rm project-v3-builder

.PHONY: docker-buildx-with-push-dev
docker-buildx-with-push-dev:
	- docker buildx create --name project-v3-builder
	- docker buildx use project-v3-builder
	docker buildx build $(DOCKER_BUILD_OPTS_DEV) --push --platform $(PLATFORMS) --tag $(OPERATOR_IMAGE_DEV):$(IMAGE_TAG) --tag $(OPERATOR_IMAGE_DEV):$(GIT_BRANCH) --progress plain .
	- docker buildx rm project-v3-builder

##@ Deployment

ifndef ignore-not-found
  ignore-not-found = false
endif

.PHONY: install
install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

.PHONY: deploy
deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE):$(CSV_VERSION)
	$(KUSTOMIZE) build config/default | kubectl apply -f -

.PHONY: undeploy
undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config. Call with ignore-not-found=true to ignore resource not found errors during deletion.
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=$(ignore-not-found) -f -

##@ Build Dependencies

## Location to install dependencies to
LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

LOCALSCRIPTS ?= $(shell pwd)/common/scripts

## Tool Binaries
KUSTOMIZE ?= $(LOCALBIN)/kustomize
CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
ENVTEST ?= $(LOCALBIN)/setup-envtest
OPERATOR_SDK ?= $(LOCALBIN)/operator-sdk
YQ ?= $(LOCALBIN)/yq

## Tool Versions
KUSTOMIZE_VERSION ?= v3.8.7
CONTROLLER_TOOLS_VERSION ?= v0.14.0

.PHONY: kustomize
kustomize: $(KUSTOMIZE) ## Download kustomize locally if necessary.
$(KUSTOMIZE): $(LOCALBIN)
	test -s $(LOCALBIN)/kustomize || { $(LOCALSCRIPTS)/install_kustomize.sh $(subst v,,$(KUSTOMIZE_VERSION)) $(LOCALBIN); }

YQ_VERSION ?= v4.30.5

.PHONY: yq
yq: $(YQ) ## Download yq locally if necessary.
$(YQ): $(LOCALBIN)
	test -s $(LOCALBIN)/yq || { $(LOCALSCRIPTS)/install_yq.sh "$(LOCALBIN)" "$(LOCAL_ARCH)" "$(LOCAL_OS)" "$(YQ_VERSION)"; }


.PHONY: operator-sdk
operator-sdk: $(OPERATOR_SDK) ## Download operator-sdk locally if necessary.
$(OPERATOR_SDK): $(LOCALBIN)
	test -s $(LOCALBIN)/operator-sdk || { $(LOCALSCRIPTS)/install_operator_sdk.sh $(LOCALBIN) $(LOCAL_ARCH); }

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN) ## Download controller-gen locally if necessary.
$(CONTROLLER_GEN): $(LOCALBIN)
	test -s $(LOCALBIN)/controller-gen || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: envtest
envtest: $(ENVTEST) ## Download envtest-setup locally if necessary.
$(ENVTEST): $(LOCALBIN)
	test -s $(LOCALBIN)/setup-envtest || GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@latest

.PHONY: bundle
bundle: manifests kustomize operator-sdk ## Generate bundle manifests and metadata, then validate generated files.
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(YQ) -i '.spec.relatedImages = load("./common/relatedImages.yaml")' ./config/manifests/bases/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE_ICR):$(CSV_VERSION)
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(YQ) -i '.spec.relatedImages = load("./common/relatedImages.yaml")' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(OPERATOR_SDK) bundle validate ./bundle

.PHONY: bundle-build
bundle-build: ## Build the bundle image. If RETAG is true, then operator image in CSV will be replaced with one from integration registry.
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMAGE):$(IMAGE_TAG) -t $(BUNDLE_IMAGE):$(CSV_VERSION) .

# images of containers used in the development catalogsource
IBM_POSTGRESQL_IMAGE ?= ibm-postgresql
IBM_LICENSE_SERVICE_REPORTER_IMAGE ?= ibm-license-service-reporter
IBM_LICENSE_SERVICE_REPORTER_UI_IMAGE ?= ibm-license-service-reporter-ui
IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE ?= ibm-license-service-reporter-oauth2-proxy
BRANCH ?= master

.PHONY: bundle-build-dev
bundle-build-dev: ## Build the bundle image.
	@echo -e "Building development bundle with scratch images \nFollowing operand container images will be used:"
	@echo "${REGISTRY_SCRATCH}/${IBM_POSTGRESQL_IMAGE}:${BRANCH}"
	@echo "${REGISTRY_SCRATCH}/${IBM_LICENSE_SERVICE_REPORTER_IMAGE}:${BRANCH}"
	@echo "${REGISTRY_SCRATCH}/${IBM_LICENSE_SERVICE_REPORTER_UI_IMAGE}:${BRANCH}"
	@echo -e "${REGISTRY_INTEGRATION}/${IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE}:${BRANCH}\n"

  ## Replace related images with scratch development images of containers
	$(YQ) -i '.spec.relatedImages = load("./common/relatedImagesDevelopment.yaml")' ./config/manifests/bases/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.relatedImages[0].image = "$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)"' ./config/manifests/bases/ibm-license-service-reporter-operator.clusterserviceversion.yaml

	$(YQ) -i '.spec.relatedImages = load("./common/relatedImagesDevelopment.yaml")' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.relatedImages[0].image = "$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml

	$(MAKE) retag-operator IMG=$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)
	$(YQ) -i '.spec.install.spec.deployments[0].spec.template.spec.containers[0].env[0].value = "${REGISTRY_SCRATCH}/${IBM_POSTGRESQL_IMAGE}:${BRANCH}"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.install.spec.deployments[0].spec.template.spec.containers[0].env[1].value = "${REGISTRY_SCRATCH}/${IBM_LICENSE_SERVICE_REPORTER_IMAGE}:${BRANCH}"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.install.spec.deployments[0].spec.template.spec.containers[0].env[2].value = "${REGISTRY_SCRATCH}/${IBM_LICENSE_SERVICE_REPORTER_UI_IMAGE}:${BRANCH}"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.install.spec.deployments[0].spec.template.spec.containers[0].env[3].value = "${REGISTRY_INTEGRATION}/${IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE}:${BRANCH}"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml
	docker build -f bundle.Dockerfile -t $(BUNDLE_IMAGE_DEV):$(IMAGE_TAG) -t $(BUNDLE_IMAGE_DEV):$(GIT_BRANCH) .

.PHONY: retag-operator
retag-operator: yq ## Retags manager image to the current $IMG.
	$(YQ) -i '(.spec.install.spec.deployments[0].spec.template.spec.containers[] | select(.name == "ibm-license-service-reporter-operator") | .image) = "$(IMG)"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml

# .PHONY: retag-auth
# retag-auth: yq
# 	$(YQ) -i ('spec.install.spec.deployments[0].spec.template.spec.containers[] | select(.name == "ibm-license-service-reporter-operator") |  .env[] | select(.name == "IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE") | .value) = "$(IMG)"' ./bundle/manifests/ibm-license-service-reporter-operator.clusterserviceversion.yaml

.PHONY: bundle-push
bundle-push: ## Push the bundle image.
	$(MAKE) docker-push IMG=$(BUNDLE_IMAGE):$(IMAGE_TAG)
	$(MAKE) docker-push IMG=$(BUNDLE_IMAGE):$(CSV_VERSION)

.PHONY: bundle-push-dev
bundle-push-dev: ## Push the bundle image
	$(MAKE) docker-push IMG=$(BUNDLE_IMAGE_DEV):$(IMAGE_TAG) 
	$(MAKE) docker-push IMG=$(BUNDLE_IMAGE_DEV):$(GIT_BRANCH)

.PHONY: opm
OPM = ./bin/opm
opm: ## Download opm locally if necessary.
ifeq (,$(wildcard $(OPM)))
ifeq (,$(shell which opm 2>/dev/null))
	@{ \
	set -e ;\
	mkdir -p $(dir $(OPM)) ;\
	OS=$(shell go env GOOS) && ARCH=$(shell go env GOARCH) && \
	curl -sSLo $(OPM) https://github.com/operator-framework/operator-registry/releases/download/v1.23.0/$${OS}-$${ARCH}-opm ;\
	chmod +x $(OPM) ;\
	}
else
OPM = $(shell which opm)
endif
endif

# A comma-separated list of bundle images (e.g. make catalog-build BUNDLE_IMAGES=example.com/operator-bundle:v0.1.0,example.com/operator-bundle:v0.2.0).
# These images MUST exist in a registry and be pull-able.
BUNDLE_IMAGES ?= $(BUNDLE_IMAGE):$(IMAGE_TAG)
BUNDLE_IMAGES_DEV ?= $(BUNDLE_IMAGE_DEV):$(IMAGE_TAG)

# The image tag given to the resulting catalog image (e.g. make catalog-build CATALOG_IMG=example.com/operator-catalog:v0.2.0).
CATALOG_IMG ?= $(OPERATOR_IMAGE)-catalog
CATALOG_IMG_DEV ?= $(OPERATOR_IMAGE_DEV)-catalog

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMAGES to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

ifneq ($(origin CATALOG_BASE_IMG_DEV), undefined)
FROM_INDEX_OPT_DEV := --from-index $(CATALOG_BASE_IMG_DEV)
endif

# Build a catalog image by adding bundle images to an empty catalog using the operator package manager tool, 'opm'.
# This recipe invokes 'opm' in 'semver' bundle add mode. For more information on add modes, see:
# https://github.com/operator-framework/community-operators/blob/7f1438c/docs/packaging-operator.md#updating-your-existing-operator
.PHONY: catalog-build
catalog-build: opm ## Build a catalog image.
	$(OPM) index add --container-tool docker --tag $(CATALOG_IMG):$(IMAGE_TAG) --bundles $(BUNDLE_IMAGES) $(FROM_INDEX_OPT)
	docker tag $(CATALOG_IMG):$(IMAGE_TAG) $(CATALOG_IMG):$(CSV_VERSION)

.PHONY: catalog-build-dev
catalog-build-dev: opm
	$(OPM) index add --container-tool docker --tag $(CATALOG_IMG_DEV):$(IMAGE_TAG) --bundles $(BUNDLE_IMAGES_DEV) $(FROM_INDEX_OPT_DEV)
	docker tag $(CATALOG_IMG_DEV):$(IMAGE_TAG) $(CATALOG_IMG_DEV):$(GIT_BRANCH)

# Push the catalog image.
.PHONY: catalog-push
catalog-push: ## Push a catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG):$(IMAGE_TAG)
	$(MAKE) docker-push IMG=$(CATALOG_IMG):$(CSV_VERSION)

.PHONY: catalog-push-dev
catalog-push-dev: ## Push dev catalog image.
	$(MAKE) docker-push IMG=$(CATALOG_IMG_DEV):$(IMAGE_TAG)
	$(MAKE) docker-push IMG=$(CATALOG_IMG_DEV):$(GIT_BRANCH)

update-docker-manifest:
	$(MAKE) docker-manifest-build-push MANIFEST=$(OPERATOR_IMAGE):$(CSV_VERSION) IMAGES=$(OPERATOR_IMAGE):$(IMAGE_TAG)

update-docker-manifest-dev:
	$(MAKE) docker-manifest-build-push MANIFEST=$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH) IMAGES=$(OPERATOR_IMAGE_DEV):$(IMAGE_TAG)

# Requires passing variables: 
# MANIFEST - manifest name: artifactory/image_name:tag 
# IMAGES   - images contained by the manifest, separated by spaces
.PHONY: docker-manifest-build-push
docker-manifest-build-push: 
	docker manifest create $(MANIFEST) $(IMAGES)
	docker manifest push $(MANIFEST)

.PHONY: generate-yaml-no-olm
generate-yaml-no-olm: yq kustomize
	$(KUSTOMIZE) build config/manifests \
	| $(YQ) 'select(.kind != "Configuration" and .kind != "ClusterServiceVersion" and .kind != "IBMLicenseServiceReporter")' \
	| sed "s/namespace: ibm-licensing/namespace: \$$LSR_NAMESPACE/g" \
	| sed "s/name: ibm-licensing/name: \$$LSR_NAMESPACE/g" > common/yamls/install-license-service-reporter-no-olm.yaml
	cat common/yamls/install-license-service-reporter-no-olm.yaml \
	| sed "s/icr\.io\/cpopen\/cpfs/\$$\{MY_DOCKER_REGISTRY\}/g" \
	| sed "s/icr\.io\/cpopen/\$$\{MY_DOCKER_REGISTRY\}/g" > common/yamls/install-license-service-reporter-no-olm-offline.yaml

.PHONY: generate-yaml-argo-cd
generate-yaml-argo-cd: yq kustomize
	@mkdir -p argo-cd && $(KUSTOMIZE) build config/manifests > argo-cd/tmp.yaml

	# Split the resources into separate YAML files
	@(echo "---" && $(YQ) 'select(.kind == "ClusterRole" or .kind == "ClusterRoleBinding")' argo-cd/tmp.yaml) > argo-cd/cluster-rbac.yaml
	@(echo "---" && $(YQ) 'select(.kind == "IBMLicenseServiceReporter")' argo-cd/tmp.yaml) > argo-cd/cr.yaml
	@(echo "---" && $(YQ) 'select(.kind == "CustomResourceDefinition")' argo-cd/tmp.yaml) > argo-cd/crd.yaml
	@(echo "---" && $(YQ) 'select(.kind == "Deployment")' argo-cd/tmp.yaml) > argo-cd/deployment.yaml
	@(echo "---" && $(YQ) 'select(.kind == "Role" or .kind == "RoleBinding")' argo-cd/tmp.yaml) > argo-cd/rbac.yaml
	@(echo "---" && $(YQ) 'select(.kind == "ServiceAccount")' argo-cd/tmp.yaml) > argo-cd/serviceaccounts.yaml

	# Add missing namespaces
	@$(YQ) -i 'select(.kind == "ClusterRoleBinding").subjects[0].namespace = "sed-me"' argo-cd/cluster-rbac.yaml
	@$(YQ) -i '.metadata.namespace = "sed-me"' argo-cd/cr.yaml
	@$(YQ) -i 'select(.kind == "RoleBinding").subjects[0].namespace = "sed-me"' argo-cd/rbac.yaml

	# Prepare resources for templating with helm
	@$(YQ) -i '.spec = ["sed-me"]' argo-cd/cr.yaml
	@$(YQ) -i '.metadata.annotations.sed-deployment-annotations-top = "sed-me" \
	| .metadata.labels.sed-deployment-labels-top = "sed-me" \
	| .spec.template.metadata.annotations.sed-deployment-annotations-bottom = "sed-me" \
	| .spec.template.metadata.labels.sed-deployment-labels-bottom = "sed-me"' argo-cd/deployment.yaml

	# Add extra fields, for example argo-cd sync waves
	@$(YQ) -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "-1"' argo-cd/crd.yaml
	@$(YQ) -i '.metadata.annotations."argocd.argoproj.io/sync-wave" = "1"' argo-cd/deployment.yaml

	# Replace all namespaces to template them with helm
	@sed -i '' "s/namespace: [^ ]*/namespace: {{ .Values.namespace }}/g" argo-cd/cluster-rbac.yaml
	@sed -i '' "s/namespace: [^ ]*/namespace: {{ .Values.namespace }}/g" argo-cd/cr.yaml
	@sed -i '' "s/namespace: [^ ]*/namespace: {{ .Values.namespace }}/g" argo-cd/deployment.yaml
	@sed -i '' "s/namespace: [^ ]*/namespace: {{ .Values.namespace }}/g" argo-cd/rbac.yaml
	@sed -i '' "s/namespace: [^ ]*/namespace: {{ .Values.namespace }}/g" argo-cd/serviceaccounts.yaml

	# Replace extra fields (in addition to the namespaces) to template them with helm
	@sed -i '' "s/- sed-me/{{- toYaml .Values.spec | nindent 2 }}/g" argo-cd/cr.yaml
	@sed -i '' "s/sed-deployment-annotations-top: sed-me/{{- if ((.Values.operator).annotations) }}\n      {{- toYaml .Values.operator.annotations | nindent 4 -}}\n    {{ end }}/g" argo-cd/deployment.yaml
	@sed -i '' "s/sed-deployment-labels-top: sed-me/{{- if ((.Values.operator).labels) }}\n      {{- toYaml .Values.operator.labels | nindent 4 -}}\n    {{ end }}/g" argo-cd/deployment.yaml
	@sed -i '' "s/sed-deployment-annotations-bottom: sed-me/{{- if ((.Values.operator).annotations) }}\n          {{- toYaml .Values.operator.annotations | nindent 4 -}}\n        {{ end }}/g" argo-cd/deployment.yaml
	@sed -i '' "s/sed-deployment-labels-bottom: sed-me/{{- if ((.Values.operator).labels) }}\n          {{- toYaml .Values.operator.labels | nindent 4 -}}\n        {{ end }}/g" argo-cd/deployment.yaml

	@rm argo-cd/tmp.yaml