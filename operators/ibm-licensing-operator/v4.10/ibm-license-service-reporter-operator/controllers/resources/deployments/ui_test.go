// Copyright 2024 IBM Corporation
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
package deployments

import (
	"testing"

	"github.com/stretchr/testify/assert"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	res "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	corev1 "k8s.io/api/core/v1"
)

func TestGetUIEnvironmentVariablesDefault(t *testing.T) {
	spec := operatorv1alpha1.IBMLicenseServiceReporterSpec{}
	enableWorkloadsEnv := corev1.EnvVar{
		Name:  "ENABLE_WORKLOADS_PROCESSING",
		Value: "true",
	}
	envVars := getReporterUIEnvironmentVariables(spec)
	assert.False(t, res.Contains(envVars, enableWorkloadsEnv), "EnableWorkloadsProcessing is not set, 'ENABLE_WORKLOADS_PROCESSING' environemnt variable should not be added to UI Receiver pod.")
}
func TestGetUIEnvironmentVariablesWorkloadsEnabled(t *testing.T) {
	spec := operatorv1alpha1.IBMLicenseServiceReporterSpec{
		EnableWorkloadsProcessing: true,
	}
	enableWorkloadsEnv := corev1.EnvVar{
		Name:  "ENABLE_WORKLOADS_PROCESSING",
		Value: "true",
	}
	envVars := getReporterUIEnvironmentVariables(spec)
	assert.True(t, res.Contains(envVars, enableWorkloadsEnv), "EnableWorkloadsProcessing is set, 'ENABLE_WORKLOADS_PROCESSING' environemnt variable should be added to UI Receiver pod.")
}

func TestGetUIEnvironmentVariablesWorkloadsExplicitDisabled(t *testing.T) {
	spec := operatorv1alpha1.IBMLicenseServiceReporterSpec{
		EnableWorkloadsProcessing: false,
	}
	enableWorkloadsEnv := corev1.EnvVar{
		Name:  "ENABLE_WORKLOADS_PROCESSING",
		Value: "true",
	}
	envVars := getReporterUIEnvironmentVariables(spec)
	assert.False(t, res.Contains(envVars, enableWorkloadsEnv), "EnableWorkloadsProcessing is set, 'ENABLE_WORKLOADS_PROCESSING' environemnt variable should be added to LSR Receiver pod.")
}
