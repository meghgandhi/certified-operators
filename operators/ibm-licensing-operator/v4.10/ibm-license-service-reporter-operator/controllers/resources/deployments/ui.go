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

package deployments

import (
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	UIContainerName              = "reporter-ui"
	OperandReporterUIImageEnvVar = "IBM_LICENSE_SERVICE_REPORTER_UI_IMAGE"
)

func getReporterUIEnvironmentVariables(spec operatorv1alpha1.IBMLicenseServiceReporterSpec) []corev1.EnvVar {
	var environmentVariables = []corev1.EnvVar{
		{
			Name:  "NODE_TLS_REJECT_UNAUTHORIZED",
			Value: "0",
		},
		{
			Name:  "HTTP_PORT",
			Value: services.ReporterUITargetPort.String(),
		},
		{
			Name:  "baseUrl",
			Value: "https://localhost:8080",
		},
		{
			Name: "apiToken",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: secrets.GetAPISecretToken(spec),
					},
					Key: secrets.ApiReceiverSecretTokenKeyName,
				},
			},
		},
	}
	if spec.EnableWorkloadsProcessing {
		environmentVariables = append(environmentVariables, corev1.EnvVar{
			Name:  "ENABLE_WORKLOADS_PROCESSING",
			Value: "true",
		})
	}
	if spec.EnableInstanaMetricCollection {
		environmentVariables = append(environmentVariables,
			corev1.EnvVar{
				Name: "INSTANA_AGENT_HOST",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.hostIP",
					},
				},
			},
			corev1.EnvVar{
				Name:  "INSTANA_DISABLE_USE_OPENTELEMETRY",
				Value: "true",
			},
		)
	}
	if spec.EnvVariable != nil {
		for key, value := range spec.EnvVariable {
			environmentVariables = append(environmentVariables, corev1.EnvVar{
				Name:  key,
				Value: value,
			})
		}
	}
	return environmentVariables

}

func getReporterUIProbeHandler() corev1.ProbeHandler {
	return corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   "/license-service-reporter/version.txt",
			Port:   services.ReporterUITargetPort,
			Scheme: "HTTP",
		},
	}
}

func GetReporterUIContainer(config IBMLicenseServiceReporterConfig) (corev1.Container, error) {
	instance := config.Instance
	spec := instance.Spec
	uiContainer, err := SetContainerFromEnv(spec.ReporterUIContainer, OperandReporterUIImageEnvVar)
	if err != nil {
		return corev1.Container{}, err
	}
	cpu200m := resource.NewMilliQuantity(200, resource.DecimalSI)
	cpu300m := resource.NewMilliQuantity(300, resource.DecimalSI)
	memory256Mi := resource.NewQuantity(256*1024*1024, resource.BinarySI)
	memory300Mi := resource.NewQuantity(300*1024*1024, resource.BinarySI)
	ephStorage, _ := resource.ParseQuantity("256Mi")
	uiContainer.InitResourcesIfNil()
	uiContainer.SetImagePullPolicyIfNotSet()
	uiContainer.SetResourceLimitMemoryIfNotSet(*memory300Mi)
	uiContainer.SetResourceRequestMemoryIfNotSet(*memory256Mi)
	uiContainer.SetResourceLimitCPUIfNotSet(*cpu300m)
	uiContainer.SetResourceRequestCPUIfNotSet(*cpu200m)
	uiContainer.SetResourceRequestEphemeralStorageIfNotSet(ephStorage)
	container := GetContainerBase(uiContainer)
	container.Env = getReporterUIEnvironmentVariables(spec)
	container.VolumeMounts = getAuthSecretVolumeMounts(config)
	container.Name = UIContainerName
	container.LivenessProbe = GetLivenessProbe(getReporterUIProbeHandler())
	container.ReadinessProbe = GetReadinessProbe(getReporterUIProbeHandler())
	return container, nil
}
