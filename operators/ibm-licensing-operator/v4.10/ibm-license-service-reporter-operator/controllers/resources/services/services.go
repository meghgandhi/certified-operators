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

package services

import (
	"fmt"

	"github.com/go-logr/logr"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	corev1 "k8s.io/api/core/v1"
	apieq "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

const (
	CustomExternalCertSecretName = "ibm-license-service-reporter-certs"
	InternalCertSecretName       = "ibm-license-service-reporter-cert-internal"
	OcpCertSecretNameTag         = "service.beta.openshift.io/serving-cert-secret-name" // #nosec
)

var (
	// The port in the pod
	ReporterUITargetPort = intstr.FromInt(3001)
	// The port that will be exposed by this service.
	ReceiverServicePort = intstr.FromInt(8080)
	// The port in the pod
	ReceiverTargetPort     = intstr.FromInt(8080)
	ReceiverTargetPortName = "receiver-port"
	// Auth port exposed by this service
	ReporterAuthServicePort = intstr.FromInt(8888)
	// Port in the pod
	ReporterAuthTargetPort     = intstr.FromInt(8888)
	ReporterAuthTargetPortName = "proxy"
)

func ReconcileService(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	expectedService := GetService(config)
	logger = AddResourceValuesToLog(logger, &expectedService)

	return ReconcileResource(
		config,
		&expectedService,
		&corev1.Service{},
		true,
		nil,
		IsServiceInDesiredState,
		PatchFoundWithSpecLabelsAndAnnotations,
		OverrideFoundWithExpected,
		logger,
		nil,
	)
}

func GetService(config IBMLicenseServiceReporterConfig) corev1.Service {
	instance := config.Instance
	metaLabels := LabelsForMeta(instance)
	var annotations map[string]string
	if config.IsServiceCAAPI {
		annotations = map[string]string{OcpCertSecretNameTag: InternalCertSecretName}
	}
	return corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:        LicenseReporterResourceBase,
			Namespace:   instance.GetNamespace(),
			Labels:      metaLabels,
			Annotations: MergeMaps(annotations, GetSpecAnnotations(instance)),
		},
		Spec: getServiceSpec(instance.GetName()),
	}
}

func getServiceSpec(instanceName string) corev1.ServiceSpec {
	return corev1.ServiceSpec{
		Type: corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{
				Name:       ReceiverTargetPortName,
				Port:       ReceiverServicePort.IntVal,
				TargetPort: ReceiverTargetPort,
				Protocol:   corev1.ProtocolTCP,
			},
			{
				Name:       ReporterAuthTargetPortName,
				Port:       ReporterAuthServicePort.IntVal,
				TargetPort: ReporterAuthTargetPort,
				Protocol:   corev1.ProtocolTCP,
			},
		},
		Selector: LabelsForSelector(instanceName),
	}
}

func hashPort(port corev1.ServicePort) string {
	return fmt.Sprintf("%s:%d:%s:%s", port.Name, port.Port, port.TargetPort.String(), port.Protocol)
}

func IsServiceInDesiredState(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, _ logr.Logger) (ResourceUpdateStatus, error) {
	foundService := found.(*corev1.Service)
	expectedService := expected.(*corev1.Service)

	// Check service type
	if foundService.Spec.Type != expectedService.Spec.Type {
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// Check ports
	if !UnorderedContainsSliceWithHashFunc(foundService.Spec.Ports, expectedService.Spec.Ports, hashPort) {
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// Check selector
	if !apieq.Semantic.DeepEqual(foundService.Spec.Selector, expectedService.Spec.Selector) {
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// Spec.labels support for resource updates
	if !MapHasAllPairsFromOther(found.GetLabels(), GetSpecLabels(config.Instance)) {
		return ResourceUpdateStatus{IsPatchSufficient: true}, nil
	}

	// Check annotations, including spec.annotations
	if !MapHasAllPairsFromOther(foundService.GetAnnotations(), expectedService.GetAnnotations()) {
		return ResourceUpdateStatus{IsPatchSufficient: true}, nil
	}

	return ResourceUpdateStatus{IsInDesiredState: true}, nil
}
