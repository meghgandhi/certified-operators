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
	"fmt"
	"strings"
	"testing"

	"github.com/go-logr/logr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	v1 "github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func createMockScannerCronJob(expectedResource, actualResource client.Object) *ScannerCronJob {
	return &ScannerCronJob{
		BaseReconcilableResource: BaseReconcilableResource{
			ExpectedResource: expectedResource,
			ActualResource:   actualResource,
			Config: ScannerConfig{
				Client:  fake.NewClientBuilder().Build(),
				Context: context.Background(),
			},
			Name:   "TestName",
			Logger: logr.Discard(),
		},
	}
}

func createMockCronJob(name, schedule string, containers []corev1.Container) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Spec: batchv1.CronJobSpec{
			Schedule: schedule,
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: containers,
						},
					},
				},
			},
		},
	}
}

func createMockCronJobWithLabelsAndAnnotations(labels, annotations map[string]string) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:        "test-cron-job",
			Labels:      labels,
			Annotations: annotations,
		},
	}
}

func createMockCronJobWithVaultAuth(initContainer []corev1.Container) *batchv1.CronJob {
	return &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name: "TestName",
		},
		Spec: batchv1.CronJobSpec{
			Schedule: "10 0 * * *",
			JobTemplate: batchv1.JobTemplateSpec{
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						Spec: corev1.PodSpec{
							Containers: []corev1.Container{{
								Name: "ContainerName",
							}},
							InitContainers: initContainer,
						},
					},
				},
			},
		},
	}
}

/*
MarkShouldUpdate for CronJob should mark resource ShouldUpdate status if there are differences in schedule or
containers.
*/
func TestMarkShouldUpdate_CronJob_SingleContainer(t *testing.T) {
	expectedResource := createMockCronJob("TestName", "10 0 * * *", []corev1.Container{{
		Name: "ContainerName",
	}})

	testCases := map[string]struct {
		resource *batchv1.CronJob
		expect   ReconcileStatus
	}{
		"CronJob with the same Schedule and Container Name": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{{
				Name: "ContainerName",
			}}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"CronJob with different Schedule and the same Container Name": {
			resource: createMockCronJob("TestName", "10 * * * *", []corev1.Container{{
				Name: "ContainerName",
			}}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with the same Schedule but different Container Name": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{{
				Name: "OtherContainerName",
			}}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with both Schedule and Container Name different": {
			resource: createMockCronJob("TestName", "10 * * * *", []corev1.Container{{
				Name: "OtherContainerName",
			}}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with the same Schedule and Container but different CronJob Name": {
			resource: createMockCronJob("OtherTestName", "10 0 * * *", []corev1.Container{{
				Name: "ContainerName",
			}}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"CronJob with a Container with different Env": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "FirstContainerName",
					Env: []corev1.EnvVar{
						{
							Name:  logLevelEnvVarName,
							Value: "DEBUG",
						},
						{
							Name:  scanNamespacesEnvVarName,
							Value: "namespace",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: registryPullSecretVolumeMountPath,
							ReadOnly:  true,
						},
					},
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with a Container with different VolumeMounts": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "FirstContainerName",
					Env: []corev1.EnvVar{
						{
							Name:  logLevelEnvVarName,
							Value: "INFO",
						},
						{
							Name:  scanNamespacesEnvVarName,
							Value: "namespace",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "other-volume",
							MountPath: "other-path",
							ReadOnly:  true,
						},
					},
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with a Container with both Env and VolumeMounts different": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "SomeOtherContainerName",
					Env: []corev1.EnvVar{
						{
							Name:  logLevelEnvVarName,
							Value: "INFO",
						},
						{
							Name:  scanNamespacesEnvVarName,
							Value: "other-namespace",
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name:      "volume",
							MountPath: "other-path",
							ReadOnly:  false,
						},
					},
				},
			}),

			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			reconcilableCronJob := createMockScannerCronJob(expectedResource, test.resource)
			err := reconcilableCronJob.MarkShouldUpdate()

			assert.Equal(t, reconcilableCronJob.status.ShouldCreate, test.expect.ShouldCreate)
			assert.Equal(t, reconcilableCronJob.status.ShouldUpdate, test.expect.ShouldUpdate)
			assert.NoError(t, err)
		})
		assert.True(t, ok)
	}
}

