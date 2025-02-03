package pvc

import (
	routev1 "github.com/openshift/api/route/v1"
	"github.com/stretchr/testify/assert"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	operatorv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	"go.uber.org/zap/zapcore"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
	"testing"

	"context"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
)

func TestReconcilePVCAppliesSpecLabels(t *testing.T) {
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
	// Basic storage class required for ReconcilePersistentVolumeClaim
	storageClass := storagev1.StorageClass{}
	storageClass.ObjectMeta.SetAnnotations(map[string]string{"storageclass.kubernetes.io/is-default-class": "true"})
	scheme := runtime.NewScheme()
	utilruntime.Must(routev1.Install(scheme))
	utilruntime.Must(clientgoscheme.AddToScheme(scheme))
	utilruntime.Must(operatorv1alpha1.AddToScheme(scheme))
	client := fake.NewClientBuilder().WithScheme(scheme).WithObjects(&storageClass).Build()
	config := IBMLicenseServiceReporterConfig{
		Instance: reporter,
		Client:   client,
		Scheme:   scheme,
	}

	// Reconcile PVC should apply spec.labels via creation
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value"}
	err := ReconcilePersistentVolumeClaim(logger, config)
	assert.NoError(t, err)

	// Check PVC created with spec.labels
	foundPVC := corev1.PersistentVolumeClaim{}
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: PersistentVolumeClaimName}, &foundPVC)
	assert.NoError(t, err)
	assert.Equal(t, "test-value", foundPVC.GetLabels()["test-label"], foundPVC.GetName()+" should have spec.labels applied")

	// Reconcile PVC should apply spec.labels via patch
	config.Instance.Spec.Labels = map[string]string{"test-label": "test-value-patched"}
	err = ReconcilePersistentVolumeClaim(logger, config)
	assert.NoError(t, err)

	// Check PVC patched with spec.labels
	err = client.Get(context.TODO(), types.NamespacedName{Namespace: namespace, Name: PersistentVolumeClaimName}, &foundPVC)
	assert.NoError(t, err)
	assert.Equal(t, "test-value-patched", foundPVC.GetLabels()["test-label"], foundPVC.GetName()+" should have spec.labels applied")
}
