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
	"context"
	"errors"
	"fmt"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	"time"

	"k8s.io/apimachinery/pkg/api/resource"

	operatorv1 "github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/api/v1"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/internal/controller/resources"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/tools/record"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

const (
	licenseNotAcceptedMessage = "To finish creation of the IBMLicenseServiceScanner instance, " +
		"please accept the license terms (ibm.biz/lsvc-lic) by setting the field \"spec.license.accept: true\"."

	defaultScanFrequency               = "10 0 * * *"
	defaultScanStartingDeadlineSeconds = 3600

	defaultRequestsResourceCPU              = "200m"
	defaultRequestsResourceMemory           = "128Mi"
	defaultRequestsResourceEphemeralStorage = "5Gi"

	defaultLimitsResourceCPU              = "500m"
	defaultLimitsResourceMemory           = "512Mi"
	defaultLimitsResourceEphemeralStorage = "5Gi"

	defaultImagePullPolicy = "IfNotPresent"
)

// ScannerReconciler reconciles an IBMLicenseServiceScanner object
type ScannerReconciler struct {
	client.Client
	Scheme   *runtime.Scheme
	Recorder record.EventRecorder
	// Determines if scanner should try to auto-connect to License Service; depends on the operand request CRD existence
	EnableAutoConnect bool
}

//+kubebuilder:rbac:groups=operator.ibm.com,namespace=ibm-licensing-scanner,resources=ibmlicenseservicescanners,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=operator.ibm.com,namespace=ibm-licensing-scanner,resources=ibmlicenseservicescanners/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=operator.ibm.com,namespace=ibm-licensing-scanner,resources=ibmlicenseservicescanners/finalizers,verbs=update
//+kubebuilder:rbac:groups=operator.ibm.com,namespace=ibm-licensing-scanner,resources=operandrequests,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups="",namespace=ibm-licensing-scanner,resources=secrets;configmaps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=batch,namespace=ibm-licensing-scanner,resources=cronjobs,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,namespace=ibm-licensing-scanner,resources=deployments,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apiextensions.k8s.io,resources=customresourcedefinitions,verbs=get;list;watch

// Reconcile is part of the main kubernetes reconciliation loop. For more details, check:
// https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.14.1/pkg/reconcile
func (r *ScannerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconcile loop starting")

	// Get scanner instance
	scannerInstance := operatorv1.IBMLicenseServiceScanner{}
	err := r.Get(ctx, req.NamespacedName, &scannerInstance)
	if err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	// Exit early if license is not accepted
	if accepted := isLicenseAccepted(&scannerInstance); !accepted {
		err = errors.New("license not accepted")
		r.Recorder.Event(&scannerInstance, "Warning", "LicenseNotAccepted", licenseNotAcceptedMessage)
		logger.Error(err, licenseNotAcceptedMessage)
		return ctrl.Result{}, err
	}

	// Set default values
	setScanFrequencyIfEmpty(&scannerInstance)
	setScanStartingDeadlineSecondsIfEmpty(&scannerInstance)
	setContainerIfEmpty(&scannerInstance)

	// Encapsulate common values
	config := resources.ScannerConfig{
		Scanner: scannerInstance,
		Context: ctx,
		Client:  r.Client,
	}

	// Initialize early to dynamically insert values
	var toReconcile []resources.Resource

	// Operand request to later populate the License Service connection secret, can be nil if the CRD is missing
	var licenseServiceOperandRequest *resources.LicenseServiceOperandRequest

	// If auto-connection to License Service is enabled, reconcile the operand request
	if r.EnableAutoConnect {
		logger.Info("Auto-connection with License Service enabled, the operand request will be reconciled")
		licenseServiceOperandRequest = &resources.LicenseServiceOperandRequest{
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   resources.LicenseServiceOperandRequestName,
				Config: config,
				Logger: logger.WithValues(
					"name", resources.LicenseServiceOperandRequestName,
					"type", "OperandRequest",
				),
			},
		}
		toReconcile = append(toReconcile, licenseServiceOperandRequest)
	} else {
		logger.Info("Auto-connection with License Service disabled - to enable it, please install ODLM" +
			" and restart the operator (e.g. recreate the pod)")
	}

	// Create a map of all resources to reconcile
	toReconcile = append(toReconcile, []resources.Resource{
		&resources.LicenseServiceUploadSecret{
			OperandRequest: licenseServiceOperandRequest,
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   scannerInstance.Spec.LicenseServiceUploadSecret,
				Config: config,
				Logger: logger.WithValues(
					"name", scannerInstance.Spec.LicenseServiceUploadSecret,
					"type", "Secret",
				),
			},
		},
		&resources.RegistryPullSecret{
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   scannerInstance.Spec.RegistryPullSecret,
				Config: config,
				Logger: logger.WithValues(
					"name", scannerInstance.Spec.RegistryPullSecret,
					"type", "Secret",
				),
			},
		},
		&resources.CacheConfigMap{
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   resources.CacheConfigMapName,
				Config: config,
				Logger: logger.WithValues(
					"name", resources.CacheConfigMapName,
					"type", "ConfigMap",
				),
			},
		},
		&resources.VaultScriptConfigMap{
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   resources.ScriptConfigMapName,
				Config: config,
				Logger: logger.WithValues(
					"name", resources.ScriptConfigMapName,
					"type", "ConfigMap",
				),
			},
		},
		&resources.ScannerCronJob{
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   resources.ScannerCronJobName,
				Config: config,
				Logger: logger.WithValues(
					"name", resources.ScannerCronJobName,
					"type", "CronJob",
				),
			},
		},
	}...)

	// Extract all unique service accounts, not to create excessive tokens
	// Map will be used as Set, value doesn't matter.
	serviceAccounts := map[string]bool{}
	registries := config.Scanner.Spec.Registries
	for index := range registries {
		if registries[index].AuthMethod == resources.VaultAuthenticationMethodName &&
			registries[index].VaultDetails != (operatorv1.VaultDetails{}) {
			// Service accounts map is considered a set, so only the key matters (using it for duplicates removal)
			serviceAccounts[registries[index].VaultDetails.ServiceAccount] = true
		}
	}

	// Create secrets storing ServiceAccount JWT
	for serviceAccount := range serviceAccounts {
		toReconcile = append(toReconcile, &resources.VaultServiceAccountTokenSecret{
			ServiceAccountName: serviceAccount,
			BaseReconcilableResource: resources.BaseReconcilableResource{
				Name:   serviceAccount,
				Config: config,
				Logger: logger.WithValues(
					"name", serviceAccount,
					"type", "Secret",
				),
			},
		})
	}

	// Reconcile resources (ignore nil-s which can be present because of optional CRD-s) -> init, check init, reconcile
	for _, resourceToReconcile := range toReconcile {
		if err := resourceToReconcile.Init(); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed to initialize resource: %w", err)
		}
		if err := resourceToReconcile.CheckInit(); err != nil {
			return ctrl.Result{}, fmt.Errorf("resource initialized incorrectly: %w", err)
		}
		result, err := resourceToReconcile.Reconcile()
		if err != nil {
			return result, fmt.Errorf("failed to reconcile resource: %w", err)
		}
		if result.Requeue {
			logger.Info("Reconcile requeue requested")
			return result, err
		}
	}

	logger.Info("Reconcile loop finished")

	return ctrl.Result{RequeueAfter: time.Hour}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ScannerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	builder := ctrl.NewControllerManagedBy(mgr).
		For(&operatorv1.IBMLicenseServiceScanner{}).
		Owns(&corev1.Secret{}).
		Owns(&corev1.ConfigMap{}).
		Owns(&batchv1.CronJob{})

	if r.EnableAutoConnect {
		builder = builder.Owns(&odlm.OperandRequest{})
	}

	return builder.Complete(r)
}

