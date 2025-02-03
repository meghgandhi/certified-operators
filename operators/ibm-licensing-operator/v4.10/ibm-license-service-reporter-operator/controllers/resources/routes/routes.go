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
	"fmt"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const ReporterConsoleRouteName = "ibm-lsr-console"
const ReporterAPIRouteName = LicenseReporterResourceBase

func annotationsForReporterRoutes(instance operatorv1alpha1.IBMLicenseServiceReporter) map[string]string {
	annotations := map[string]string{"haproxy.router.openshift.io/timeout": "90s"}

	return MergeMaps(annotations, GetSpecAnnotations(instance))
}

func GetReporterRoute(instance operatorv1alpha1.IBMLicenseServiceReporter, defaultRouteTLS routev1.TLSConfig) *routev1.Route {
	var tls *routev1.TLSConfig
	if instance.Spec.RouteOptions != nil {
		if instance.Spec.RouteOptions.TLS != nil {
			tls = instance.Spec.RouteOptions.TLS
		}
	} else {
		tls = &defaultRouteTLS
	}

	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ReporterAPIRouteName,
			Namespace:   instance.GetNamespace(),
			Labels:      LabelsForMeta(instance),
			Annotations: annotationsForReporterRoutes(instance),
		},
		Spec: routev1.RouteSpec{
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: LicenseReporterResourceBase,
			},
			Port: &routev1.RoutePort{
				TargetPort: services.ReceiverServicePort, // TODO: powinny miec service porty tylko routy
			},
			TLS: tls,
		},
	}
}

func GetConsoleRoute(instance operatorv1alpha1.IBMLicenseServiceReporter, routeTLS routev1.TLSConfig) *routev1.Route {
	return &routev1.Route{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ReporterConsoleRouteName,
			Namespace:   instance.GetNamespace(),
			Labels:      LabelsForMeta(instance),
			Annotations: annotationsForReporterRoutes(instance),
		},
		Spec: routev1.RouteSpec{
			Path: "/license-service-reporter",
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: LicenseReporterResourceBase,
			},
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(services.ReporterAuthTargetPortName),
			},
			TLS: &routeTLS,
		},
	}
}

func ReconcileRouteWithTLS(logger logr.Logger, config IBMLicenseServiceReporterConfig, tlsConfig routev1.TLSConfig, expectedRoute *routev1.Route) error {
	logger = AddResourceValuesToLog(logger, expectedRoute)

	return ReconcileResource(config, expectedRoute, &routev1.Route{}, true, nil, IsRouteInDesiredState, PatchFoundWithSpecLabelsAndAnnotations, OverrideFoundWithExpected, logger, nil)
}

func getTLSDataAsString(route routev1.Route) string {
	return fmt.Sprintf("{Termination: %v, InsecureEdgeTerminationPolicy: %v, Certificate: %s, CACertificate: %s, DestinationCACertificate: %s}",
		route.Spec.TLS.Termination, route.Spec.TLS.InsecureEdgeTerminationPolicy,
		route.Spec.TLS.Certificate, route.Spec.TLS.CACertificate, route.Spec.TLS.DestinationCACertificate)
}

func CompareRoutes(config IBMLicenseServiceReporterConfig, foundRoute, expectedRoute routev1.Route, logger logr.Logger) ResourceUpdateStatus {
	if foundRoute.ObjectMeta.Name != expectedRoute.ObjectMeta.Name {
		logger.Info("Names not equal", "old", foundRoute.ObjectMeta.Name, "new", expectedRoute.ObjectMeta.Name)
		return ResourceUpdateStatus{IsInDesiredState: false}
	}
	if foundRoute.Spec.To.Name != expectedRoute.Spec.To.Name {
		logger.Info("Specs To Name not equal",
			"old", fmt.Sprintf("%v", foundRoute.Spec),
			"new", fmt.Sprintf("%v", expectedRoute.Spec))
		return ResourceUpdateStatus{IsInDesiredState: false}
	}
	if foundRoute.Spec.TLS == nil && expectedRoute.Spec.TLS != nil {
		logger.Info("Found Route has empty TLS options, but Expected Route has not empty TLS options",
			"old", fmt.Sprintf("%v", foundRoute.Spec.TLS),
			"new", fmt.Sprintf("%v", getTLSDataAsString(expectedRoute)))
		return ResourceUpdateStatus{IsInDesiredState: false}
	}
	if foundRoute.Spec.TLS != nil && expectedRoute.Spec.TLS == nil {
		logger.Info("Expected Route has empty TLS options, but Found Route has not empty TLS options",
			"old", fmt.Sprintf("%v", getTLSDataAsString(foundRoute)),
			"new", fmt.Sprintf("%v", expectedRoute.Spec.TLS))
		return ResourceUpdateStatus{IsInDesiredState: false}
	}
	if foundRoute.Spec.TLS != nil && expectedRoute.Spec.TLS != nil {
		if foundRoute.Spec.TLS.Termination != expectedRoute.Spec.TLS.Termination {
			logger.Info("Expected Route has different TLS Termination option than Found Route",
				"old", fmt.Sprintf("%v", foundRoute.Spec.TLS.Termination),
				"new", fmt.Sprintf("%v", expectedRoute.Spec.TLS.Termination))
			return ResourceUpdateStatus{IsInDesiredState: false}
		}
		if foundRoute.Spec.TLS.InsecureEdgeTerminationPolicy != expectedRoute.Spec.TLS.InsecureEdgeTerminationPolicy {
			logger.Info("Expected Route has different TLS InsecureEdgeTerminationPolicy option than Found Route",
				"old", fmt.Sprintf("%v", foundRoute.Spec.TLS.InsecureEdgeTerminationPolicy),
				"new", fmt.Sprintf("%v", expectedRoute.Spec.TLS.InsecureEdgeTerminationPolicy))
			return ResourceUpdateStatus{IsInDesiredState: false}
		}
		if !areTLSCertsSame(*expectedRoute.Spec.TLS, *foundRoute.Spec.TLS) {
			logger.Info("Expected route has different certificate info in the TLS section than Found Route",
				"old", fmt.Sprintf("%v", getTLSDataAsString(foundRoute)),
				"new", fmt.Sprintf("%v", getTLSDataAsString(expectedRoute)))
			return ResourceUpdateStatus{IsInDesiredState: false}
		}
	}

	// Spec.labels support for resource updates
	if !MapHasAllPairsFromOther(foundRoute.GetLabels(), GetSpecLabels(config.Instance)) {
		return ResourceUpdateStatus{IsPatchSufficient: true}
	}

	// Spec.annotations support for resource updates
	if !MapHasAllPairsFromOther(foundRoute.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
		return ResourceUpdateStatus{IsPatchSufficient: true}
	}

	return ResourceUpdateStatus{IsInDesiredState: true}
}

func areTLSCertsSame(expected, found routev1.TLSConfig) bool {
	return expected.CACertificate == found.CACertificate &&
		expected.Certificate == found.Certificate &&
		expected.Key == found.Key &&
		expected.DestinationCACertificate == found.DestinationCACertificate
}

func IsRouteInDesiredState(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
	expectedRoute := *(expected.(*routev1.Route))
	foundRoute := *(found.(*routev1.Route))

	return CompareRoutes(config, foundRoute, expectedRoute, logger), nil
}

func GetExistingRoute(client client.Client, routeName, namespace string) (*routev1.Route, error) {
	foundRoute := routev1.Route{}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: routeName}, &foundRoute); err != nil {
		return &routev1.Route{}, fmt.Errorf("could not retrieve route "+routeName+": %v", err)
	}

	return &foundRoute, nil
}
