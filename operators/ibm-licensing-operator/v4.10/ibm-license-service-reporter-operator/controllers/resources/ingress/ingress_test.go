package ingress

import (
	"flag"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/mocks"
	"go.uber.org/zap/zapcore"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
)

var instanceName = "reporter"

func TestShouldUpdateIngress(t *testing.T) {
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	logger := zap.New(zap.UseFlagOptions(&opts))

	ingressTests := []struct {
		ingressName      string
		getIngressMethod func(config IBMLicenseServiceReporterConfig) *networkingv1.Ingress
		path             string
		port             int32
	}{
		{
			"ibm-license-service-reporter-api-ingress",
			GetApiIngress,
			"/",
			8080,
		},
		{
			"ibm-license-service-reporter-console-ingress",
			GetConsoleIngress,
			"/license-service-reporter",
			8888,
		},
	}
	for _, ingressTest := range ingressTests {
		tests := []struct {
			name                 string
			foundIngress         *networkingv1.Ingress
			config               IBMLicenseServiceReporterConfig
			expectedShouldUpdate resources.ResourceUpdateStatus
		}{
			{
				"Found ingress the same as expected - no update needed",
				getMockIngress(ingressTest.ingressName, "ingress-ns", mocks.GetLabelsForMeta(instanceName), map[string]string{}, networkingv1.IngressSpec{
					IngressClassName: ptr.To("ingress-class"),
					Rules: []networkingv1.IngressRule{
						{
							Host: "host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     ingressTest.path,
											PathType: ptr.To(pathType),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: LicenseReporterResourceBase,
													Port: networkingv1.ServiceBackendPort{
														Number: ingressTest.port,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				getMockConfig(v1alpha1.IBMLicenseServiceReporterSpec{
					IngressEnabled: true,
					IngressOptions: &v1alpha1.IBMLicenseServiceReporterIngressOptions{
						CommonOptions: &v1alpha1.IBMLicenseServiceReporterIngressCommonOptions{
							Host:             ptr.To("host"),
							IngressClassName: ptr.To("ingress-class"),
						},
					},
				}),
				resources.ResourceUpdateStatus{
					IsInDesiredState:  true,
					IsPatchSufficient: false,
				},
			},
			{
				"Found ingress has different labels - update needed",
				getMockIngress(ingressTest.ingressName, "ingress-ns", map[string]string{"label": "value"}, map[string]string{}, networkingv1.IngressSpec{
					IngressClassName: ptr.To("ingress-class"),
					Rules: []networkingv1.IngressRule{
						{
							Host: "host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     ingressTest.path,
											PathType: ptr.To(pathType),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: LicenseReporterResourceBase,
													Port: networkingv1.ServiceBackendPort{
														Number: ingressTest.port,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				getMockConfig(v1alpha1.IBMLicenseServiceReporterSpec{
					IngressEnabled: true,
					IngressOptions: &v1alpha1.IBMLicenseServiceReporterIngressOptions{
						CommonOptions: &v1alpha1.IBMLicenseServiceReporterIngressCommonOptions{
							Host:             ptr.To("host"),
							IngressClassName: ptr.To("ingress-class"),
						},
					},
				}),
				resources.ResourceUpdateStatus{
					IsInDesiredState:  false,
					IsPatchSufficient: false,
				},
			},
			{
				"Found ingress has different annotations - update needed",
				getMockIngress(ingressTest.ingressName, "ingress-ns", mocks.GetLabelsForMeta(instanceName), map[string]string{}, networkingv1.IngressSpec{
					IngressClassName: ptr.To("ingress-class"),
					Rules: []networkingv1.IngressRule{
						{
							Host: "host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     ingressTest.path,
											PathType: ptr.To(pathType),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: LicenseReporterResourceBase,
													Port: networkingv1.ServiceBackendPort{
														Number: ingressTest.port,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				getMockConfig(v1alpha1.IBMLicenseServiceReporterSpec{
					IngressEnabled: true,
					IngressOptions: &v1alpha1.IBMLicenseServiceReporterIngressOptions{
						CommonOptions: &v1alpha1.IBMLicenseServiceReporterIngressCommonOptions{
							Host:             ptr.To("host"),
							IngressClassName: ptr.To("ingress-class"),
						},
						ApiOptions: &v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions{
							Annotations: map[string]string{"ingress.config": "pls-work"},
						},
						ConsoleOptions: &v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions{
							Annotations: map[string]string{"ingress.config": "pls-work-here-too"},
						},
					},
				}),
				resources.ResourceUpdateStatus{
					IsInDesiredState:  false,
					IsPatchSufficient: false,
				},
			},
			{
				"Found ingress has different spec - update needed",
				getMockIngress(ingressTest.ingressName, "ingress-ns", mocks.GetLabelsForMeta(instanceName), map[string]string{}, networkingv1.IngressSpec{
					IngressClassName: ptr.To("ingress-class"),
					Rules: []networkingv1.IngressRule{
						{
							Host: "host",
							IngressRuleValue: networkingv1.IngressRuleValue{
								HTTP: &networkingv1.HTTPIngressRuleValue{
									Paths: []networkingv1.HTTPIngressPath{
										{
											Path:     ingressTest.path,
											PathType: ptr.To(pathType),
											Backend: networkingv1.IngressBackend{
												Service: &networkingv1.IngressServiceBackend{
													Name: LicenseReporterResourceBase,
													Port: networkingv1.ServiceBackendPort{
														Number: ingressTest.port,
													},
												},
											},
										},
									},
								},
							},
						},
					},
				}),
				getMockConfig(v1alpha1.IBMLicenseServiceReporterSpec{
					IngressEnabled: true,
					IngressOptions: &v1alpha1.IBMLicenseServiceReporterIngressOptions{
						CommonOptions: &v1alpha1.IBMLicenseServiceReporterIngressCommonOptions{
							Host:             ptr.To("host"),
							IngressClassName: ptr.To("ingress-class-2"),
						},
					},
				}),
				resources.ResourceUpdateStatus{
					IsInDesiredState:  false,
					IsPatchSufficient: false,
				},
			},
		}
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				expectedIngress := ingressTest.getIngressMethod(tt.config)
				result, _ := isIngressInDesiredState(IBMLicenseServiceReporterConfig{}, tt.foundIngress, expectedIngress, logger)
				assert.Equal(t, tt.expectedShouldUpdate, result)
			})
		}
	}
}

func getMockConfig(spec v1alpha1.IBMLicenseServiceReporterSpec) IBMLicenseServiceReporterConfig {
	scheme := runtime.NewScheme()
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&mocks.ConsoleRoute).Build()

	instance := v1alpha1.IBMLicenseServiceReporter{}
	instance.Namespace = "test"
	instance.Name = instanceName
	instance.Spec = spec

	return IBMLicenseServiceReporterConfig{
		Instance: instance,
		Client:   client,
		Scheme:   scheme,
	}
}

func getMockIngress(name, namespace string, labels, annotations map[string]string, spec networkingv1.IngressSpec) *networkingv1.Ingress {
	return &networkingv1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Namespace:   namespace,
			Annotations: annotations,
			Labels:      labels,
		},
		Spec: spec,
	}
}
