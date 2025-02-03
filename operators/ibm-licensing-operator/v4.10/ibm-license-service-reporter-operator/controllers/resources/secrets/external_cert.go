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
	"context"
	"fmt"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func ReconcileExternalCertificateSecret(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	instance := config.Instance
	if config.IsRouteAPI && instance.Spec.HTTPSCertsSource != CustomCertsSource {
		expectedSecret := corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:        ExternalCertName,
				Namespace:   instance.GetNamespace(),
				Labels:      LabelsForMeta(instance),
				Annotations: GetSpecAnnotations(instance),
			},
		}
		logger = AddResourceValuesToLog(logger, &expectedSecret)
		// we need to get route data first
		routeNamespacedName := types.NamespacedName{Namespace: instance.GetNamespace(), Name: LicenseReporterResourceBase}
		route := routev1.Route{}
		err := config.Client.Get(context.TODO(), routeNamespacedName, &route)
		if err != nil {
			return fmt.Errorf("cannot get route: %v", err)
		}
		routeHost := []string{route.Spec.Host}
		return ReconcileResource(
			config,
			&expectedSecret,
			&corev1.Secret{},
			true,
			func(expected ExpectedObject) (ExpectedObject, error) {
				return overrideExternalCertExpectedSecret(logger, config, routeHost, expected)
			},
			func(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
				foundSecret := found.(*corev1.Secret)
				expectedSecret := expected.(*corev1.Secret)

				return IsSecretCertificateInDesiredState(config, logger, *foundSecret, *expectedSecret, routeHost)
			},
			PatchFoundWithSpecLabelsAndAnnotations,
			func(found FoundObject, expected ExpectedObject) (client.Object, bool, error) {
				overriddenExpected, err := overrideExternalCertExpectedSecret(logger, config, routeHost, expected)
				if err != nil {
					return nil, false, err
				}
				return overriddenExpected, false, nil
			},
			logger,
			nil,
		)
	}

	return nil
}

func overrideExternalCertExpectedSecret(logger logr.Logger, config IBMLicenseServiceReporterConfig, routeHosts []string, expected ExpectedObject) (ExpectedObject, error) {
	// get secret with the host from route data
	expectedSecret, err := getSelfSignedCertWithOwnerReference(logger, config, types.NamespacedName{
		Namespace: expected.GetNamespace(),
		Name:      expected.GetName(),
	}, routeHosts)
	if err != nil {
		return nil, fmt.Errorf("error generating external self-signed certificate: %v", err)
	}
	return expectedSecret, nil
}
