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

package secrets

import (
	"github.com/go-logr/logr"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const ApiReceiverSecretTokenKeyName = "token"
const DefaultReporterTokenSecretName = "ibm-license-service-reporter-token" // secret used by LS to push data to LSR

func GetAPISecretToken(spec v1alpha1.IBMLicenseServiceReporterSpec) string {
	name := spec.APISecretToken
	if name == "" {
		name = DefaultReporterTokenSecretName
	}
	return name
}

func ReconcileAPISecretToken(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	instance := config.Instance
	name := GetAPISecretToken(instance.Spec)
	expectedSecret, err := GetSecretToken(
		name,
		instance.GetNamespace(),
		ApiReceiverSecretTokenKeyName,
		LabelsForMeta(instance),
		GetSpecAnnotations(instance),
	)
	logger = AddResourceValuesToLog(logger, &expectedSecret)
	if err != nil {
		return err
	}
	return ReconcileResource(
		config,
		&expectedSecret,
		&corev1.Secret{},
		true,
		nil,
		func(config IBMLicenseServiceReporterConfig, found FoundObject, _ ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
			foundSecret := found.(*corev1.Secret)

			if _, ok := foundSecret.Data[ApiReceiverSecretTokenKeyName]; !ok {
				logger.Info("Updating " + name + " due to not having " + ApiReceiverSecretTokenKeyName + " in Data")
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}

			// Spec.labels support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetLabels(), GetSpecLabels(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			// Spec.annotations support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			return ResourceUpdateStatus{IsInDesiredState: true}, nil
		},
		PatchFoundWithSpecLabelsAndAnnotations,
		OverrideFoundWithExpected,
		logger,
		nil,
	)
}

func GetSecretToken(name, namespace, secretKey string, metaLabels, metaAnnotations map[string]string) (corev1.Secret, error) {
	randString, err := RandString(24)
	if err != nil {
		return corev1.Secret{}, err
	}
	expectedSecret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Labels:      metaLabels,
			Annotations: metaAnnotations,
		},
		Type:       corev1.SecretTypeOpaque,
		StringData: map[string]string{secretKey: randString},
	}
	return expectedSecret, nil
}
