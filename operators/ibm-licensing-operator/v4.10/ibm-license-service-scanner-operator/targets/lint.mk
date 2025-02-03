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

.PHONY: lint/scripts
lint/scripts: deps/require-shellcheck
	@for file in $(shell find scripts -name "*.sh"); do shellcheck -x --format=gcc $$file; done;

.PHONY: lint/go
lint/go: deps/require-golangci-lint deps/require-lll
	@$(LOCAL_BIN_DIR)/golangci-lint run ./...
	@$(LOCAL_BIN_DIR)/lll -l 120 -g -e "\+kubebuilder|protobuf" . | tee /dev/stderr | if [[ "$$(wc -c)" -ne 0 ]]; then exit 1; fi

.PHONY: lint/yaml
lint/yaml: deps/require-yamllint
	@$(LOCAL_BIN_DIR)/.venv/bin/yamllint -c ./.yamllinter-config.yaml .

# All linters
.PHONY: lint/all
lint/all: fmt vet lint/go lint/scripts lint/yaml
	@echo "Linting finished successfully"