/*
Check if product license is accepted.
*/
func isLicenseAccepted(scannerInstance *operatorv1.IBMLicenseServiceScanner) bool {
	if scannerInstance.Spec.License == nil || !scannerInstance.Spec.License.Accept {
		return false
	}

	return true
}

/*
By default, the scan runs 10 minutes after midnight every day.
*/
func setScanFrequencyIfEmpty(scannerInstance *operatorv1.IBMLicenseServiceScanner) {
	if scannerInstance.Spec.Scan.Frequency == "" {
		scannerInstance.Spec.Scan.Frequency = defaultScanFrequency
	}
}

/*
By default, scheduled but not started jobs expire after 1 hour.
*/
func setScanStartingDeadlineSecondsIfEmpty(scannerInstance *operatorv1.IBMLicenseServiceScanner) {
	if scannerInstance.Spec.Scan.StartingDeadlineSeconds == 0 {
		scannerInstance.Spec.Scan.StartingDeadlineSeconds = defaultScanStartingDeadlineSeconds
	}
}

/*
Set default container resource requirements and image pull policy.

5GB-s storage limit is set to allow for heavy images download.
*/
func setContainerIfEmpty(scannerInstance *operatorv1.IBMLicenseServiceScanner) {
	if scannerInstance.Spec.Container == nil {
		scannerInstance.Spec.Container = &operatorv1.Container{
			Resources: operatorv1.ResourceRequirementsNoClaims{
				Requests: getDefaultContainerResourceRequests(),
				Limits:   getDefaultContainerResourceLimits(),
			},
			ImagePullPolicy: defaultImagePullPolicy,
		}
	} else {
		if scannerInstance.Spec.Container.Resources.Requests == nil {
			scannerInstance.Spec.Container.Resources.Requests = getDefaultContainerResourceRequests()
		}

		if scannerInstance.Spec.Container.Resources.Limits == nil {
			scannerInstance.Spec.Container.Resources.Limits = getDefaultContainerResourceLimits()
		}

		if scannerInstance.Spec.Container.ImagePullPolicy == "" {
			scannerInstance.Spec.Container.ImagePullPolicy = defaultImagePullPolicy
		}
	}
}

/*
Get default container resource list for requests.
*/
func getDefaultContainerResourceRequests() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse(defaultRequestsResourceCPU),
		corev1.ResourceMemory:           resource.MustParse(defaultRequestsResourceMemory),
		corev1.ResourceEphemeralStorage: resource.MustParse(defaultRequestsResourceEphemeralStorage),
	}
}

/*
Get default container resource list for limits.
*/
func getDefaultContainerResourceLimits() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse(defaultLimitsResourceCPU),
		corev1.ResourceMemory:           resource.MustParse(defaultLimitsResourceMemory),
		corev1.ResourceEphemeralStorage: resource.MustParse(defaultLimitsResourceEphemeralStorage),
	}
}
