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

LOCAL_BIN_DIR=$1
YAMLLINT_VERSION=$2

set -ex

# Create Python virtual environment to install dependencies (prefer python3)
if python3 --version &> /dev/null; then
    python3 -m venv "$LOCAL_BIN_DIR"/.venv
else
    python -m venv "$LOCAL_BIN_DIR"/.venv
fi

# Switch to new Python virtual environment
# shellcheck disable=SC1091
source "$LOCAL_BIN_DIR"/.venv/bin/activate

# Install dependency
pip install yamllint=="$YAMLLINT_VERSION"