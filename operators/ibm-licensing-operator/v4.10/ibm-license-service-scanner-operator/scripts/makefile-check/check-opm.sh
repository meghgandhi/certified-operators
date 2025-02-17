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

set -e

# Check if opm command works (in case e.g. installed with go already), if so, always exit with an optional warning
if opm version &> /dev/null; then
    version=$(opm version | awk '{print $2}' | cut -d ":" -f2 | tr -d ',"')
    if [ "$version" = "$OPM_VERSION" ]; then
        exit 0
    else
        echo "WARNING! Version mismatch: expected $OPM_VERSION, got $version"
        exit 0
    fi
fi

# OPM command didn't work, so check file exists
test -s "$OPM"

# Warn about version mismatch of local files
version=$("$OPM" version | awk '{print $2}' | cut -d ":" -f2 | tr -d ',"')
if [ "$version" != "$OPM_VERSION" ]; then
    echo "WARNING! Version mismatch: expected $OPM_VERSION, got $version"
fi
