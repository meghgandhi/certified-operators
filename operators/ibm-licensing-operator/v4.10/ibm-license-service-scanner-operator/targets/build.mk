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

## Docker

.PHONY: build/docker-login
build/docker-login:
	docker login -u $(ARTIFACTORY_LOGIN) -p $(ARTIFACTORY_TOKEN) $(REGISTRY_BASE)

.PHONY: build/docker-push
build/docker-push:
	docker push $(IMG)

.PHONY: build/docker-manifest
build/docker-manifest:
	docker manifest create $(OPERATOR_IMAGE):$(CSV_VERSION) $(OPERATOR_IMAGE):$(IMAGE_TAG)
	docker manifest push $(OPERATOR_IMAGE):$(CSV_VERSION)

.PHONY: build/docker-manifest-dev
build/docker-manifest-dev:
	docker manifest create $(OPERATOR_IMAGE_DEV):$(GIT_BRANCH) $(OPERATOR_IMAGE_DEV):$(IMAGE_TAG)
	docker manifest push $(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)

PLATFORMS ?= linux/amd64
.PHONY: build/buildx
build/buildx: deps/require-docker-buildx
	- docker buildx create --name project-v3-builder
	- docker buildx use project-v3-builder
	docker buildx build $(DOCKER_BUILD_OPTS) --push --platform $(PLATFORMS) --tag $(OPERATOR_IMAGE):$(IMAGE_TAG) --tag $(OPERATOR_IMAGE):$(CSV_VERSION) --progress plain --provenance=false .
	- docker buildx rm project-v3-builder

.PHONY: build/buildx-dev
build/buildx-dev: deps/require-docker-buildx
	- docker buildx create --name project-v3-builder
	- docker buildx use project-v3-builder
	docker buildx build $(DOCKER_BUILD_OPTS_DEV) --push --platform $(PLATFORMS) --tag $(OPERATOR_IMAGE_DEV):$(IMAGE_TAG) --tag $(OPERATOR_IMAGE_DEV):$(GIT_BRANCH) --progress plain --provenance=false .
	- docker buildx rm project-v3-builder

## Binaries

.PHONY: build/bin-amd64
build/bin-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -o ./bin/manager-linux-amd64 ./cmd/main.go

## Bundle

.PHONY: build/bundle
build/bundle:
	docker build -f ./bundle.Dockerfile -t $(BUNDLE_IMAGE):$(IMAGE_TAG) -t $(BUNDLE_IMAGE):$(CSV_VERSION) .

# TODO: Automate YQ assignment, also copy bases/manifest one to bundle/manifest
.PHONY: build/bundle-dev
build/bundle-dev: deps/require-yq
	$(YQ) -i '(.spec.install.spec.deployments[0].spec.template.spec.containers[] | select(.name == "ibm-licensing-scanner-operator") | .image) = "$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)"' ./bundle/manifests/ibm-license-service-scanner-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.relatedImages[0].image = "$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)"' ./config/manifests/bases/ibm-license-service-scanner-operator.clusterserviceversion.yaml
	$(YQ) -i '.spec.relatedImages[0].image = "$(OPERATOR_IMAGE_DEV):$(GIT_BRANCH)"' ./bundle/manifests/ibm-license-service-scanner-operator.clusterserviceversion.yaml
	docker build -f ./bundle.Dockerfile -t $(BUNDLE_IMAGE_DEV):$(IMAGE_TAG) -t $(BUNDLE_IMAGE_DEV):$(GIT_BRANCH) .

.PHONY: build/bundle-push
build/bundle-push:
	$(MAKE) build/docker-push IMG=$(BUNDLE_IMAGE):$(IMAGE_TAG)
	$(MAKE) build/docker-push IMG=$(BUNDLE_IMAGE):$(CSV_VERSION)

.PHONY: build/bundle-push-dev
build/bundle-push-dev:
	$(MAKE) build/docker-push IMG=$(BUNDLE_IMAGE_DEV):$(IMAGE_TAG)
	$(MAKE) build/docker-push IMG=$(BUNDLE_IMAGE_DEV):$(GIT_BRANCH)

## Catalog

.PHONY: build/catalog
build/catalog: deps/require-opm
	$(OPM) index add --container-tool docker --tag $(CATALOG_IMG):$(IMAGE_TAG) --bundles $(BUNDLE_IMAGES) $(FROM_INDEX_OPT)
	docker tag $(CATALOG_IMG):$(IMAGE_TAG) $(CATALOG_IMG):$(CSV_VERSION)

.PHONY: build/catalog-dev
build/catalog-dev: deps/require-opm
	$(OPM) index add --container-tool docker --tag $(CATALOG_IMG_DEV):$(IMAGE_TAG) --bundles $(BUNDLE_IMAGES_DEV) $(FROM_INDEX_OPT_DEV)
	docker tag $(CATALOG_IMG_DEV):$(IMAGE_TAG) $(CATALOG_IMG_DEV):$(GIT_BRANCH)

.PHONY: build/catalog-push
build/catalog-push:
	$(MAKE) build/docker-push IMG=$(CATALOG_IMG):$(IMAGE_TAG)
	$(MAKE) build/docker-push IMG=$(CATALOG_IMG):$(CSV_VERSION)

.PHONY: build/catalog-push-dev
build/catalog-push-dev:
	$(MAKE) build/docker-push IMG=$(CATALOG_IMG_DEV):$(IMAGE_TAG)
	$(MAKE) build/docker-push IMG=$(CATALOG_IMG_DEV):$(GIT_BRANCH)

## Complete build-push process

.PHONY: build/all
build/all: build/bin-amd64 build/buildx build/bundle build/bundle-push build/catalog build/catalog-push build/docker-manifest
	@echo "All build operations successful"

.PHONY: build/all-dev
build/all-dev: build/bin-amd64 build/buildx-dev build/bundle-dev build/bundle-push-dev build/catalog-dev build/catalog-push-dev build/docker-manifest-dev
	@echo "All (dev) build operations successful"
