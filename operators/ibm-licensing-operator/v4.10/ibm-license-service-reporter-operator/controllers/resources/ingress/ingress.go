package ingress

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	networkingv1 "k8s.io/api/networking/v1"
	apieq "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	pathType = networkingv1.PathTypeImplementationSpecific

	defaultApiPath = "/"
	apiIngressName = LicenseReporterResourceBase + "-api-ingress"

	DefaultConsolePath = "/license-service-reporter"
	ConsoleIngressName = LicenseReporterResourceBase + "-console-ingress"
)

func ReconcileApiIngress(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	if config.Instance.Spec.IngressEnabled {
		expectedIngress := GetApiIngress(config)

		return ReconcileResource(
			config,
			expectedIngress,
			&networkingv1.Ingress{},
			true,
			nil,
			isIngressInDesiredState,
			PatchFoundWithSpecLabelsAndAnnotations,
			OverrideFoundWithExpected,
			logger,
			nil,
		)
	}
	return nil
}

func ReconcileConsoleIngress(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	if config.Instance.Spec.IngressEnabled {
		expectedIngress := GetConsoleIngress(config)

		return ReconcileResource(
			config,
			expectedIngress,
			&networkingv1.Ingress{},
			true,
			nil,
			isIngressInDesiredState,
			PatchFoundWithSpecLabelsAndAnnotations,
			OverrideFoundWithExpected,
			logger,
			RestartReporterOperandPod,
		)
	}
	return nil
}

func GetApiIngress(config IBMLicenseServiceReporterConfig) *networkingv1.Ingress {
	return getIngress(
		config.Instance,
		config.Instance.Spec.IngressOptions.ApiOptions,
		apiIngressName,
		services.ReceiverServicePort.IntVal,
		defaultApiPath,
	)
}

func GetConsoleIngress(config IBMLicenseServiceReporterConfig) *networkingv1.Ingress {
	return getIngress(
		config.Instance,
		config.Instance.Spec.IngressOptions.ConsoleOptions,
		ConsoleIngressName,
		services.ReporterAuthTargetPort.IntVal,
		DefaultConsolePath,
	)
}

func getIngress(
	instance v1alpha1.IBMLicenseServiceReporter,
	specificOptions *v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions,
	name string,
	port int32,
	defaultPath string,
) *networkingv1.Ingress {
	var (
		tls              []networkingv1.IngressTLS
		host             string
		annotations      map[string]string
		ingressClassName *string
	)
	ingressOptions := instance.Spec.IngressOptions
	path := defaultPath
	if ingressOptions != nil {
		if specificOptions != nil {
			tls = specificOptions.TLS
			annotations = specificOptions.Annotations

			if specificOptions.Path != nil {
				path = *specificOptions.Path
			}
		}
		if ingressOptions.CommonOptions.Host != nil {
			host = *ingressOptions.CommonOptions.Host
		}
		ingressClassName = ingressOptions.CommonOptions.IngressClassName
	}
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   instance.GetNamespace(),
			Annotations: annotations,
			Labels:      LabelsForMeta(instance),
		},
		Spec: networkingv1.IngressSpec{
			TLS:              tls,
			IngressClassName: ingressClassName,
			Rules: []networkingv1.IngressRule{
				{
					Host: host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: []networkingv1.HTTPIngressPath{
								{
									Path:     path,
									PathType: ptr.To(pathType),
									Backend: networkingv1.IngressBackend{
										Service: &networkingv1.IngressServiceBackend{
											Name: LicenseReporterResourceBase,
											Port: networkingv1.ServiceBackendPort{
												Number: port,
											},
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func isIngressInDesiredState(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
	expectedIngress := expected.(*networkingv1.Ingress)
	foundIngress := found.(*networkingv1.Ingress)

	if foundIngress.ObjectMeta.Name != expectedIngress.ObjectMeta.Name {
		logger.Info("Ingress has wrong name", "found", foundIngress.ObjectMeta.Name, "expected", expectedIngress.ObjectMeta.Name)
	} else if !MapHasAllPairsFromOther(foundIngress.ObjectMeta.Labels, expectedIngress.ObjectMeta.Labels) {
		logger.Info("Ingress has wrong labels",
			"found", fmt.Sprintf("%v", foundIngress.ObjectMeta.Labels),
			"expected", fmt.Sprintf("%v", expectedIngress.ObjectMeta.Labels))
	} else if !apieq.Semantic.DeepEqual(foundIngress.ObjectMeta.Annotations, expectedIngress.ObjectMeta.Annotations) {
		logger.Info("Ingress has wrong annotations",
			"found", fmt.Sprintf("%v", foundIngress.ObjectMeta.Annotations),
			"expected", fmt.Sprintf("%v", expectedIngress.ObjectMeta.Annotations))
	} else if !apieq.Semantic.DeepEqual(foundIngress.Spec, expectedIngress.Spec) {
		logger.Info("Ingress has wrong spec",
			"found", fmt.Sprintf("%v", foundIngress.Spec),
			"expected", fmt.Sprintf("%v", expectedIngress.Spec))
	} else {
		return ResourceUpdateStatus{IsInDesiredState: true}, nil
	}
	return ResourceUpdateStatus{IsInDesiredState: false}, nil
}

func GetExistingIngress(client client.Client, ingressName string, namespace string) (*networkingv1.Ingress, error) {
	foundIngress := networkingv1.Ingress{}

	if err := client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ingressName}, &foundIngress); err != nil {
		return &networkingv1.Ingress{}, fmt.Errorf("could not retrieve ingress %s: %v", ConsoleIngressName, err)
	}

	return &foundIngress, nil
}
