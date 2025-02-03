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
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFindOAuthParamValue(t *testing.T) {
	oauth := OAuth{
		Parameters: []string{
			"--param2=jjjjj",
			"--param1=ppppp",
			"--param3=22222",
			"--param7=ggggg",
		},
	}

	val, found := oauth.FindOAuthParamValue("--param3")
	assert.True(t, found)
	assert.Equal(t, "22222", val)
}

func TestFindOAuthParamValueEmptyParams(t *testing.T) {
	oauth := OAuth{
		Parameters: []string{},
	}

	_, found := oauth.FindOAuthParamValue("--param3")
	assert.False(t, found)
}

func TestFindOAuthParamValueNoSuchParam(t *testing.T) {
	oauth := OAuth{
		Parameters: []string{
			"--param2=jjjjj",
			"--param1=ppppp",
			"--param3=22222",
			"--param7=ggggg",
		},
	}

	_, found := oauth.FindOAuthParamValue("--param4")
	assert.False(t, found)
}

func TestFindOAuthParamValueNoSuchParam2(t *testing.T) {
	oauth := OAuth{
		Parameters: []string{
			"param2=jjjjj",
			"param1=ppppp",
			"param3=22222",
			"param7=ggggg",
		},
	}

	_, found := oauth.FindOAuthParamValue("--param3")
	assert.False(t, found)
}

// TODO params in wrong format?
