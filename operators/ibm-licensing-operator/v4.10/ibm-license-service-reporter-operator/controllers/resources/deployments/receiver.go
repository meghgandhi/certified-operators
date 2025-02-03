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
	"strconv"

	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ReceiverContainerName              = "receiver"
	APISecretTokenVolumeName           = "api-token"
	ReceiverTmpVolumeName              = "receiver-tmp"
	HTTPSCertsVolumeName               = "license-reporter-https-certs"
	OperandReporterReceiverImageEnvVar = "IBM_LICENSE_SERVICE_REPORTER_IMAGE"
)

func getReceiverProbeHandler() corev1.ProbeHandler {
	return corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   "/",
			Port:   services.ReceiverTargetPort,
			Scheme: "HTTPS",
		},
	}
}

func getReceiverVolumeMounts() []corev1.VolumeMount {
	var volumeMounts = []corev1.VolumeMount{
		{
			Name:      APISecretTokenVolumeName,
			MountPath: "/opt/ibm/licensing",
			ReadOnly:  true,
		},
		{
			Name:      DatabaseCredentialsVolumeName,
			MountPath: "/opt/ibm/licensing/" + DatabaseConfigSecretName,
			ReadOnly:  true,
		},
		{
			Name:      ReceiverTmpVolumeName,
			MountPath: "/tmp",
			ReadOnly:  false,
		},
		{
			Name:      HTTPSCertsVolumeName,
			MountPath: "/opt/licensing/certs/",
			ReadOnly:  true,
		},
	}
	return volumeMounts
}

func getReceiverEnvVariables(spec operatorv1alpha1.IBMLicenseServiceReporterSpec) []corev1.EnvVar {
	environmentVariables := []corev1.EnvVar{
		{
			Name:  "HTTPS_CERTS_SOURCE",
			Value: secrets.ExternalCertsSource,
		},
		{
			Name:  "ENABLE_INSTANA_METRIC_COLLECTION",
			Value: strconv.FormatBool(spec.EnableInstanaMetricCollection),
		},
	}
	if spec.IsDebug() {
		environmentVariables = append(environmentVariables, corev1.EnvVar{
			Name:  "logging.level.com.ibm",
			Value: "DEBUG",
		})
	}
	if spec.IsDebug() || spec.IsVerbose() {
		environmentVariables = append(environmentVariables, corev1.EnvVar{
			Name:  "SPRING_PROFILES_ACTIVE",
			Value: "verbose",
		})
	}
	if spec.EnableWorkloadsProcessing {
		environmentVariables = append(environmentVariables, corev1.EnvVar{
			Name:  "ENABLE_WORKLOADS_PROCESSING",
			Value: "true",
		})
	}
	if spec.WorkloadsCustomColumnsRetencyDays != nil {
		environmentVariables = append(environmentVariables, corev1.EnvVar{
			Name:  "WORKLOADS_CC_RETENTION_PERIOD_DAYS",
			Value: strconv.Itoa(*spec.WorkloadsCustomColumnsRetencyDays),
		})
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

func GetReceiverContainer(config IBMLicenseServiceReporterConfig) (corev1.Container, error) {
	instance := config.Instance
	spec := instance.Spec
	apiContainer, err := SetContainerFromEnv(spec.ReceiverContainer, OperandReporterReceiverImageEnvVar)
	if err != nil {
		return corev1.Container{}, err
	}
	cpu200m := resource.NewMilliQuantity(200, resource.DecimalSI)
	cpu300m := resource.NewMilliQuantity(300, resource.DecimalSI)
	memory256Mi := resource.NewQuantity(256*1024*1024, resource.BinarySI)
	memory384Mi := resource.NewQuantity(384*1024*1024, resource.BinarySI) // TODO: would be better to use resource.ParseQuantity("384Mi") to avoid mistakes
	ephStorage, _ := resource.ParseQuantity("256Mi")
	apiContainer.InitResourcesIfNil()
	apiContainer.SetImagePullPolicyIfNotSet()
	apiContainer.SetResourceLimitMemoryIfNotSet(*memory384Mi)
	apiContainer.SetResourceRequestMemoryIfNotSet(*memory256Mi)
	apiContainer.SetResourceLimitCPUIfNotSet(*cpu300m)
	apiContainer.SetResourceRequestCPUIfNotSet(*cpu200m)
	apiContainer.SetResourceRequestEphemeralStorageIfNotSet(ephStorage)
	container := GetContainerBase(apiContainer)
	container.Env = getReceiverEnvVariables(spec)
	container.VolumeMounts = getReceiverVolumeMounts()
	container.Name = ReceiverContainerName
	container.Ports = []corev1.ContainerPort{
		{
			ContainerPort: services.ReceiverTargetPort.IntVal,
			Protocol:      corev1.ProtocolTCP,
		},
	}
	container.LivenessProbe = GetLivenessProbe(getReceiverProbeHandler())
	container.ReadinessProbe = GetReadinessProbe(getReceiverProbeHandler())
	return container, nil
}
