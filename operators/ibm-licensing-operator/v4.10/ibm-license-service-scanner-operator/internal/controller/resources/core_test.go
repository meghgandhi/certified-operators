/*
Copyright 2024 IBM Corporation

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"context"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/version"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createMockBaseReconcilableResource() *BaseReconcilableResource {
	sampleResource := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "TestName",
			Namespace: "TestNamespace",
		},
	}

	return &BaseReconcilableResource{
		ExpectedResource: sampleResource,
		ActualResource:   sampleResource.DeepCopy(),
		Config: ScannerConfig{
			Client:  fake.NewClientBuilder().Build(),
			Context: context.Background(),
		},
		Name:   "TestName",
		Logger: logr.Discard(),
	}
}

func createResourceWithLabelsAndAnnotations(
	expectedLabels, expectedAnnotations, actualLabels, actualAnnotations map[string]string,
) *BaseReconcilableResource {
	sample := createMockBaseReconcilableResource()

	sample.ExpectedResource.SetLabels(expectedLabels)
	sample.ExpectedResource.SetAnnotations(expectedAnnotations)
	sample.ActualResource.SetLabels(actualLabels)
	sample.ActualResource.SetAnnotations(actualAnnotations)

	return sample
}

/*
CheckInit should throw errors when a resource wasn't initialized correctly.
*/
func TestCheckInitThrowsErrors(t *testing.T) {
	testCases := map[string]struct {
		resource BaseReconcilableResource
		expect   string
	}{
		"Empty expected resource": {
			resource: BaseReconcilableResource{
				ExpectedResource: nil,
				Logger:           logr.Discard(),
			},
			expect: "expected resource is nil",
		},
		"Empty actual resource": {
			resource: BaseReconcilableResource{
				ExpectedResource: &corev1.Secret{},
				ActualResource:   nil,
				Logger:           logr.Discard(),
			},
			expect: "actual resource is nil",
		},
		"Empty Context": {
			resource: BaseReconcilableResource{
				ExpectedResource: &corev1.Secret{},
				ActualResource:   &corev1.Secret{},
				Config: ScannerConfig{
					Context: nil,
					Client:  nil,
				},
				Logger: logr.Discard(),
			},
			expect: "context is nil",
		},
		"Empty Client": {
			resource: BaseReconcilableResource{
				ExpectedResource: &corev1.Secret{},
				ActualResource:   &corev1.Secret{},
				Config: ScannerConfig{
					Context: context.Background(),
					Client:  nil,
				},
				Logger: logr.Discard(),
			},
			expect: "client is nil",
		},
		"Empty resource name": {
			resource: BaseReconcilableResource{
				ExpectedResource: &corev1.Secret{},
				ActualResource:   &corev1.Secret{},
				Config: ScannerConfig{
					Client:  fake.NewClientBuilder().Build(),
					Context: context.Background(),
				},
				Name:   "",
				Logger: logr.Discard(),
			},
			expect: "resource name is empty",
		},
		"Empty expected resource's name": {
			resource: BaseReconcilableResource{
				ExpectedResource: &corev1.Secret{},
				ActualResource:   &corev1.Secret{},
				Config: ScannerConfig{
					Client:  fake.NewClientBuilder().Build(),
					Context: context.Background(),
				},
				Name:   "NamedResource",
				Logger: logr.Discard(),
			},
			expect: "expected resource's name is empty",
		},
		"Empty expected resource's namespace": {
			resource: BaseReconcilableResource{
				ExpectedResource: &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						Name: "TestNamedResource",
					},
				},
				ActualResource: &corev1.Secret{},
				Config: ScannerConfig{
					Client:  fake.NewClientBuilder().Build(),
					Context: context.Background(),
				},
				Name:   "NamedResource",
				Logger: logr.Discard(),
			},
			expect: "expected resource's namespaces is empty",
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			err := test.resource.CheckInit()
			assert.Error(t, err)
			assert.Equal(t, test.expect, err.Error())
		})
		assert.True(t, ok)
	}
}

/*
CheckInit should return no error for correctly initialized resources.
*/
func TestCheckInitNoErrors(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	// Override actual resource as empty to mimic the actual codebase better
	resource.ActualResource = &corev1.Secret{}

	err := resource.CheckInit()
	assert.NoError(t, err)
}

/*
MarkShouldCreate should mark ShouldCreate status field as true by default.
*/
func TestMarkShouldCreate(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	err := resource.MarkShouldCreate()
	assert.NoError(t, err)
	assert.Equal(t, true, resource.status.ShouldCreate)
}

/*
MarkShouldUpdate should mark ShouldUpdate status field as false by default.
*/
func TestMarkShouldUpdate(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	err := resource.MarkShouldUpdate()
	assert.NoError(t, err)
	assert.Equal(t, false, resource.status.ShouldUpdate)
}

/*
GetResource should return without an error if resource was found.
*/
func TestGetResourceNoErrors(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	// Mark resource as existing by replacing the client with one that builds it
	resource.Config.Client = fake.NewClientBuilder().WithObjects(resource.ExpectedResource).Build()
	err := resource.GetResource()
	assert.NoError(t, err)
}

