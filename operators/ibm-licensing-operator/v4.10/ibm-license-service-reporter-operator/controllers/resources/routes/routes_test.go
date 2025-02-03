package routes

import (
	"context"
	"testing"

	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/mocks"
	"go.uber.org/zap/zapcore"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"

	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
)

func TestReconcileRoutesWithoutCertificatesAppliesSpecLabels(t *testing.T) {
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
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance:   reporter,
		Client:     client,
		Scheme:     scheme,
		IsRouteAPI: true,
	}

	// Reconcile routes should apply spec.labels via creation
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value"}
	err := ReconcileRoutesWithoutCertificates(logger, config)
	assert.NoError(t, err)

	// Check routes created with spec.labels
	foundRoute := routev1.Route{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: LicenseReporterResourceBase}, &foundRoute)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", foundRoute.GetLabels()["test-label"], foundRoute.GetName()+" should have spec.labels applied")
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterConsoleRouteName}, &foundRoute)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", foundRoute.GetLabels()["test-label"], foundRoute.GetName()+" should have spec.labels applied")

	// Reconcile routes should apply spec.labels via patch
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value-patched"}
	err = ReconcileRoutesWithoutCertificates(logger, config)
	assert.NoError(t, err)

	// Check routes patched with spec.labels
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: LicenseReporterResourceBase}, &foundRoute)
	assert.NoError(t, err)
	assert.Equal(t, "test-value-patched", foundRoute.GetLabels()["test-label"], foundRoute.GetName()+" should have spec.labels applied")
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: ReporterConsoleRouteName}, &foundRoute)
	assert.NoError(t, err)
	assert.Equal(t, "test-value-patched", foundRoute.GetLabels()["test-label"], foundRoute.GetName()+" should have spec.labels applied")
}

func TestReconcileRouteWithCertificatesAppliesSpecLabels(t *testing.T) {
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
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&mocks.ExternalCertSecret, &mocks.InternalCertSecret).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance:   reporter,
		Client:     client,
		Scheme:     scheme,
		IsRouteAPI: true,
	}

	// Reconcile route should apply spec.labels via creation
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value"}
	err := ReconcileRouteWithCertificates(logger, config)
	assert.NoError(t, err)

	// Check route created with spec.labels
	foundRoute := routev1.Route{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: LicenseReporterResourceBase}, &foundRoute)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", foundRoute.GetLabels()["test-label"], foundRoute.GetName()+" should have spec.labels applied")

	// Reconcile route should apply spec.labels via patch
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value-patched"}
	err = ReconcileRouteWithCertificates(logger, config)
	assert.NoError(t, err)

	// Check route patched with spec.labels
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: LicenseReporterResourceBase}, &foundRoute)
	assert.NoError(t, err)
	assert.Equal(t, "test-value-patched", foundRoute.GetLabels()["test-label"], foundRoute.GetName()+" should have spec.labels applied")
}
