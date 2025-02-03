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

package controllers

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	api "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	res "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

// Check if updateStatus method creates LSR CR status section when LSR Pod exists
func TestUpdateStatus(t *testing.T) {
	namespace := "test-ns"
	opts := zap.Options{
		Development: true,
		TimeEncoder: zapcore.RFC3339TimeEncoder,
	}
	logger := zap.New(zap.UseFlagOptions(&opts))

	instance := api.IBMLicenseServiceReporter{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "instance",
			Namespace: namespace,
		},
	}

	instance.Labels = res.LabelsForMeta(instance)

	lsrPod := corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "license-service-reporter-instance-pod",
			Namespace: namespace,
			Labels:    res.LabelsForPod(instance),
		},
		Status: corev1.PodStatus{
			Phase:      corev1.PodRunning,
			Conditions: []corev1.PodCondition{},
		},
	}

	scheme := runtime.NewScheme()
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(api.AddToScheme(scheme))

	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&instance, &lsrPod).Build()
	reconciler := IBMLicenseServiceReporterReconciler{
		Client: client,
		Scheme: scheme,
	}

	assert.Empty(t, instance.Status.LicenseServiceReporterPods)

	reconciler.updateStatus(context.TODO(), logger, &instance)

	assert.NotEmpty(t, instance.Status.LicenseServiceReporterPods)
	// Check if pod was found and was added to the CR status section
	assert.Equal(t, corev1.PodRunning, instance.Status.LicenseServiceReporterPods[0].Phase)
}
