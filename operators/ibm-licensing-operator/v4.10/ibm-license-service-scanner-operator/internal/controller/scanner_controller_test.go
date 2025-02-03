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

package controller

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/stretchr/testify/assert"
	operatorv1 "github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/api/v1"
)

/*
License acceptance should support true/false and not being set.
*/
func TestLicenseAcceptance(t *testing.T) {
	testCases := map[string]struct {
		instance operatorv1.IBMLicenseServiceScanner
		expected bool
	}{
		"License acceptance is set to true": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					License: &operatorv1.License{
						Accept: true,
					},
				},
			},
			expected: true,
		},
		"License acceptance is set to false": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					License: &operatorv1.License{
						Accept: false,
					},
				},
			},
			expected: false,
		},
		"License acceptance is not set": {
			instance: operatorv1.IBMLicenseServiceScanner{},
			expected: false,
		},
	}

	var instance operatorv1.IBMLicenseServiceScanner
	for name, test := range testCases {
		instance = test.instance
		ok := t.Run(name, func(t *testing.T) {
			assert.Equal(t, test.expected, isLicenseAccepted(&instance))
		})
		assert.True(t, ok)
	}
}

/*
setScanFrequencyIfEmpty should set default scan frequency if not set.
*/
func TestSetScanFrequencyIfEmpty(t *testing.T) {
	testCases := map[string]struct {
		instance operatorv1.IBMLicenseServiceScanner
		expected string
	}{
		"Frequency is empty": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					Scan: operatorv1.IBMLicenseServiceScannerConfig{
						Frequency: "",
					},
				},
			},
			expected: defaultScanFrequency,
		},
		"Frequency is not empty": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					Scan: operatorv1.IBMLicenseServiceScannerConfig{
						Frequency: "1 1 * * *",
					},
				},
			},
			expected: "1 1 * * *",
		},
	}

	var instance operatorv1.IBMLicenseServiceScanner
	for name, test := range testCases {
		instance = test.instance
		ok := t.Run(name, func(t *testing.T) {
			setScanFrequencyIfEmpty(&instance)
			assert.Equal(t, test.expected, instance.Spec.Scan.Frequency)
		})
		assert.True(t, ok)
	}
}

/*
setContainerIfEmpty should set default container values if not set.
*/
func TestSetContainerIfEmpty(t *testing.T) {
	testCases := map[string]struct {
		instance operatorv1.IBMLicenseServiceScanner
		expected *operatorv1.Container
	}{
		"Container is not provided": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{},
			},
			expected: &operatorv1.Container{
				Resources: operatorv1.ResourceRequirementsNoClaims{
					Requests: getDefaultContainerResourceRequests(),
					Limits:   getDefaultContainerResourceLimits(),
				},
				ImagePullPolicy: defaultImagePullPolicy,
			},
		},
		"Requests provided": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					Container: &operatorv1.Container{
						Resources: operatorv1.ResourceRequirementsNoClaims{
							Requests: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("500Mi"),
							},
						},
					},
				},
			},
			expected: &operatorv1.Container{
				Resources: operatorv1.ResourceRequirementsNoClaims{
					Requests: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("500Mi"),
					},
					Limits: getDefaultContainerResourceLimits(),
				},
				ImagePullPolicy: defaultImagePullPolicy,
			},
		},
		"Limits provided": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					Container: &operatorv1.Container{
						Resources: operatorv1.ResourceRequirementsNoClaims{
							Limits: corev1.ResourceList{
								corev1.ResourceCPU: resource.MustParse("500Mi"),
							},
						},
					},
				},
			},
			expected: &operatorv1.Container{
				Resources: operatorv1.ResourceRequirementsNoClaims{
					Requests: getDefaultContainerResourceRequests(),
					Limits: corev1.ResourceList{
						corev1.ResourceCPU: resource.MustParse("500Mi"),
					},
				},
				ImagePullPolicy: defaultImagePullPolicy,
			},
		},
		"Image pull policy provided": {
			instance: operatorv1.IBMLicenseServiceScanner{
				Spec: operatorv1.IBMLicenseServiceScannerSpec{
					Container: &operatorv1.Container{
						ImagePullPolicy: "Always",
					},
				},
			},
			expected: &operatorv1.Container{
				Resources: operatorv1.ResourceRequirementsNoClaims{
					Requests: getDefaultContainerResourceRequests(),
					Limits:   getDefaultContainerResourceLimits(),
				},
				ImagePullPolicy: "Always",
			},
		},
	}

	var instance operatorv1.IBMLicenseServiceScanner
	for name, test := range testCases {
		instance = test.instance
		ok := t.Run(name, func(t *testing.T) {
			setContainerIfEmpty(&instance)
			assert.Equal(t, test.expected, instance.Spec.Container)
		})
		assert.True(t, ok)
	}
}
