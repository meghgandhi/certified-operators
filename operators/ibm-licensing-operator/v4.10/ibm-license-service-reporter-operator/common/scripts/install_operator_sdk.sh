#!/bin/bash
#
# Licensed Materials - Property of IBM
# (C) Copyright IBM Corporation 2023. All Rights Reserved
# US Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP Schedule Contract with IBM Corp.
#

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
ARCH=$2
export OS=$(uname | awk '{print tolower($0)}')
export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.27.0
echo ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
curlf -L ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH} 1> ${LOCALBIN}/operator-sdk
res=$?
if [ $res != 0 ]; then
  rm ${LOCALBIN}/operator-sdk
  exit $res
fi
chmod +x ${LOCALBIN}/operator-sdk