/*
GetResource should throw error if resource is not found.
*/
func TestGetResourceNotFound(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	err := resource.GetResource()

	assert.True(t, apierrors.IsNotFound(err))
	assert.Equal(t, "secrets \"TestName\" not found", err.Error())
}

/*
LabelsEqual should be true iff all expected resource's labels are matching the actual one's.
*/
func TestLabelsEqual(t *testing.T) {
	testCases := map[string]struct {
		resource *BaseReconcilableResource
		expect   bool
	}{
		"Exact match": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test": "test"},
				nil,
			),
			expect: true,
		},
		"Labels missing from actual": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test"},
				nil,
				nil,
				nil,
			),
			expect: false,
		},
		"Extra labels in expected": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test", "test-extra": "test"},
				nil,
				map[string]string{"test": "test"},
				nil,
			),
			expect: false,
		},
		"Extra labels in actual": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test": "test", "test-extra": "test"},
				nil,
			),
			expect: true,
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expect, test.resource.labelsEqual())
		})
		assert.True(t, ok)
	}
}

/*
AnnotationsEqual should be true iff all expected resource's annotations are matching the actual one's.
*/
func TestAnnotationsEqual(t *testing.T) {
	testCases := map[string]struct {
		resource *BaseReconcilableResource
		expect   bool
	}{
		"Exact match": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test": "test"},
			),
			expect: true,
		},
		"Annotations missing from actual": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test"},
				nil,
				nil,
			),
			expect: false,
		},
		"Extra annotations in expected": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test", "test-extra": "test"},
				nil,
				map[string]string{"test": "test"},
			),
			expect: false,
		},
		"Extra annotations in actual": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test": "test", "test-extra": "test"},
			),
			expect: true,
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expect, test.resource.annotationsEqual())
		})
		assert.True(t, ok)
	}
}

/*
Base labels should be populated from spec + predefined.
*/
func TestGetBaseLabels(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	resource.Config.Scanner.Spec.Labels = map[string]string{"test": "test"}

	labels := resource.GetBaseLabels()
	assert.Equal(t, "test", labels["test"])
	assert.Equal(t, applicationName, labels["app.kubernetes.io/name"])
	assert.Equal(t, version.Version, labels["app.kubernetes.io/version"])
	assert.Equal(t, "TestName", labels["app.kubernetes.io/component"])
	assert.Equal(t, operatorResourceName, labels["app.kubernetes.io/managed-by"])
}

/*
Base annotations should be populated from spec + predefined.
*/
func TestGetBaseAnnotations(t *testing.T) {
	resource := createMockBaseReconcilableResource()
	resource.Config.Scanner.Spec.Annotations = map[string]string{"test": "test"}

	annotations := resource.GetBaseAnnotations()
	assert.Equal(t, "test", annotations["test"])
	assert.Equal(t, annotationProductID, annotations["productID"])
	assert.Equal(t, annotationProductName, annotations["productName"])
	assert.Equal(t, annotationProductMetric, annotations["productMetric"])
}

/*
Check labels from a cluster resource get added to the expected one without overriding already existing labels.
*/
func TestPopulateWithLabels(t *testing.T) {
	testCases := map[string]struct {
		resource *BaseReconcilableResource
		expect   map[string]string
	}{
		"No actual labels": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test"},
				nil,
				nil,
				nil,
			),
			expect: map[string]string{"test": "test"},
		},
		"Extra actual label": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test-new": "test"},
				nil,
			),
			expect: map[string]string{"test": "test", "test-new": "test"},
		},
		"Already existing actual label key": {
			resource: createResourceWithLabelsAndAnnotations(
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test": "test-new"},
				nil,
			),
			expect: map[string]string{"test": "test"},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			test.resource.populateExpectedWithActualLabels()
			assert.Equal(t, test.expect, test.resource.ExpectedResource.GetLabels())
		})
		assert.True(t, ok)
	}
}

/*
Check annotations from a cluster resource get added to the expected one without overriding already existing annotations.
*/
func TestPopulateWithAnnotations(t *testing.T) {
	testCases := map[string]struct {
		resource *BaseReconcilableResource
		expect   map[string]string
	}{
		"No actual annotations": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test"},
				nil,
				nil,
			),
			expect: map[string]string{"test": "test"},
		},
		"Extra actual annotation": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test-new": "test"},
			),
			expect: map[string]string{"test": "test", "test-new": "test"},
		},
		"Already existing actual annotation key": {
			resource: createResourceWithLabelsAndAnnotations(
				nil,
				map[string]string{"test": "test"},
				nil,
				map[string]string{"test": "test-new"},
			),
			expect: map[string]string{"test": "test"},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			test.resource.populateExpectedWithActualAnnotations()
			assert.Equal(t, test.expect, test.resource.ExpectedResource.GetAnnotations())
		})
		assert.True(t, ok)
	}
}
