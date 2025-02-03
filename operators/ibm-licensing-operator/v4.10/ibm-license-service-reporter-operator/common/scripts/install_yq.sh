#!/bin/bash
#
# Licensed Materials - Property of IBM
# (C) Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#

echo ">>> Installing Yq"

curlf() {
  OUTPUT_FILE=$(mktemp)
  HTTP_CODE=$(curl --silent --output $OUTPUT_FILE --write-out "%{http_code}" "$@")
  if [[ ${HTTP_CODE} -lt 200 || ${HTTP_CODE} -gt 299 ]] ; then
    >&2 cat $OUTPUT_FILE
    >&2 echo
    return 22
  fi
  cat $OUTPUT_FILE
  rm $OUTPUT_FILE
}

LOCALBIN=$1
LOCAL_ARCH=$2
LOCAL_OS=$3
YQ_VERSION=$4
# Download binary
echo https://github.com/mikefarah/yq/releases/download/"${YQ_VERSION}"/yq_"${LOCAL_OS}"_"${LOCAL_ARCH}"
curlf -L https://github.com/mikefarah/yq/releases/download/"${YQ_VERSION}"/yq_"${LOCAL_OS}"_"${LOCAL_ARCH}" 1> ${LOCALBIN}/yq
res=$?
if [ $res != 0 ]; then
  rm ${LOCALBIN}/yq
  exit $res
fi
chmod +x ${LOCALBIN}/yq