func TestMarkShouldUpdate_CronJob_MultipleContainers(t *testing.T) {
	expectedResource := createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
		{
			Name: "FirstContainerName",
		},
		{
			Name: "SecondContainerName",
		},
	})

	testCases := map[string]struct {
		resource *batchv1.CronJob
		expect   ReconcileStatus
	}{
		"CronJob with all the same Containers": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "FirstContainerName",
				},
				{
					Name: "SecondContainerName",
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"CronJob with one of the Containers different": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "FirstContainerName",
				},
				{
					Name: "OtherContainerName",
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with all different Containers": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "SomeOtherContainerName",
				},
				{
					Name: "YetAnotherContainerName",
				},
			}),

			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob missing one of the Containers": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "FirstContainerName",
				},
			}),

			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"CronJob with one Container more": {
			resource: createMockCronJob("TestName", "10 0 * * *", []corev1.Container{
				{
					Name: "FirstContainerName",
				},
				{
					Name: "SecondContainerName",
				},
				{
					Name: "ThirdContainerName",
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			reconcilableCronJob := createMockScannerCronJob(expectedResource, test.resource)

			err := reconcilableCronJob.MarkShouldUpdate()

			assert.Equal(t, reconcilableCronJob.status.ShouldCreate, test.expect.ShouldCreate)
			assert.Equal(t, reconcilableCronJob.status.ShouldUpdate, test.expect.ShouldUpdate)
			assert.NoError(t, err)
		})
		assert.True(t, ok)
	}
}

