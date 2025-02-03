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
	"fmt"

	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/pvc"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const emptyDirSizeLimit = "256Mi"

var DefaultSecretMode int32 = 420

func GetLicenseServiceReporterVolumes(config IBMLicenseServiceReporterConfig) ([]corev1.Volume, error) {
	instance := config.Instance
	spec := instance.Spec

	emptyDirSizeLimit, _ := resource.ParseQuantity(emptyDirSizeLimit)

	v := []corev1.Volume{
		{
			Name: APISecretTokenVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  secrets.GetAPISecretToken(spec),
					DefaultMode: &DefaultSecretMode,
				},
			},
		},
		{
			Name: ReporterAuthCookieSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  secrets.ReporterAuthCookieSecret,
					DefaultMode: &DefaultSecretMode,
				},
			},
		},
		{
			Name: ReporterAuthHtpasswdVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  secrets.ReporterHtpasswdSecretName,
					DefaultMode: &DefaultSecretMode,
				},
			},
		},
		{
			Name: PersistentVolumeClaimVolumeName,
			VolumeSource: corev1.VolumeSource{
				PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
					ClaimName: pvc.PersistentVolumeClaimName,
				},
			},
		},
		{
			Name: DatabaseCredentialsVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  DatabaseConfigSecretName,
					DefaultMode: &DefaultSecretMode,
				},
			},
		},
		{
			Name: ReceiverTmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &emptyDirSizeLimit,
				},
			},
		},
		{
			Name: DatabaseTmpVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &emptyDirSizeLimit,
				},
			},
		},
		{
			Name: DatabaseSocketsVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{
					SizeLimit: &emptyDirSizeLimit,
				},
			},
		},
		GetVolumeFromSecret(HTTPSCertsVolumeName, services.InternalCertSecretName),
	}

	oauth := config.Instance.Spec.Authentication.OAuth

	// Mount external secrets necessary for OAuth authentication
	if oauth.Enabled {
		secretName, found := oauth.FindOAuthParamValue(OAuth2ProxyClientSecretNameFlag)
		if !found {
			return []corev1.Volume{}, fmt.Errorf("client-secret-name for oauth authentication must be specified")
		}
		v = append(v, GetVolumeFromSecret(ReporterAuthClientSecretFileVolumeName, secretName))
		// TODO For now we don't require use of external certificate
		if secretName, found := oauth.FindOAuthParamValue(OAuth2ProxyProviderCASecretNameFlag); found {
			v = append(v, GetVolumeFromSecret(ReporterAuthProviderCAVolumeName, secretName))
		}
	}

	return v, nil
}

func GetVolumeFromSecret(volumeName, secretName string) corev1.Volume {
	trueVar := true
	return corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: &DefaultSecretMode,
				Optional:    &trueVar,
			},
		},
	}
}
