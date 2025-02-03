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

package routes

import (
	"context"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"k8s.io/apimachinery/pkg/types"
)

func ReconcileRoutesWithoutCertificates(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	// Reconcile API route
	err := ReconcileRouteWithoutCertificates(logger, config, LicenseReporterResourceBase, GetReporterRoute)
	if err != nil {
		return err
	}

	// Reconcile UI route
	return ReconcileRouteWithoutCertificates(logger, config, ReporterConsoleRouteName, GetConsoleRoute)
}

func ReconcileRouteWithoutCertificates(logger logr.Logger, config IBMLicenseServiceReporterConfig, name string, getExpectedRoute func(instance operatorv1alpha1.IBMLicenseServiceReporter, defaultRouteTLS routev1.TLSConfig) *routev1.Route) error {
	if config.IsRouteAPI {
		instance := config.Instance
		routeNamespacedName := types.NamespacedName{Namespace: instance.GetNamespace(), Name: name}
		route := &routev1.Route{}
		tlsConfig := routev1.TLSConfig{
			Termination:                   routev1.TLSTerminationReencrypt,
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
		}
		expectedRoute := getExpectedRoute(instance, tlsConfig)

		err := config.Client.Get(context.TODO(), routeNamespacedName, route)
		if err != nil {
			logger.Info("Route does not exist, reconciling route without certificates")
			return ReconcileRouteWithTLS(logger, config, tlsConfig, expectedRoute)
		}

		// Reconcile when route exists but only patch spec.labels
		return ReconcileResource(
			config,
			expectedRoute,
			route,
			true,
			nil,
			func(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {

				// Spec.labels support for resource updates
				if !MapHasAllPairsFromOther(found.GetLabels(), GetSpecLabels(config.Instance)) {
					return ResourceUpdateStatus{IsPatchSufficient: true}, nil
				}

				// Spec.annotations support for resource updates
				if !MapHasAllPairsFromOther(found.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
					return ResourceUpdateStatus{IsPatchSufficient: true}, nil
				}

				return ResourceUpdateStatus{IsInDesiredState: true}, nil
			},
			PatchFoundWithSpecLabelsAndAnnotations,
			nil,
			logger,
			nil,
		)
	}
	return nil
}
