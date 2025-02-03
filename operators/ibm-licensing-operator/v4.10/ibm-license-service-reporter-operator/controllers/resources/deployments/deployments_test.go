package deployments

import (
	"context"
	"os"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/mocks"
	"go.uber.org/zap/zapcore"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
)

func TestReconcileDeploymentAppliesSpecLabels(t *testing.T) {
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
	reporter.Spec.RouteEnabled = ptr.To(true)
	scheme := runtime.NewScheme()
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(v1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&mocks.ConsoleRoute).Build()
	config := IBMLicenseServiceReporterConfig{
		IsRouteAPI: true,
		Instance:   reporter,
		Client:     client,
		Scheme:     scheme,
	}

	// Required to use ReconcileDeployment
	err := os.Setenv(OperandReporterDatabaseImageEnvVar, "test/test:test")
	assert.NoError(t, err)
	err = os.Setenv(OperandReporterReceiverImageEnvVar, "test/test:test")
	assert.NoError(t, err)
	err = os.Setenv(OperandReporterUIImageEnvVar, "test/test:test")
	assert.NoError(t, err)
	err = os.Setenv(OperandReporterAuthImageEnvVar, "test/test:test")
	assert.NoError(t, err)

	// Reconcile deployment should apply spec.labels via creation
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value"}
	err = ReconcileDeployment(logger, config)
	assert.NoError(t, err)

	// Check deployment created with spec.labels
	foundDeployment := appsv1.Deployment{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: GetDefaultResourceName(config.Instance.GetName())}, &foundDeployment)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", foundDeployment.GetLabels()["test-label"], foundDeployment.GetName()+" should have spec.labels applied")
	assert.Equal(t, "test-value", foundDeployment.Spec.Template.GetLabels()["test-label"], foundDeployment.GetName()+"'s spec.template should have spec.labels applied")

	// Reconcile deployment should apply spec.labels via patch
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value-patched"}
	err = ReconcileDeployment(logger, config)
	assert.NoError(t, err)

	// Check deployment patched with spec.labels
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: GetDefaultResourceName(config.Instance.GetName())}, &foundDeployment)
	assert.NoError(t, err)
	assert.Equal(t, "test-value-patched", foundDeployment.GetLabels()["test-label"], foundDeployment.GetName()+" should have spec.labels applied")
	assert.Equal(t, "test-value-patched", foundDeployment.Spec.Template.GetLabels()["test-label"], foundDeployment.GetName()+"'s spec.template should have spec.labels applied")
}
