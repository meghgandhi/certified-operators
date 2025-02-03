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
	"context"
	"github.com/stretchr/testify/assert"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	"k8s.io/apimachinery/pkg/util/intstr"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
)

func TestIsServiceInDesiredState(t *testing.T) {
	// Test cases
	expected := corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{"key": "value"},
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{Name: "port1", Port: 8080, TargetPort: intstr.Parse("8080"), Protocol: corev1.ProtocolTCP},
				{Name: "port2", Port: 8081, TargetPort: intstr.Parse("8082"), Protocol: corev1.ProtocolUDP},
			},
			Selector: map[string]string{"app": "example", "app2": "example2"},
		},
	}
	tests := []struct {
		name     string
		found    *corev1.Service
		expected *corev1.Service
		want     bool
	}{
		{
			name:     "Equal services",
			found:    expected.DeepCopy(),
			expected: expected.DeepCopy(),
			want:     true,
		},
		{
			name: "Found service has more annotations",
			found: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"onlyFoundKey": "onlyFoundVal", "key": "value"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{Name: "port1", Port: 8080, TargetPort: intstr.Parse("8080"), Protocol: corev1.ProtocolTCP},
						{Name: "port2", Port: 8081, TargetPort: intstr.Parse("8082"), Protocol: corev1.ProtocolUDP},
					},
					Selector: map[string]string{"app": "example", "app2": "example2"},
				},
			},
			expected: expected.DeepCopy(),
			want:     true,
		},
		{
			name: "Found service has selector without all needed keys",
			found: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{Name: "port1", Port: 8080, TargetPort: intstr.Parse("8080"), Protocol: corev1.ProtocolTCP},
						{Name: "port2", Port: 8081, TargetPort: intstr.Parse("8082"), Protocol: corev1.ProtocolUDP},
					},
					Selector: map[string]string{"app": "example"},
				},
			},
			expected: expected.DeepCopy(),
			want:     false,
		},
		{
			name: "Found service has different order of ports with additional fields", //TODO: is this expected behavior?
			found: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{Name: "port2", Port: 8081, TargetPort: intstr.Parse("8082"), Protocol: corev1.ProtocolUDP},
						{
							Name:       "port1",
							Protocol:   corev1.ProtocolTCP,
							Port:       8080,
							TargetPort: intstr.Parse("8080"),
							NodePort:   8080,
						},
					},
					Selector: map[string]string{"app": "example", "app2": "example2"},
				},
			},
			expected: expected.DeepCopy(),
			want:     true,
		},
		{
			name: "Found service has different protocol in ports",
			found: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeClusterIP,
					Ports: []corev1.ServicePort{
						{Name: "port2", Port: 8081, TargetPort: intstr.Parse("8082"), Protocol: corev1.ProtocolSCTP},
						{Name: "port1", Port: 8080, TargetPort: intstr.Parse("8080"), Protocol: corev1.ProtocolTCP},
					},
					Selector: map[string]string{"app": "example", "app2": "example2"},
				},
			},
			expected: expected.DeepCopy(),
			want:     false,
		},
		{
			name: "Found service has different type",
			found: &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{"key": "value"},
				},
				Spec: corev1.ServiceSpec{
					Type: corev1.ServiceTypeNodePort,
					Ports: []corev1.ServicePort{
						{Name: "port1", Port: 8080, TargetPort: intstr.Parse("8080"), Protocol: corev1.ProtocolTCP},
						{Name: "port2", Port: 8081, TargetPort: intstr.Parse("8082"), Protocol: corev1.ProtocolUDP},
					},
					Selector: map[string]string{"app": "example", "app2": "example2"},
				},
			},
			expected: expected.DeepCopy(),
			want:     false,
		},
	}

	// Required for IsServiceInDesiredState check
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	reporter := v1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	client := fake.NewClientBuilder().Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, _ := IsServiceInDesiredState(config, tt.found, tt.expected, logger)
			if got.IsInDesiredState != tt.want {
				t.Errorf("IsServiceInDesiredState() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestReconcileServiceAppliesSpecLabels(t *testing.T) {
	namespace := "test"
	instanceName := "instance"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))
	reporter := v1alpha1.IBMLicenseServiceReporter{}
	reporter.Namespace = namespace
	reporter.Name = instanceName
	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	// Reconcile service should apply spec.labels via creation
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value"}
	err := ReconcileService(logger, config)
	assert.NoError(t, err)

	// Check service created with spec.labels
	foundService := corev1.Service{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: LicenseReporterResourceBase}, &foundService)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", foundService.GetLabels()["test-label"], foundService.GetName()+" should have spec.labels applied")

	// Reconcile service should apply spec.labels via patch
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value-patched"}
	err = ReconcileService(logger, config)
	assert.NoError(t, err)

	// Check service patched with spec.labels
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: LicenseReporterResourceBase}, &foundService)
	assert.NoError(t, err)
	assert.Equal(t, "test-value-patched", foundService.GetLabels()["test-label"], foundService.GetName()+" should have spec.labels applied")
}
