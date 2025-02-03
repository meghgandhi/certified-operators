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

SHELLCHECK_VERSION=$1

set -e

if shellcheck -V &> /dev/null; then
  version=v$( shellcheck -V | grep "version:" | cut -d " " -f2)
  if [ "$version" != "$SHELLCHECK_VERSION" ]; then
      echo "WARNING! Version mismatch: expected $SHELLCHECK_VERSION, got $version"
  fi
else
  echo "Please install shellcheck from https://github.com/koalaman/shellcheck"
  exit 1
fi
