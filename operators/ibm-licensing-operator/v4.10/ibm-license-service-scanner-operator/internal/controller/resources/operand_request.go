package resources

import (
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	LicenseServiceOperandRequestName = resourceNamePrefix + "ls-operand-request"

	operandRequestName             = "ibm-licensing-operator"
	operandRequestBindingName      = "public-api-upload"
	operandRequestBindingSecret    = "ibm-licensing-upload-token"
	operandRequestBindingConfigMap = "ibm-licensing-upload-config"
)

type LicenseServiceOperandRequest struct {
	BaseReconcilableResource
}

func (r *LicenseServiceOperandRequest) Init() error {
	r.Logger.Info("Initializing resource")

	// Configure operand request for license service -> ask for upload config and upload token data
	r.ExpectedResource = &odlm.OperandRequest{
		ObjectMeta: metav1.ObjectMeta{
			Name:      r.Name,
			Namespace: r.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: r.Config.Scanner.APIVersion,
				Kind:       r.Config.Scanner.Kind,
				Name:       r.Config.Scanner.Name,
				UID:        r.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      r.GetBaseLabels(),
			Annotations: r.GetBaseAnnotations(),
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
				Registry: r.Name,
			}},
		},
	}

	// Initialize an empty object to populate later
	r.ActualResource = &odlm.OperandRequest{}

	return nil
}

func (r *LicenseServiceOperandRequest) PopulateExpectedFromActual() {
	r.BaseReconcilableResource.PopulateExpectedFromActual()

	// Persist members to save information about the operand request status
	r.ExpectedResource.(*odlm.OperandRequest).Status.Members = r.ActualResource.(*odlm.OperandRequest).Status.Members
}

/*
GetRequestPhase retrieves operand request phase for the License Service member.
*/
func (r *LicenseServiceOperandRequest) GetRequestPhase() odlm.ServicePhase {
	phase := odlm.ServiceNotFound

	for _, member := range r.ActualResource.(*odlm.OperandRequest).Status.Members {
		if member.Name == operandRequestName {
			phase = member.Phase.OperandPhase
		}
	}

	return phase
}

func (r *LicenseServiceOperandRequest) Reconcile() (ctrl.Result, error) {
	r.Logger.Info("Reconciling License Service operand request")

	return ReconcileResource(r)
}
