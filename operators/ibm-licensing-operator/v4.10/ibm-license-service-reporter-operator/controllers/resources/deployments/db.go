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
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	DatabaseContainerName              = "database"
	DatabaseCredentialsVolumeName      = "db-config" //nolint:gosec
	DatabaseConfigSecretName           = "license-service-reporter-hub-db-config"
	PersistentVolumeClaimVolumeName    = "data"
	DatabaseTmpVolumeName              = "db-tmp"
	DatabaseSocketsVolumeName          = "db-sockets"
	OperandReporterDatabaseImageEnvVar = "IBM_POSTGRESQL_IMAGE"
)

func getDatabaseProbeHandler() (corev1.ProbeHandler, error) {
	dbUser, err := resources.GetDatabaseUsername()
	if err != nil {
		return corev1.ProbeHandler{}, err
	}
	return corev1.ProbeHandler{
		Exec: &corev1.ExecAction{
			Command: []string{
				"psql",
				"-w",
				"-U",
				dbUser,
				"-d",
				secrets.DatabaseName,
				"-c",
				"SELECT 1",
			},
		},
	}, nil
}

func getEnvVariable(spec operatorv1alpha1.IBMLicenseServiceReporterSpec) []corev1.EnvVar {
	var environmentVariables []corev1.EnvVar

	if spec.EnvVariable != nil {
		for key, value := range spec.EnvVariable {
			environmentVariables = append(environmentVariables, corev1.EnvVar{
				Name:  key,
				Value: value,
			})
		}
	}
	environmentVariables = append(environmentVariables, corev1.EnvVar{
		Name:  "ENABLE_INSTANA_METRIC_COLLECTION",
		Value: strconv.FormatBool(spec.EnableInstanaMetricCollection),
	})
	return environmentVariables
}

func getDatabaseVolumeMounts() []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      PersistentVolumeClaimVolumeName,
			MountPath: secrets.DatabaseMountPath,
		},
		{
			Name:      DatabaseCredentialsVolumeName,
			MountPath: "/opt/ibm/licensing/" + secrets.DatabaseConfigSecretName,
			ReadOnly:  true,
		},
		{
			Name:      DatabaseSocketsVolumeName,
			MountPath: "/var/run/",
			ReadOnly:  false,
		},
		{
			Name:      DatabaseTmpVolumeName,
			MountPath: "/tmp/",
			ReadOnly:  false,
		},
	}
	return mounts
}

func GetDatabaseContainer(spec operatorv1alpha1.IBMLicenseServiceReporterSpec) (corev1.Container, error) {
	apiContainer, err := SetContainerFromEnv(spec.DatabaseContainer, OperandReporterDatabaseImageEnvVar)
	if err != nil {
		return corev1.Container{}, err
	}
	probeHandler, err := getDatabaseProbeHandler()
	if err != nil {
		return corev1.Container{}, err
	}
	cpu200m := resource.NewMilliQuantity(200, resource.DecimalSI)
	cpu300m := resource.NewMilliQuantity(300, resource.DecimalSI)
	memory256Mi := resource.NewQuantity(256*1024*1024, resource.BinarySI)
	memory300Mi := resource.NewQuantity(300*1024*1024, resource.BinarySI)
	ephStorage, _ := resource.ParseQuantity("256Mi")
	apiContainer.InitResourcesIfNil()
	apiContainer.SetImagePullPolicyIfNotSet()
	apiContainer.SetResourceLimitMemoryIfNotSet(*memory300Mi)
	apiContainer.SetResourceRequestMemoryIfNotSet(*memory256Mi)
	apiContainer.SetResourceLimitCPUIfNotSet(*cpu300m)
	apiContainer.SetResourceRequestCPUIfNotSet(*cpu200m)
	apiContainer.SetResourceRequestEphemeralStorageIfNotSet(ephStorage)
	container := GetContainerBase(apiContainer)
	container.Env = getEnvVariable(spec)
	container.VolumeMounts = getDatabaseVolumeMounts()
	container.Name = DatabaseContainerName
	container.LivenessProbe = GetLivenessProbe(probeHandler)
	container.ReadinessProbe = GetReadinessProbe(probeHandler)
	return container, nil
}
