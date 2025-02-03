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

# Dependency binaries
DOCKER_BUILDX ?= $(LOCAL_BIN_DIR)/docker-buildx
OPERATOR_SDK ?= $(LOCAL_BIN_DIR)/operator-sdk
KUSTOMIZE ?= $(LOCAL_BIN_DIR)/kustomize
CONTROLLER_GEN ?= $(LOCAL_BIN_DIR)/controller-gen
YQ ?= $(LOCAL_BIN_DIR)/yq
OPM ?= $(LOCAL_BIN_DIR)/opm
GOLANGCI_LINT ?= $(LOCAL_BIN_DIR)/golangci-lint
YAMLLINT ?= $(LOCAL_BIN_DIR)/.venv/bin/yamllint
SHELLCHECK ?= $(LOCAL_BIN_DIR)/shellcheck
LLL ?= $(LOCAL_BIN_DIR)/lll

# Dependency versions
DOCKER_BUILDX_VERSION ?= v0.12.1
OPERATOR_SDK_VERSION ?= v1.34.1
KUSTOMIZE_VERSION ?= v5.3.0
CONTROLLER_GEN_VERSION ?= v0.14.0
YQ_VERSION ?= v4.40.5
OPM_VERSION ?= v1.32.0
GOLANGCI_LINT_VERSION ?= v1.61.0
YAMLLINT_VERSION ?= 1.32.0
SHELLCHECK_VERSION ?= v0.10.0

# Dependency check and install scripts, for ease of use
LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR ?= $(LOCAL_SCRIPTS_DIR)/makefile-check
LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR ?= $(LOCAL_SCRIPTS_DIR)/makefile-install

## Docker Buildx

# Dir must exist for plugin installation (dir will not be created if buildx command already works and passes the check)
DOCKER_CLI_PLUGINS ?= ~/.docker/cli-plugins
.PHONY: deps/require-cli-plugins-dir
deps/require-cli-plugins-dir:
	mkdir -p $(DOCKER_CLI_PLUGINS)

.PHONY: deps/require-docker-buildx
deps/require-docker-buildx:
	@$(MAKE) deps/check-docker-buildx || $(MAKE) deps/install-docker-buildx

.PHONY: deps/check-docker-buildx
deps/check-docker-buildx: require-local-bin-dir
	@echo "Checking dependency: docker buildx"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-docker-buildx.sh $(DOCKER_BUILDX) $(DOCKER_BUILDX_VERSION) $(DOCKER_CLI_PLUGINS)
	@echo "Dependency satisfied: docker buildx"

.PHONY: deps/install-docker-buildx
deps/install-docker-buildx: require-local-bin-dir deps/require-cli-plugins-dir
	@echo "Installing dependency: docker buildx"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-docker-buildx.sh $(DOCKER_BUILDX) $(DOCKER_BUILDX_VERSION) $(DOCKER_CLI_PLUGINS) $(LOCAL_OS) $(LOCAL_ARCH)
	@echo "Dependency installed: docker buildx"
	@echo "Checking installation successful: docker buildx"
	@$(MAKE) deps/check-docker-buildx

## Operator SDK

.PHONY: deps/require-operator-sdk
deps/require-operator-sdk:
	@$(MAKE) deps/check-operator-sdk || $(MAKE) deps/install-operator-sdk

.PHONY: deps/check-operator-sdk
deps/check-operator-sdk: require-local-bin-dir
	@echo "Checking dependency: operator-sdk"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-operator-sdk.sh $(OPERATOR_SDK) $(OPERATOR_SDK_VERSION)
	@echo "Dependency satisfied: operator-sdk"

.PHONY: deps/install-operator-sdk
deps/install-operator-sdk: require-local-bin-dir
	@echo "Installing dependency: operator-sdk"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-operator-sdk.sh $(OPERATOR_SDK) $(OPERATOR_SDK_VERSION) $(LOCAL_OS) $(LOCAL_ARCH)
	@echo "Dependency installed: operator-sdk"
	@echo "Checking installation successful: operator-sdk"
	@$(MAKE) deps/check-operator-sdk

## Kustomize

.PHONY: deps/require-kustomize
deps/require-kustomize:
	@$(MAKE) deps/check-kustomize || $(MAKE) deps/install-kustomize

.PHONY: deps/check-kustomize
deps/check-kustomize: require-local-bin-dir
	@echo "Checking dependency: kustomize"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-kustomize.sh $(KUSTOMIZE) $(KUSTOMIZE_VERSION)
	@echo "Dependency satisfied: kustomize"

.PHONY: deps/install-kustomize
deps/install-kustomize: require-local-bin-dir
	@echo "Installing dependency: kustomize"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-kustomize.sh $(KUSTOMIZE_VERSION) $(LOCAL_BIN_DIR)
	@echo "Dependency installed: kustomize"
	@echo "Checking installation successful: kustomize"
	@$(MAKE) deps/check-kustomize

## Controller-gen

.PHONY: deps/require-controller-gen
deps/require-controller-gen:
	@$(MAKE) deps/check-controller-gen || $(MAKE) deps/install-controller-gen

.PHONY: deps/check-controller-gen
deps/check-controller-gen: require-local-bin-dir
	@echo "Checking dependency: controller-gen"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-controller-gen.sh $(CONTROLLER_GEN) $(CONTROLLER_GEN_VERSION)
	@echo "Dependency satisfied: controller-gen"

