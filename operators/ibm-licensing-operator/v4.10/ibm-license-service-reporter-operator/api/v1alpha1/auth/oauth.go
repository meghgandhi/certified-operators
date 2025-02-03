//
// Copyright 2023 IBM Corporation
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
//

package auth

import (
	"strings"
)

type OAuth struct {
	Enabled    bool     `json:"enabled,omitempty"`
	Parameters []string `json:"parameters,omitempty"`
}

// Returns true and value of the expected parameter or false and empty string if it was not found
func (oauth *OAuth) FindOAuthParamValue(expectedParam string) (string, bool) {
	if oauth.Parameters == nil || len(oauth.Parameters) == 0 {
		return "", false
	}

	for _, param := range oauth.Parameters {
		splitParam := strings.Split(param, "=")
		if splitParam[0] == expectedParam {
			if len(splitParam) != 2 {
				return "", false
			}
			return splitParam[1], true
		}
	}

	return "", false
}
