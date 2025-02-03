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
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1/auth"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/mocks"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestGetAuthSecretVolumeMounts(t *testing.T) {
	type args struct {
		config IBMLicenseServiceReporterConfig
	}
	type want struct {
		volumeMounts []corev1.VolumeMount
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"Volume mounts for basic auth only",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: "test",
						},
					},
					IsRouteAPI: true,
				},
			},
			want{
				mocks.BasicAuthVolumeMounts,
			},
		},
		{
			"Volume mounts for oauth with client secret and CA secret",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: "test",
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
								OAuth: auth.OAuth{
									Enabled: true,
									// Flags provided by user are changed by the operator to "correct" ones
									Parameters: []string{
										"--client-secret-name=bbbb",
										"--provider-ca-secret-name=2137",
									},
								},
							},
						},
					},
					IsRouteAPI: true,
				},
			},
			want{
				mocks.OAuthVolumeMountsCAClientSecret,
			},
		},
		{
			"Volume mounts for oauth with CA secret",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: "test",
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
								OAuth: auth.OAuth{
									Enabled: true,
									// Flags provided by user are changed by the operator to "correct" ones
									Parameters: []string{
										"--provider-ca-secret-name=wack",
									},
								},
							},
						},
					},
					IsRouteAPI: true,
				},
			},
			want{
				mocks.OAuthVolumeMountsCA,
			},
		},
		{
			"Volume mounts for oauth with client secret",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: "test",
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
								OAuth: auth.OAuth{
									Enabled: true,
									// Flags provided by user are changed by the operator to "correct" ones
									Parameters: []string{
										"--client-secret-name=crack",
										"--provider-ca-secret-name=szpak",
									},
								},
							},
						},
					},
					IsRouteAPI: true,
				},
			},
			want{
				mocks.OAuthVolumeMountsCAClientSecret,
			},
		},
		{
			"Volume mounts for oauth - wrong client secret flag",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: "test",
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
								OAuth: auth.OAuth{
									Enabled: true,
									// Flags provided by user are changed by the operator to "correct" ones
									Parameters: []string{
										"--client-secret-name-wrong=aaaa",
										"--provider-ca-secret-name=aaaa",
									},
								},
							},
						},
					},
					IsRouteAPI: true,
				},
			},
			want{
				mocks.OAuthVolumeMountsCA,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			volumeMounts := getAuthSecretVolumeMounts(tt.args.config)
			assert.ElementsMatch(t, tt.want.volumeMounts, volumeMounts)
		})
	}
}

