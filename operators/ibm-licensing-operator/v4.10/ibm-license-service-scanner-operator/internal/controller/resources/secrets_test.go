package resources

import (
	"context"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"testing"
)

const (
	namespace         = "test-namespace"
	testToken         = "test-token"
	testURL           = "test-url"
	testCert          = "test-cert"
	testExistingToken = "test-existing-token"
)

/*
Create resources which would normally be copied by the operand request. Then build the client with them.
*/
func buildMockConfig() *ScannerConfig {
	licenseServiceSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      licenseServiceUploadTokenName,
			Namespace: namespace,
		},
		Data: map[string][]byte{
			"token-upload": []byte(testToken),
		},
	}
	licenseServiceConfig := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      licenseServiceUploadConfigName,
			Namespace: namespace,
		},
		Data: map[string]string{
			"url":     testURL,
			"crt.pem": testCert,
		},
	}

	return &ScannerConfig{
		Client:  fake.NewClientBuilder().WithObjects(licenseServiceSecret, licenseServiceConfig).Build(),
		Context: context.Background(),
	}
}

func createMockSecretWithOperandRequest(
	secretData map[string][]byte,
	operandRequestPhase odlm.ServicePhase,
) (*LicenseServiceUploadSecret, *LicenseServiceOperandRequest) {
	config := *buildMockConfig()

	// Create top-level resources that will be returned by this function
	operandRequest := LicenseServiceOperandRequest{
		BaseReconcilableResource: BaseReconcilableResource{
			Name:   "test-operand-request-resource",
			Config: config,
		},
	}
	secret := LicenseServiceUploadSecret{
		OperandRequest: &operandRequest,
		BaseReconcilableResource: BaseReconcilableResource{
			Name:   "test-upload-secret-resource",
			Config: config,
		},
	}

	// Populate secret data, assume it was already reconciled so expected is equal to actual
	secret.ActualResource = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-upload-secret",
			Namespace: namespace,
		},
		Data: secretData,
	}
	secret.ExpectedResource = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-upload-secret",
			Namespace: namespace,
		},
		Data: secretData,
	}

	// Populate operand request data, it won't be reconciled in any way so no need to have the expected resource
	operandRequest.ActualResource = &odlm.OperandRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-operand-request",
			Namespace: namespace,
		},
		Spec: odlm.OperandRequestSpec{
			Requests: []odlm.Request{{
				Operands: []odlm.Operand{{
					Name: operandRequestName,
					Bindings: map[string]odlm.SecretConfigmap{
						operandRequestBindingName: {
							Secret:    operandRequestBindingSecret,
							Configmap: operandRequestBindingConfigMap,
						},
					},
				}},
				// This field is unused by License Service but required by CRD
				Registry: "fake-registry",
			}},
		},
		Status: odlm.OperandRequestStatus{
			Members: []odlm.MemberStatus{
				{
					Name: operandRequestName,
					Phase: odlm.MemberPhase{
						OperandPhase: operandRequestPhase,
					},
				},
			},
		},
	}

	return &secret, &operandRequest
}

/*
Check that data is always overridden by data provided by the operand request, unless it's an exact match.
*/
func TestMarkShouldUpdate_UploadSecret_WithOperandRequest(t *testing.T) {
	testCases := map[string]struct {
		secretData          map[string][]byte
		operandRequestPhase odlm.ServicePhase
		expect              ReconcileStatus
	}{
		"Empty data with empty cluster phase": {
			secretData:          map[string][]byte{},
			operandRequestPhase: odlm.ServiceNotFound,
			expect: ReconcileStatus{
				ShouldUpdate: false,
			},
		},
		"Existing data with empty cluster phase": {
			secretData: map[string][]byte{
				"token": []byte(testExistingToken),
			},
			operandRequestPhase: odlm.ServiceFailed,
			expect: ReconcileStatus{
				ShouldUpdate: false,
			},
		},
		"Empty data with running cluster phase": {
			secretData:          map[string][]byte{},
			operandRequestPhase: odlm.ServiceRunning,
			expect: ReconcileStatus{
				ShouldUpdate: true,
			},
		},
		"Existing data with running cluster phase": {
			secretData: map[string][]byte{
				"token": []byte(testExistingToken),
			},
			operandRequestPhase: odlm.ServiceRunning,
			expect: ReconcileStatus{
				ShouldUpdate: true,
			},
		},
		"Exact match data": {
			secretData: map[string][]byte{
				"token":   []byte(testToken),
				"url":     []byte(testURL),
				"crt.pem": []byte(testCert),
			},
			operandRequestPhase: odlm.ServiceRunning,
			expect: ReconcileStatus{
				ShouldUpdate: false,
			},
		},
	}

	for name, test := range testCases {
		ok := t.Run(name, func(t *testing.T) {
			secret, _ := createMockSecretWithOperandRequest(test.secretData, test.operandRequestPhase)

			err := secret.MarkShouldUpdate()

			assert.Equal(t, test.expect.ShouldUpdate, secret.status.ShouldUpdate)
			assert.NoError(t, err)
		})
		assert.True(t, ok)
	}
}
