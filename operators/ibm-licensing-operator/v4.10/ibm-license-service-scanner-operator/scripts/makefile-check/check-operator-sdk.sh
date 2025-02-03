#!/usr/bin/env bash

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

OPERATOR_SDK=$1
OPERATOR_SDK_VERSION=$2

set -e

# Check file exists
test -s "$OPERATOR_SDK"

# Warn about version mismatch
version=$("$OPERATOR_SDK" version | awk '{print $3}' | tr -d '",')
if [ "$version" != "$OPERATOR_SDK_VERSION" ]; then
    echo "WARNING! Version mismatch: expected $OPERATOR_SDK_VERSION, got $version"
fi
