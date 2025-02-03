# Copyright 2024 IBM Corporation
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

# TODO: Move some variables to the build.mk file, consider splitting into more files, rearrange some targets, etc.

include targets/*.mk

## General

# Local bin folder used e.g. for storing dependencies' binaries
LOCAL_BIN_DIR ?= $(abspath $(dir $(firstword $(MAKEFILE_LIST))))/bin

# Must be created if doesn't exist, as some targets place dependencies into it
.PHONY: require-local-bin-dir
require-local-bin-dir:
	mkdir -p $(LOCAL_BIN_DIR)

# Local scripts folder used e.g. to store dependencies' installation scripts
LOCAL_SCRIPTS_DIR ?= ./scripts

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

# Setting SHELL to bash allows bash commands to be executed by recipes.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

## Docker setup

# Prod/dev registries
REGISTRY_BASE ?= docker-na-public.artifactory.swg-devops.com
REGISTRY_INTEGRATION ?= ${REGISTRY_BASE}/hyc-cloud-private-integration-docker-local/ibmcom
REGISTRY_EDGE ?= ${REGISTRY_BASE}/hyc-cloud-private-edge-docker-local/ibmcom
REGISTRY_SCRATCH ?= ${REGISTRY_BASE}/hyc-cloud-private-scratch-docker-local/ibmcom

# Docker images
OPERATOR_IMAGE ?= $(REGISTRY_INTEGRATION)/ibm-license-service-scanner-operator
OPERATOR_IMAGE_DEV ?= $(REGISTRY_SCRATCH)/ibm-license-service-scanner-operator
OPERATOR_IMAGE_ICR ?= icr.io/cpopen/ibm-license-service-scanner-operator

# Images-related tags and annotations
CSV_VERSION ?= $(shell cat ./version/version.go | grep "Version =" | awk '{ print $$3}' | tr -d '"')
COMMIT_VERSION ?= $(shell git describe --exact-match 2> /dev/null || git describe --match=$(git rev-parse --short=8 HEAD) --always --abbrev=8)
IMAGE_TAG ?= $(COMMIT_VERSION)

# Git branch is used as a tag in dev images
GIT_BRANCH=$(shell git rev-parse --abbrev-ref HEAD)
ifeq ($(GIT_BRANCH),HEAD)
	GIT_BRANCH=$(TRAVIS_BRANCH)
endif


GIT_BRANCH_ORIG := $(GIT_BRANCH)

# Renovate generates branch names with special characters (e.g., "/").
# This replaces all special characters other than '-' and '.' with '-' to ensure the valid Docker tag is used.
GIT_BRANCH:=$(shell echo $(GIT_BRANCH_ORIG) | sed 's/[^a-zA-Z0-9.-]/-/g')

echo "Sanitized branch name that will be used as image tag: ${GIT_BRANCH}"

# Variables for redhat ubi certification required labels
IMAGE_NAME=$(OPERATOR_IMAGE):$(IMAGE_TAG)
IMAGE_NAME_DEV=$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)
IMAGE_DISPLAY_NAME=IBM License Service Scanner Operator
IMAGE_MAINTAINER=talk2sam@us.ibm.com
IMAGE_VENDOR=IBM
IMAGE_DESCRIPTION=Operator used to install IBM License Service Scanner.
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

# Passed to docker buildx commands
$(eval DOCKER_BUILD_OPTS := --build-arg "IMAGE_NAME=$(IMAGE_NAME)" --build-arg "IMAGE_DISPLAY_NAME=$(IMAGE_DISPLAY_NAME)" --build-arg "IMAGE_MAINTAINER=$(IMAGE_MAINTAINER)" --build-arg "IMAGE_VENDOR=$(IMAGE_VENDOR)" --build-arg "IMAGE_VERSION=$(IMAGE_VERSION)" --build-arg "VERSION=$(CSV_VERSION)" --build-arg "IMAGE_RELEASE=$(IMAGE_RELEASE)"  --build-arg "IMAGE_BUILDDATE=$(IMAGE_BUILDDATE)" --build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" --build-arg "IMAGE_SUMMARY=$(IMAGE_SUMMARY)" --build-arg "IMAGE_OPENSHIFT_TAGS=$(IMAGE_OPENSHIFT_TAGS)" --build-arg "VCS_REF=$(VCS_REF)" --build-arg "VCS_URL=$(GIT_REMOTE_URL)" --build-arg "IMAGE_NAME_ARCH=$(IMAGE_NAME)-$(LOCAL_ARCH)")
$(eval DOCKER_BUILD_OPTS_DEV := --build-arg "IMAGE_NAME=$(IMAGE_NAME_DEV)" --build-arg "IMAGE_DISPLAY_NAME=$(IMAGE_DISPLAY_NAME)" --build-arg "IMAGE_MAINTAINER=$(IMAGE_MAINTAINER)" --build-arg "IMAGE_VENDOR=$(IMAGE_VENDOR)" --build-arg "IMAGE_VERSION=$(IMAGE_VERSION)" --build-arg "VERSION=$(CSV_VERSION)" --build-arg "IMAGE_RELEASE=$(IMAGE_RELEASE)"  --build-arg "IMAGE_BUILDDATE=$(IMAGE_BUILDDATE)" --build-arg "IMAGE_DESCRIPTION=$(IMAGE_DESCRIPTION)" --build-arg "IMAGE_SUMMARY=$(IMAGE_SUMMARY)" --build-arg "IMAGE_OPENSHIFT_TAGS=$(IMAGE_OPENSHIFT_TAGS)" --build-arg "VCS_REF=$(VCS_REF)" --build-arg "VCS_URL=$(GIT_REMOTE_URL)" --build-arg "IMAGE_NAME_ARCH=$(IMAGE_NAME_DEV)-$(LOCAL_ARCH)")

## Bundle

CHANNELS ?= v4.2
DEFAULT_CHANNEL ?= v4.2

BUNDLE_IMAGE ?= $(OPERATOR_IMAGE)-bundle
BUNDLE_IMAGE_DEV ?= $(OPERATOR_IMAGE_DEV)-bundle
BUNDLE_IMAGES ?= $(BUNDLE_IMAGE):$(IMAGE_TAG)
BUNDLE_IMAGES_DEV ?= $(BUNDLE_IMAGE_DEV):$(IMAGE_TAG)

ifneq ($(origin CHANNELS), undefined)
BUNDLE_CHANNELS := --channels=$(CHANNELS)
endif

ifneq ($(origin DEFAULT_CHANNEL), undefined)
BUNDLE_DEFAULT_CHANNEL := --default-channel=$(DEFAULT_CHANNEL)
endif
BUNDLE_METADATA_OPTS ?= $(BUNDLE_CHANNELS) $(BUNDLE_DEFAULT_CHANNEL)

BUNDLE_GEN_FLAGS ?= -q --overwrite --extra-service-accounts ibm-licensing-scanner --version $(CSV_VERSION) $(BUNDLE_METADATA_OPTS)

USE_IMAGE_DIGESTS ?= false
ifeq ($(USE_IMAGE_DIGESTS), true)
	BUNDLE_GEN_FLAGS += --use-image-digests
endif

.PHONY: bundle
bundle: deps/require-operator-sdk deps/require-kustomize manifests
	$(OPERATOR_SDK) generate kustomize manifests -q
	$(YQ) -i '.spec.relatedImages[0].image = "$(OPERATOR_IMAGE_ICR):$(CSV_VERSION)"' ./config/manifests/bases/ibm-license-service-scanner-operator.clusterserviceversion.yaml
	pushd config/manager && $(KUSTOMIZE) edit set image controller=$(OPERATOR_IMAGE_ICR):$(CSV_VERSION) && popd
	$(KUSTOMIZE) build config/manifests | $(OPERATOR_SDK) generate bundle $(BUNDLE_GEN_FLAGS)
	$(YQ) eval-all 'select(fileIndex==0).spec.relatedImages = select(fileIndex==1).spec.relatedImages | select(fileIndex==0)' ./bundle/manifests/ibm-license-service-scanner-operator.clusterserviceversion.yaml ./config/manifests/bases/ibm-license-service-scanner-operator.clusterserviceversion.yaml -i
	$(OPERATOR_SDK) bundle validate ./bundle

## Catalog
CATALOG_IMG ?= $(OPERATOR_IMAGE)-catalog
CATALOG_IMG_DEV ?= $(OPERATOR_IMAGE_DEV)-catalog

# Set CATALOG_BASE_IMG to an existing catalog image tag to add $BUNDLE_IMAGES to that image.
ifneq ($(origin CATALOG_BASE_IMG), undefined)
FROM_INDEX_OPT := --from-index $(CATALOG_BASE_IMG)
endif

ifneq ($(origin CATALOG_BASE_IMG_DEV), undefined)
FROM_INDEX_OPT_DEV := --from-index $(CATALOG_BASE_IMG_DEV)
endif

## Controller-gen related

.PHONY: manifests
manifests: deps/require-yq deps/require-controller-gen
	$(YQ) -i '.metadata.annotations.containerImage = "$(OPERATOR_IMAGE_ICR):$(CSV_VERSION)"' ./config/manifests/bases/ibm-license-service-scanner-operator.clusterserviceversion.yaml
	$(CONTROLLER_GEN) rbac:roleName=operator-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	# For clarity and to distinguish between operator and operand resources, rename role.yaml to role_operator.yaml
	mv ./config/rbac/role.yaml ./config/rbac/role_operator.yaml

.PHONY: generate
generate: deps/require-controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

## Common targets TODO: Rename

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

# Run the operator
OPERATOR_NAMESPACE ?= ibm-licensing-scanner
VAULT_CONNECTOR_SCRIPT_PATH ?= scripts/vault-connector.sh

# Operand latest dev image
IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE ?= $(REGISTRY_INTEGRATION)/ibm-licensing-scanner:latest

.PHONY: run
run:
	OPERATOR_NAMESPACE=$(OPERATOR_NAMESPACE) \
	IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE=$(IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE) \
	VAULT_CONNECTOR_SCRIPT_PATH=$(VAULT_CONNECTOR_SCRIPT_PATH) \
	go run ./cmd/main.go

# Replace existing version (CURRENT_VERSION var) with the provided value (NEW_VERSION var)
# Example: make update-version CURRENT_VERSION=0.0.0 NEW_VERSION=0.0.1
.PHONY: update-version
update-version:
	# version.go
	sed -i '' "s/$(CURRENT_VERSION)/$(NEW_VERSION)/g" ./version/version.go

	# config/manifests/bases
	sed -i '' "s/$(CURRENT_VERSION)/$(NEW_VERSION)/g" ./config/manifests/bases/ibm-license-service-scanner-operator.clusterserviceversion.yaml

	# config/manager
	sed -i '' "s/$(CURRENT_VERSION)/$(NEW_VERSION)/g" ./config/manager/kustomization.yaml
	sed -i '' "s/$(CURRENT_VERSION)/$(NEW_VERSION)/g" ./config/manager/manager.yaml

	# bundle
	$(MAKE) bundle

# If needed, call below targets with `TO_FILE="> output.yaml"` to direct the generated data into an output file
# For example, `make generate-yaml-no-olm TO_FILE="> output.yaml"`

.PHONY: generate-yaml-no-olm
generate-yaml-no-olm: deps/require-yq deps/require-kustomize
	@$(KUSTOMIZE) build config/manifests \
	| $(YQ) 'select(.kind != "Configuration" and .kind != "ClusterServiceVersion" and .kind != "IBMLicenseServiceScanner")' $(TO_FILE)

.PHONY: generate-yaml-no-olm-dev
generate-yaml-no-olm-dev: deps/require-yq deps/require-kustomize
	@$(KUSTOMIZE) build config/manifests \
	| $(YQ) 'select(.kind != "Configuration" and .kind != "ClusterServiceVersion" and .kind != "IBMLicenseServiceScanner")' \
	| $(YQ) '(select(.kind == "Deployment") | .spec.template.spec.containers[] | select(.name == "ibm-licensing-scanner-operator") | .image) = "$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)"' \
	| $(YQ) '(select(.kind == "Deployment") | .spec.template.spec.containers[] | select(.name == "ibm-licensing-scanner-operator") | .env[] | select(.name == "IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE") | .value) = "$(IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE)"' $(TO_FILE)

.PHONY: generate-yaml-argo-cd
generate-yaml-argo-cd: deps/require-yq deps/require-kustomize
	@mkdir -p argo-cd && $(KUSTOMIZE) build config/manifests > argo-cd/tmp.yaml

	# Split the resources into separate YAML files
	@(echo "---" && $(YQ) 'select(.kind == "ClusterRole" or .kind == "ClusterRoleBinding")' argo-cd/tmp.yaml) > argo-cd/cluster-rbac.yaml
	@(echo "---" && $(YQ) 'select(.kind == "IBMLicenseServiceScanner")' argo-cd/tmp.yaml) > argo-cd/cr.yaml
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