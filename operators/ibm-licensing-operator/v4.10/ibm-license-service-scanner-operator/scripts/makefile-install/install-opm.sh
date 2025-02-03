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

OPM=$1
OPM_VERSION=$2
LOCAL_OS=$3
LOCAL_ARCH=$4

set -ex

# Download the binary file
curl -s -L -f "https://github.com/operator-framework/operator-registry/releases/download/$OPM_VERSION/${LOCAL_OS}-$LOCAL_ARCH-opm" > "$OPM"
chmod a+x "$OPM"