.PHONY: deps/install-controller-gen
deps/install-controller-gen: require-local-bin-dir
	@echo "Installing dependency: controller-gen"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-controller-gen.sh $(CONTROLLER_GEN_VERSION) $(LOCAL_BIN_DIR)
	@echo "Dependency installed: controller-gen"
	@echo "Checking installation successful: controller-gen"
	@$(MAKE) deps/check-controller-gen

## YQ

.PHONY: deps/require-yq
deps/require-yq:
	@$(MAKE) deps/check-yq || $(MAKE) deps/install-yq

.PHONY: deps/check-yq
deps/check-yq: require-local-bin-dir
	@echo "Checking dependency: yq"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-yq.sh $(YQ) $(YQ_VERSION)
	@echo "Dependency satisfied: yq"

.PHONY: deps/install-yq
deps/install-yq: require-local-bin-dir
	@echo "Installing dependency: yq"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-yq.sh $(YQ) $(YQ_VERSION) $(LOCAL_OS) $(LOCAL_ARCH)
	@echo "Dependency installed: yq"
	@echo "Checking installation successful: yq"
	@$(MAKE) deps/check-yq

## OPM

.PHONY: deps/require-opm
deps/require-opm:
	@$(MAKE) deps/check-opm || $(MAKE) deps/install-opm

.PHONY: deps/check-opm
deps/check-opm: require-local-bin-dir
	@echo "Checking dependency: opm"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-opm.sh $(OPM) $(OPM_VERSION)
	@echo "Dependency satisfied: opm"

.PHONY: deps/install-opm
deps/install-opm: require-local-bin-dir
	@echo "Installing dependency: opm"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-opm.sh $(OPM) $(OPM_VERSION) $(LOCAL_OS) $(LOCAL_ARCH)
	@echo "Dependency installed: opm"
	@echo "Checking installation successful: opm"
	@$(MAKE) deps/check-opm

## golangci-lint

.PHONY: deps/require-golangci-lint
deps/require-golangci-lint:
	@$(MAKE) deps/check-golangci-lint || $(MAKE) deps/install-golangci-lint

.PHONY: deps/check-golangci-lint
deps/check-golangci-lint: require-local-bin-dir
	@echo "Checking dependency: golangci-lint"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-golangci-lint.sh $(GOLANGCI_LINT) $(GOLANGCI_LINT_VERSION)
	@echo "Dependency satisfied: golangci-lint"

.PHONY: deps/install-golangci-lint
deps/install-golangci-lint: require-local-bin-dir
	@echo "Checking dependency: golangci-lint"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-golangci-lint.sh $(GOLANGCI_LINT_VERSION) $(LOCAL_BIN_DIR)
	@echo "Dependency installed: golangci-lint"
	@echo "Checking installation successful: golangci-lint"
	@$(MAKE) deps/check-golangci-lint

## yamllint

.PHONE: deps/require-yamllint
deps/require-yamllint:
	@$(MAKE) deps/check-yamllint || $(MAKE) deps/install-yamllint

.PHONY: deps/check-yamllint
deps/check-yamllint: require-local-bin-dir
	@echo "Checking dependency: yamllint"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-yamllint.sh $(YAMLLINT) $(YAMLLINT_VERSION)
	@echo "Dependency satisfied: yamllint"

.PHONY: deps/install-yamllint
deps/install-yamllint: require-local-bin-dir
	@echo "Installing dependency: yamllint"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-yamllint.sh $(LOCAL_BIN_DIR) $(YAMLLINT_VERSION)
	@echo "Dependency installed: yamllint"
	@echo "Checking installation successful: yamllint"
	@$(MAKE) deps/check-yamllint

## shellcheck

.PHONY: deps/require-shellcheck
deps/require-shellcheck:
	@$(MAKE) deps/check-shellcheck

.PHONY: deps/check-shellcheck
deps/check-shellcheck: require-local-bin-dir
	@echo "Checking dependency: shellcheck"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-shellcheck.sh $(SHELLCHECK_VERSION)
	@echo "Dependency satisfied: shellcheck"

## lll

.PHONY: deps/require-lll
deps/require-lll:
	@$(MAKE) deps/check-lll || $(MAKE) deps/install-lll

.PHONY: deps/check-lll
deps/check-lll: require-local-bin-dir
	@echo "Checking dependency: lll"
	@$(LOCAL_SCRIPTS_MAKEFILE_CHECK_DIR)/check-lll.sh $(LLL)
	@echo "Dependency satisfied: lll"

.PHONY: deps/install-lll
deps/install-lll: require-local-bin-dir
	@echo "Checking dependency: lll"
	@$(LOCAL_SCRIPTS_MAKEFILE_INSTALL_DIR)/install-lll.sh $(LOCAL_BIN_DIR)
	@echo "Dependency installed: lll"
	@echo "Checking installation successful: lll"
	@$(MAKE) deps/check-lll

# All dependencies

.PHONY: deps/all
deps/all: deps/require-docker-buildx deps/require-operator-sdk deps/require-kustomize deps/require-controller-gen deps/require-yq deps/require-opm deps/require-golangci-lint deps/require-yamllint deps/require-shellcheck deps/require-lll
	@echo "All dependencies satisfied"
