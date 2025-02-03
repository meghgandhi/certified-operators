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

FROM docker-na-public.artifactory.swg-devops.com/hyc-cloud-private-edge-docker-local/build-images/ubi9-minimal:latest

WORKDIR /root/ibm-license-service-scanner-operator

ARG TARGETOS
ARG TARGETARCH
ARG IMAGE_NAME
ARG IMAGE_DISPLAY_NAME
ARG IMAGE_NAME_ARCH
ARG IMAGE_MAINTAINER
ARG IMAGE_VENDOR
ARG IMAGE_VERSION
ARG VERSION
ARG IMAGE_RELEASE
ARG IMAGE_BUILDDATE
ARG IMAGE_DESCRIPTION
ARG IMAGE_SUMMARY
ARG IMAGE_OPENSHIFT_TAGS
ARG VCS_REF
ARG VCS_URL

LABEL org.label-schema.vendor="$IMAGE_VENDOR" \
      org.label-schema.name="$IMAGE_NAME_ARCH" \
      org.label-schema.description="$IMAGE_DESCRIPTION" \
      org.label-schema.vcs-ref=$VCS_REF \
      org.label-schema.vcs-url=$VCS_URL \
      org.label-schema.license="Licensed Materials - Property of IBM" \
      org.label-schema.schema-version="1.0" \
      name="$IMAGE_NAME" \
      maintainer="$IMAGE_MAINTAINER" \
      vendor="$IMAGE_VENDOR" \
      image-version="$IMAGE_VERSION" \
      version="$VERSION" \
      release="$IMAGE_RELEASE" \
      build-date="$IMAGE_BUILDDATE" \
      description="$IMAGE_DESCRIPTION" \
      summary="$IMAGE_SUMMARY" \
      io.k8s.display-name="$IMAGE_DISPLAY_NAME" \
      io.k8s.description="$IMAGE_DESCRIPTION" \
      io.openshift.tags="$IMAGE_OPENSHIFT_TAGS"

ENV OPERATOR=/usr/local/bin/ibm-license-service-scanner-operator \
    DEPLOY_DIR=/deploy \
    USER_UID=1001 \
    USER_NAME=ibm-license-service-scanner-operator \
    IMAGE_RELEASE="$IMAGE_RELEASE" \
    IMAGE_BUILDDATE="$IMAGE_BUILDDATE" \
    VAULT_CONNECTOR_SCRIPT_PATH=/opt/scanner/vault-connector.sh

# The command to run the operator is now ibm-license-service-scanner-operator -> must be configured in config/manager
COPY bin/manager-${TARGETOS}-${TARGETARCH} ${OPERATOR}

# Copy licenses
RUN mkdir /licenses
COPY LICENSE /licenses

# From https://docs.openshift.com/container-platform/4.14/openshift_images/create-images.html#images-create-guide-openshift_create-images
# TODO: Check if needed
RUN mkdir -p ${HOME} && chgrp -R 0 ${HOME} && chmod -R g=u ${HOME}

RUN mkdir -p /opt/scanner

# Add commit image release
RUN  echo -n "$IMAGE_RELEASE" > /opt/scanner/IMAGE_RELEASE \
  && echo -n "$IMAGE_BUILDDATE" > /opt/scanner/IMAGE_BUILDDATE

COPY scripts/vault-connector.sh /opt/scanner/vault-connector.sh

# Run the operator
ENTRYPOINT "exec ${OPERATOR} $@"

USER ${USER_UID}