func TestMarkShouldUpdate_CronJob_LabelsAndAnnotations(t *testing.T) {
	expectedResource := createMockCronJobWithLabelsAndAnnotations(
		map[string]string{"test-label": "test"},
		map[string]string{"test-annotation": "test"},
	)

	testCases := map[string]struct {
		resource *batchv1.CronJob
		expect   ReconcileStatus
	}{
		"Missing both labels and annotations": {
			resource: createMockCronJobWithLabelsAndAnnotations(nil, nil),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Missing labels only": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				nil,
				map[string]string{"test-annotation": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Missing annotations only": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test"},
				nil,
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Labels and annotations match": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test"},
				map[string]string{"test-annotation": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"Labels value mismatch": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test2"},
				map[string]string{"test-annotation": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Annotations value mismatch": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test"},
				map[string]string{"test-annotation": "test2"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Extra label present": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test", "test-label-2": "test"},
				map[string]string{"test-annotation": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"Extra annotation present": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test"},
				map[string]string{"test-annotation": "test", "test-annotation-2": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"Extra label present but base label missing": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label-2": "test"},
				map[string]string{"test-annotation": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Extra annotation present but base annotation missing": {
			resource: createMockCronJobWithLabelsAndAnnotations(
				map[string]string{"test-label": "test"},
				map[string]string{"test-annotation-2": "test"},
			),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			reconcilableCronJob := createMockScannerCronJob(expectedResource, test.resource)
			err := reconcilableCronJob.MarkShouldUpdate()

			assert.Equal(t, reconcilableCronJob.status.ShouldCreate, test.expect.ShouldCreate)
			assert.Equal(t, reconcilableCronJob.status.ShouldUpdate, test.expect.ShouldUpdate)
			assert.NoError(t, err)
		})
		assert.True(t, ok)
	}
}

func TestMarkShouldUpdate_CronJob_VaultAuthentication(t *testing.T) {
	// Create reference InitContainer
	expectedResource := createMockCronJobWithVaultAuth([]corev1.Container{
		{
			Name: "TestContainer",
			Env: []corev1.EnvVar{
				{
					Name:  "ROLE_0",
					Value: "role",
				},
				{
					Name:  "CERT_PATH_0",
					Value: "/opt/scanner/vaults/cert/ca.crt",
				},
				{
					Name:  "REGISTRY_NAME_0",
					Value: "icr.io",
				},
				{
					Name:  "REGISTRY_USER_0",
					Value: "iamapikey",
				},
				{
					Name:  "SA_TOKEN_0",
					Value: "/opt/scanner/vaults/sa/token",
				},
				{
					Name:  "SECRET_KEY_0",
					Value: "api_key",
				},
				{
					Name:  "VAULT_SECRET_ADDR_0",
					Value: "https:vault.com/v1/kv/data/secret",
				},
				{
					Name:  "VAULT_SECRET_ADDR_0",
					Value: "https:vault.com/v1/auth/kubernetes/login",
				},
			},
		},
	})

	testCases := map[string]struct {
		resource *batchv1.CronJob
		expect   ReconcileStatus
	}{
		"Cron job with the same Init Container": {
			resource: createMockCronJobWithVaultAuth([]corev1.Container{
				{
					Name: "TestContainer",
					Env: []corev1.EnvVar{
						{
							Name:  "ROLE_0",
							Value: "role",
						},
						{
							Name:  "CERT_PATH_0",
							Value: "/opt/scanner/vaults/cert/ca.crt",
						},
						{
							Name:  "REGISTRY_NAME_0",
							Value: "icr.io",
						},
						{
							Name:  "REGISTRY_USER_0",
							Value: "iamapikey",
						},
						{
							Name:  "SA_TOKEN_0",
							Value: "/opt/scanner/vaults/sa/token",
						},
						{
							Name:  "SECRET_KEY_0",
							Value: "api_key",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/kv/data/secret",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/auth/kubernetes/login",
						},
					},
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: false,
			},
		},
		"Other environment variables values": {
			resource: createMockCronJobWithVaultAuth([]corev1.Container{
				{
					Name: "TestContainer",
					Env: []corev1.EnvVar{
						{
							Name:  "ROLE_0",
							Value: "other_role",
						},
						{
							Name:  "CERT_PATH_0",
							Value: "/opt/scanner/vaults/cert/ca.crt",
						},
						{
							Name:  "REGISTRY_NAME_0",
							Value: "docker.io",
						},
						{
							Name:  "REGISTRY_USER_0",
							Value: "kkey",
						},
						{
							Name:  "SA_TOKEN_0",
							Value: "/opt/scanner/vaults/sa/token",
						},
						{
							Name:  "SECRET_KEY_0",
							Value: "api_key",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/kv/data/secret",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/auth/kubernetes/login",
						},
					},
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Missing variables": {
			resource: createMockCronJobWithVaultAuth([]corev1.Container{
				{
					Name: "TestContainer",
					Env: []corev1.EnvVar{
						{
							Name:  "ROLE_0",
							Value: "role",
						},
						{
							Name:  "CERT_PATH_0",
							Value: "/opt/scanner/vaults/cert/ca.crt",
						},
						{
							Name:  "REGISTRY_NAME_0",
							Value: "icr.io",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/kv/data/secret",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/auth/kubernetes/login",
						},
					},
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"Second set of variables": {
			resource: createMockCronJobWithVaultAuth([]corev1.Container{
				{
					Name: "TestContainer",
					Env: []corev1.EnvVar{
						{
							Name:  "ROLE_0",
							Value: "role",
						},
						{
							Name:  "CERT_PATH_0",
							Value: "/opt/scanner/vaults/cert/ca.crt",
						},
						{
							Name:  "REGISTRY_NAME_0",
							Value: "icr.io",
						},
						{
							Name:  "REGISTRY_USER_0",
							Value: "iamapikey",
						},
						{
							Name:  "SA_TOKEN_0",
							Value: "/opt/scanner/vaults/sa/token",
						},
						{
							Name:  "SECRET_KEY_0",
							Value: "api_key",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/kv/data/secret",
						},
						{
							Name:  "VAULT_SECRET_ADDR_0",
							Value: "https:vault.com/v1/auth/kubernetes/login",
						},
						{
							Name:  "ROLE_1",
							Value: "other_role",
						},
						{
							Name:  "CERT_PATH_1",
							Value: "/opt/scanner/vaults/cert2/ca.crt",
						},
						{
							Name:  "REGISTRY_NAME_1",
							Value: "quay.io",
						},
						{
							Name:  "REGISTRY_USER_1",
							Value: "iamapikey",
						},
						{
							Name:  "SA_TOKEN_1",
							Value: "/opt/scanner/vaults/sa2/token",
						},
						{
							Name:  "SECRET_KEY_1",
							Value: "other_api_key",
						},
						{
							Name:  "VAULT_SECRET_ADDR_1",
							Value: "https://vault.com/v1/kv/data/secret/secret2",
						},
						{
							Name:  "VAULT_SECRET_ADDR_1",
							Value: "https://vault.com/v1/auth/kubernetes/login",
						},
					},
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
		"No variables set": {
			resource: createMockCronJobWithVaultAuth([]corev1.Container{
				{
					Name: "TestContainer",
				},
			}),
			expect: ReconcileStatus{
				ShouldCreate: false,
				ShouldUpdate: true,
			},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			reconcilableCronJob := createMockScannerCronJob(expectedResource, test.resource)
			err := reconcilableCronJob.MarkShouldUpdate()

			assert.Equal(t, reconcilableCronJob.status.ShouldCreate, test.expect.ShouldCreate)
			assert.Equal(t, reconcilableCronJob.status.ShouldUpdate, test.expect.ShouldUpdate)
			assert.NoError(t, err)
		})
		assert.True(t, ok)
	}
}

// Assisted by WCA for GP
// Latest GenAI contribution: granite-20B-code-instruct-v2 model
func TestGetScanNamespaces_AlphabeticallyOrdered(t *testing.T) {
	testCases := []struct {
		name         string
		crNamespaces []string
		expected     []string // elements are expected to be alphabetically ordered
	}{
		{
			name:         "Certain namespaces",
			crNamespaces: []string{"my-ns", "default", "kube-system"},
			expected:     []string{"default", "kube-system", "my-ns"},
		},
		{
			name:         "All namespaces wildcard",
			crNamespaces: []string{"*"},
			expected:     []string{"default", "kube-system", "my-ns", "my-ns-2", "ns-1", "ns-2", "ns-3"},
		},
		{
			name:         "Single prefix wildcard",
			crNamespaces: []string{"ns-*"},
			expected:     []string{"ns-1", "ns-2", "ns-3"},
		},
		{
			name:         "Prefix wildcard mixed with plain namespaces",
			crNamespaces: []string{"ns-*", "default"},
			expected:     []string{"default", "ns-1", "ns-2", "ns-3"},
		},
		{
			name:         "Multiple prefix wildcards",
			crNamespaces: []string{"ns-*", "my-*"},
			expected:     []string{"my-ns", "my-ns-2", "ns-1", "ns-2", "ns-3"},
		},
		{
			name:         "Multiple prefix wildcard mixed with plain namespaces",
			crNamespaces: []string{"ns-*", "default", "my-*"},
			expected:     []string{"default", "my-ns", "my-ns-2", "ns-1", "ns-2", "ns-3"},
		},
		// Malicious regexps
		{
			name:         "Multiple prefix wildcard mixed with plain namespaces",
			crNamespaces: []string{"ns-[a-z]+*"},
			expected:     nil,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j := ScannerCronJob{
				BaseReconcilableResource: BaseReconcilableResource{
					Config: ScannerConfig{
						Scanner: v1.IBMLicenseServiceScanner{
							Spec: v1.IBMLicenseServiceScannerSpec{
								Scan: v1.IBMLicenseServiceScannerConfig{
									Namespaces: tc.crNamespaces,
								},
							},
						},
						Client: getFakeClientWithNamespaces(),
					},
				},
			}
			actual, err := j.getScanNamespaces()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual) // order of elements MUST be deterministic
		})
	}
}

func TestGetScanNamespaces_Errors(t *testing.T) {
	testCases := []struct {
		name         string
		crNamespaces []string
		expectedErr  error // elements are expected to be alphabetically ordered
	}{
		{
			name:         "Suffix wildcard",
			crNamespaces: []string{"*-2"},
			expectedErr:  fmt.Errorf("unsupported wildcard detected: *-2"),
		},
		{
			name:         "Infix wildcard",
			crNamespaces: []string{"ns-*-2"},
			expectedErr:  fmt.Errorf("unsupported wildcard detected: ns-*-2"),
		},
		{
			name:         "Multiple asterisks wildcard",
			crNamespaces: []string{"ns-*-2*"},
			expectedErr:  fmt.Errorf("unsupported wildcard detected: ns-*-2*"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j := ScannerCronJob{
				BaseReconcilableResource: BaseReconcilableResource{
					Config: ScannerConfig{
						Scanner: v1.IBMLicenseServiceScanner{
							Spec: v1.IBMLicenseServiceScannerSpec{
								Scan: v1.IBMLicenseServiceScannerConfig{
									Namespaces: tc.crNamespaces,
								},
							},
						},
						Client: getFakeClientWithNamespaces(),
					},
				},
			}
			res, err := j.getScanNamespaces()
			require.Error(t, err)
			assert.Equal(t, []string{}, res)
			strings.Contains(err.Error(), tc.expectedErr.Error())
		})
	}
}

func getFakeClientWithNamespaces() client.Client {
	return fake.NewClientBuilder().WithObjects(
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-1",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-2",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ns",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "my-ns-2",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "ns-3",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "default",
			},
		},
		&corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: "kube-system",
			},
		},
	).Build()
}

func TestImagePull_Prefix(t *testing.T) {
	testCases := []struct {
		name     string
		prefix   string
		expected string
	}{
		{
			name:     "Empty-registry",
			prefix:   "",
			expected: "not.icr/image/path@some-digest",
		},
		{
			name:     "ICR",
			prefix:   "icr.io",
			expected: "icr.io/image/path@some-digest",
		},
		{
			name:     "Other registry",
			prefix:   "test.docker",
			expected: "test.docker/image/path@some-digest",
		},
	}

	// Required to j.getContainers
	t.Setenv("IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE", "not.icr/image/path@some-digest")

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j := ScannerCronJob{
				BaseReconcilableResource: BaseReconcilableResource{
					Config: ScannerConfig{
						Scanner: v1.IBMLicenseServiceScanner{
							Spec: v1.IBMLicenseServiceScannerSpec{
								Container: &v1.Container{
									ImagePullPrefix: tc.prefix,
								},
								Scan: v1.IBMLicenseServiceScannerConfig{
									Namespaces: []string{"fake-test"},
								},
							},
						},
						Client: getFakeClientWithNamespaces(),
					},
				},
			}
			actual, err := j.getContainers()
			require.NoError(t, err)
			assert.Equal(t, tc.expected, actual[0].Image)
		})
	}
}

func TestImagePull_Secrets(t *testing.T) {
	testCases := []struct {
		name     string
		secrets  []string
		expected []corev1.LocalObjectReference
	}{
		{
			name:     "No secrets",
			secrets:  nil,
			expected: nil,
		},
		{
			name:     "Sample secrets",
			secrets:  []string{"sample-secret-1", "sample-secret-2"},
			expected: []corev1.LocalObjectReference{{Name: "sample-secret-1"}, {Name: "sample-secret-2"}},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			j := ScannerCronJob{
				BaseReconcilableResource: BaseReconcilableResource{
					Config: ScannerConfig{
						Scanner: v1.IBMLicenseServiceScanner{
							Spec: v1.IBMLicenseServiceScannerSpec{
								Container: &v1.Container{
									ImagePullSecrets: tc.secrets,
								},
								Scan: v1.IBMLicenseServiceScannerConfig{
									Namespaces: []string{"fake-test"},
								},
							},
						},
						Client: getFakeClientWithNamespaces(),
					},
				},
			}
			actual := j.getImagePullSecrets()
			assert.Equal(t, tc.expected, actual)
		})
	}
}