func TestParseOauth2ProxyArgs(t *testing.T) {
	namespace := "test"
	consoleRoute := mocks.ConsoleRoute
	scheme := runtime.NewScheme()
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&consoleRoute).Build()

	type args struct {
		config IBMLicenseServiceReporterConfig
	}
	type want struct {
		params []string
	}
	tests := []struct {
		name string
		args args
		want want
	}{
		{
			"Basic auth params",
			args{
				IBMLicenseServiceReporterConfig{
					IsRouteAPI: true,
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							RouteEnabled: ptr.To(true),
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetBasicAuthOnlyParams("", "", []string{}),
			},
		},
		{
			"Basic auth params when ingress enabled",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
							},
							IngressEnabled: true,
							IngressOptions: &v1alpha1.IBMLicenseServiceReporterIngressOptions{
								CommonOptions: &v1alpha1.IBMLicenseServiceReporterIngressCommonOptions{
									Host: ptr.To("top-app-no-cap.com"),
								},
								ApiOptions: &v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions{
									Path: ptr.To("/api"),
								},
								ConsoleOptions: &v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions{
									Path: ptr.To("/console"),
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetBasicAuthOnlyParams("/license-service-reporter", "https://top-app-no-cap.com/license-service-reporter", []string{}),
			},
		},
		{
			"OAuth params",
			args{
				IBMLicenseServiceReporterConfig{
					IsRouteAPI: true,
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							RouteEnabled: ptr.To(true),
							Authentication: v1alpha1.Authentication{
								OAuth: auth.OAuth{
									Enabled: true,
									Parameters: []string{
										"--any-param-to-invoke-defaults=a",
									},
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetOAuthOnlyParams("", "", []string{"--any-param-to-invoke-defaults=a"}),
			},
		},
		{
			"OAuth params when ingress enabled",
			args{
				IBMLicenseServiceReporterConfig{
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							Authentication: v1alpha1.Authentication{
								OAuth: auth.OAuth{
									Enabled: true,
									Parameters: []string{
										"--any-param-to-invoke-defaults=a",
									},
								},
							},
							IngressEnabled: true,
							IngressOptions: &v1alpha1.IBMLicenseServiceReporterIngressOptions{
								CommonOptions: &v1alpha1.IBMLicenseServiceReporterIngressCommonOptions{
									Host: ptr.To("top-app-no-cap.com"),
								},
								ApiOptions: &v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions{
									Path: ptr.To("/api"),
								},
								ConsoleOptions: &v1alpha1.IBMLicenseServiceReporterIngressSpecificOptions{
									Path: ptr.To("/console"),
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetOAuthOnlyParams("/license-service-reporter", "https://top-app-no-cap.com/license-service-reporter", []string{"--any-param-to-invoke-defaults=a"}),
			},
		},
		{
			"Basic auth and oauth params",
			args{
				IBMLicenseServiceReporterConfig{
					IsRouteAPI: true,
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							RouteEnabled: ptr.To(true),
							Authentication: v1alpha1.Authentication{
								Useradmin: auth.Useradmin{
									Enabled: true,
								},
								OAuth: auth.OAuth{
									Enabled: true,
									Parameters: []string{
										"--any-param-to-invoke-defaults=a",
									},
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetBasicAuthOAuthParams([]string{"--any-param-to-invoke-defaults=a"}),
			},
		},
		{
			"OAuth - do not change default core params",
			args{
				IBMLicenseServiceReporterConfig{
					IsRouteAPI: true,
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							RouteEnabled: ptr.To(true),
							Authentication: v1alpha1.Authentication{
								OAuth: auth.OAuth{
									Enabled: true,
									Parameters: []string{
										"--https-address=http://dummy.com",
										"--proxy-prefix=/dummt",
										"--upstream=none",
										"--tls-cert-file=0",
										"--tls-key-file=0",
										"--htpasswd-file=/etc/hosts",
										"--display-htpasswd-form=false",
										"--custom-templates-dir=/",
										"--redirect-url=https://dummy.com",
									},
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetOAuthOnlyParams("", "", []string{}),
			},
		},
		{
			"OAuth - overwrite true default param (email-domain)",
			args{
				IBMLicenseServiceReporterConfig{
					IsRouteAPI: true,
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							RouteEnabled: ptr.To(true),
							Authentication: v1alpha1.Authentication{
								OAuth: auth.OAuth{
									Enabled: true,
									Parameters: []string{
										"--email-domain=test",
									},
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				func() []string {
					expectedParams := mocks.GetOAuthOnlyParams("", "", []string{})
					n := len(expectedParams)
					// by default there's --email-domain=*
					expectedParams[n-1] = "--email-domain=test"
					return expectedParams
				}(),
			},
		},
		{
			"OAuth - add custom secrets (CA and client-secret)",
			args{
				IBMLicenseServiceReporterConfig{
					IsRouteAPI: true,
					Instance: v1alpha1.IBMLicenseServiceReporter{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "instance",
							Namespace: namespace,
						},
						Spec: v1alpha1.IBMLicenseServiceReporterSpec{
							RouteEnabled: ptr.To(true),
							Authentication: v1alpha1.Authentication{
								OAuth: auth.OAuth{
									Enabled: true,
									Parameters: []string{
										"--client-secret-name=my-secret",
										"--provider-ca-secret-name=my-ca-secret-name",
									},
								},
							},
						},
					},
					Client: client,
				},
			},
			want{
				mocks.GetOAuthOnlyParams("", "", []string{"--client-secret-file=/opt/oauth2-proxy/config/client-secret",
					"--provider-ca-file=/opt/oauth2-proxy/config/provider-ca",
				}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params, err := parseOauth2ProxyArgs(tt.args.config)
			assert.NoError(t, err)
			assert.ElementsMatch(t, tt.want.params, params)
		})
	}
}

func TestGetCoreParams(t *testing.T) {
	expectedCoreParams := map[string]string{
		"--https-address":         "",
		"--proxy-prefix":          "",
		"--upstream":              "",
		"--tls-cert-file":         "",
		"--tls-key-file":          "",
		"--htpasswd-file":         "",
		"--display-htpasswd-form": "",
		"--custom-templates-dir":  "",
		"--redirect-url":          "",
		"--client-secret-file":    "",
		"--provider-ca-file":      "",
	}

	assert.Equal(t, expectedCoreParams, getCoreParams())
}

func TestGetExtSecretsMountPaths(t *testing.T) {
	expectedPaths := map[string]string{
		"--client-secret-name":      "--client-secret-file=/opt/oauth2-proxy/config/client-secret",
		"--provider-ca-secret-name": "--provider-ca-file=/opt/oauth2-proxy/config/provider-ca",
	}
	paths := getExtSecretsMountPaths()

	assert.Equal(t, expectedPaths, paths)
}

func TestGetTrueDefaultParams(t *testing.T) {
	expectedParams := map[string]string{
		"--email-domain": "*",
	}
	params := getTrueDefaultParams()

	assert.Equal(t, expectedParams, params)
}

func TestGetOAuthContainer(t *testing.T) {
	consoleRoute := mocks.ConsoleRoute
	scheme := runtime.NewScheme()
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&consoleRoute).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: v1alpha1.IBMLicenseServiceReporter{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "instance",
				Namespace: "test",
			},
			Spec: v1alpha1.IBMLicenseServiceReporterSpec{
				RouteEnabled: ptr.To(true),
				Authentication: v1alpha1.Authentication{
					Useradmin: auth.Useradmin{
						Enabled: true,
					},
				},
			},
		},
		Client:     client,
		Scheme:     scheme,
		IsRouteAPI: true,
	}

	t.Setenv("IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE", "icr.io/cpopen/ibm-lsr:test")
	authContainer, err := GetOAuthContainer(config)

	assert.NoError(t, err)

	cpuLimits := resource.NewMilliQuantity(100, resource.DecimalSI)
	assert.Equal(t, authContainer.Resources.Limits.Cpu(), cpuLimits)

	cpuRequests := cpuLimits
	assert.Equal(t, authContainer.Resources.Requests.Cpu(), cpuRequests)

	memoryLimits := resource.NewQuantity(50*1024*1024, resource.BinarySI)
	assert.Equal(t, authContainer.Resources.Limits.Memory(), memoryLimits)

	memoryRequests := memoryLimits
	assert.Equal(t, authContainer.Resources.Requests.Memory(), memoryRequests)

	assert.ElementsMatch(t, mocks.GetBasicAuthOnlyParams("", "", []string{}), authContainer.Args)

	assert.ElementsMatch(t, mocks.BasicAuthVolumeMounts, authContainer.VolumeMounts)

	assert.NotNil(t, authContainer.LivenessProbe)
	assert.NotNil(t, authContainer.ReadinessProbe)
}
