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
	"errors"
	"fmt"

	"time"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	api "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	res "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/deployments"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/ingress"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/pvc"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/routes"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	apieq "k8s.io/apimachinery/pkg/api/equality"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/tools/record"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/builder"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/log"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

// IBMLicenseServiceReporterReconciler reconciles a IBMLicenseServiceReporter object
type IBMLicenseServiceReporterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
	// record.EventRecorder allows reconciler to send events to k8s API
	Recorder record.EventRecorder
	Dynamic  dynamic.Interface
}

// RBAC for OPERATOR, run make bundle after changing this
// Cluster role
// +kubebuilder:rbac:groups=storage.k8s.io,resources=storageclasses,verbs=get;list;watch
// +kubebuilder:rbac:groups=operator.openshift.io,resources=servicecas,verbs=list
// +kubebuilder:rbac:groups=operator.ibm.com,resources=ibmlicenseservicereporters;ibmlicenseservicereporters/status;ibmlicenseservicereporters/finalizers,verbs=get;list;watch;create;update;patch;delete

// Role
// +kubebuilder:rbac:namespace=ibm-licensing,groups=apps,resources=deployments;daemonsets;replicasets;statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-licensing,groups=route.openshift.io,resources=routes;routes/custom-host,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-licensing,groups="",resources=events,verbs=create;patch
// +kubebuilder:rbac:namespace=ibm-licensing,groups="",resources=services;services/finalizers;endpoints;persistentvolumeclaims;configmaps;secrets;namespaces;serviceaccounts,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:namespace=ibm-licensing,groups="",resources=pods,verbs=list;watch;delete
// +kubebuilder:rbac:namespace=ibm-licensing,groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// Reconcile function compares the state specified by
// the IBMLicenseServiceReporter object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *IBMLicenseServiceReporterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)
	logger.Info("Reconciling IBMLicenseServiceReporter")

	//get the configuration from request
	instance, err := r.GetInstance(req)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile req.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// if some cleaning after deleting needs to be added then it should be here with handling not found error
		return ctrl.Result{}, err
	}

	// fill additional information from cluster that resource reconcilers might need
	config := r.GetFullInformation(ctx, logger, instance)

	// TODO: it would be nice to create some dependencies between these reconcile functions
	//  it would probably be best to use some common interface
	//  then there could be function taking all these interface implementations and based on dependencies creating reconcile order
	//  such solution would allow less error prone plugging new resources,
	//  also it could have interface for providing variables/functions from dependencies to avoid circular dependencies

	// The creation of Custom Resource is available after accepting license terms
	if !instance.Spec.IsLicenseAccepted() {
		r.handleLicenseNotAccepted(instance)
		return ctrl.Result{}, nil
	}

	// resources without dependencies on different ones
	if err = secrets.ReconcileAPISecretToken(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = secrets.ReconcileDatabaseSecret(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = secrets.ReconcileCredentialsSecrets(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = secrets.ReconcileCookieSecret(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	// service needed for cert handling
	if err = services.ReconcileService(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	// cert handling needed for deployment and its volumes
	if err = routes.ReconcileRoutesWithoutCertificates(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = secrets.ReconcileExternalCertificateSecret(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = secrets.ReconcileInternalCertificateSecret(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = routes.ReconcileRouteWithCertificates(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	// pvc needed for deployment
	if err = pvc.ReconcilePersistentVolumeClaim(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = deployments.ReconcileDeployment(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = ingress.ReconcileApiIngress(logger, config); err != nil {
		return ctrl.Result{}, err
	}
	if err = ingress.ReconcileConsoleIngress(logger, config); err != nil {
		return ctrl.Result{}, err
	}

	// TODO: create some way of communication with existing LS to automatically create sender

	if err := r.updateStatus(ctx, logger, &instance); err != nil {
		return ctrl.Result{}, err
	}

	res.AddResourceValuesToLog(logger, &instance).Info("Reconcile all done")

	return ctrl.Result{}, nil
}

func (r *IBMLicenseServiceReporterReconciler) handleLicenseNotAccepted(instance api.IBMLicenseServiceReporter) {
	// Generate the current timestamp in the specified format
	timestamp := time.Now().UTC().Format("2006-01-02T15:04:05Z")
	// Format the ERROR log message without stacktrace
	fmt.Printf("%s ERROR "+api.LicenseNotAcceptedMessage+"\n", timestamp)
	// Publish an event with error message
	r.Recorder.Event(&instance, "Warning", "LicenseNotAccepted", api.LicenseNotAcceptedMessage)
}

// SetupWithManager sets up the controller with the Manager.
func (r *IBMLicenseServiceReporterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&api.IBMLicenseServiceReporter{}).
		Watches(
			&source.Kind{Type: &networkingv1.Ingress{}},
			handler.EnqueueRequestsFromMapFunc(r.checkIfConsoleIngressChanged),
			builder.WithPredicates(predicate.ResourceVersionChangedPredicate{})).
		Complete(r)
}

/*
checkIfConsoleIngressChanged narrows down the reconciliation to only those Ingresses changes that are applied to the
Ingress specific to LSR Console since this needs to be synced with the LSR pod (oauth2 proxy needs to know where the
identity provider should redirect the customer after log in attempt). There's no way to add watcher to a specific resource,
you subscribe to changes to the resources of a specific kind for which you create the reconciler (with `For` method),
resources you've created that need to be properly equipped with the OwnerReferences field stating that this reconciler is
their owner (with `Owns` method), or by watching a certain kind, but by default you get the notification about every change
in your namespace of a resource with that kind (with `Watches` method). furthermore, you need to pass the CR(s) as the
object to be reconciled, reconciliation will find out the LSR needs to be recreated due to the auth container args change
*/
func (r *IBMLicenseServiceReporterReconciler) checkIfConsoleIngressChanged(changedIngress client.Object) []reconcile.Request {
	// we care only if the Console ingress changed
	if changedIngress.GetName() == ingress.ConsoleIngressName {
		ctx := context.TODO()
		crs := &api.IBMLicenseServiceReporterList{}
		if err := r.List(ctx, crs); err != nil {
			return []reconcile.Request{}
		}

		requests := make([]reconcile.Request, len(crs.Items))
		for i, item := range crs.Items {
			// reconcile only when we're in the scenario where customer is
			// required to provide manually created Ingress for the LSR Console
			config := r.GetFullInformation(ctx, log.FromContext(ctx), item)
			if !(config.IsRouteAPI && config.Instance.Spec.IsRouteEnabled()) && !config.Instance.Spec.IngressEnabled {
				requests[i] = reconcile.Request{
					NamespacedName: types.NamespacedName{
						Name:      item.GetName(),
						Namespace: item.GetNamespace(),
					},
				}
			}
		}
		return requests
	}
	return []reconcile.Request{}
}

func (r *IBMLicenseServiceReporterReconciler) GetFullInformation(
	ctx context.Context, logger logr.Logger, instance api.IBMLicenseServiceReporter) res.IBMLicenseServiceReporterConfig {

	logger.Info("Filling additional information from cluster")

	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
	}
	config := res.IBMLicenseServiceReporterConfig{Instance: instance, Client: r.Client, Scheme: r.Scheme}

	var err error
	routeToFind := &routev1.RouteList{}
	err = r.Client.List(ctx, routeToFind, listOpts...)
	if err == nil {
		config.IsRouteAPI = true
		logger.Info("Route feature is enabled")
	} else {
		config.IsRouteAPI = false
		logger.Info("Route feature is disabled")
	}

	if config.IsRouteAPI {
		if r.Dynamic != nil {
			serviceCAGVR := schema.GroupVersionResource{
				Group:    "operator.openshift.io",
				Version:  "v1",
				Resource: "servicecas",
			}
			_, err = r.Dynamic.Resource(serviceCAGVR).List(ctx, metav1.ListOptions{})
			if err == nil {
				config.IsServiceCAAPI = true
			}
		} else {
			err = errors.New("dynamic k8s config is not initialized")
		}
		if config.IsServiceCAAPI {
			logger.Info("ServiceCA feature is enabled")
		} else {
			logger.Error(err, "ServiceCA feature is disabled")
		}
	}

	if config.Instance.Spec.RouteEnabled == nil {
		config.Instance.Spec.RouteEnabled = ptr.To(config.IsRouteAPI)
	}

	return config
}

func (r *IBMLicenseServiceReporterReconciler) GetInstance(req ctrl.Request) (api.IBMLicenseServiceReporter, error) {
	foundInstance := api.IBMLicenseServiceReporter{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, &foundInstance)
	return foundInstance, err
}

func (r *IBMLicenseServiceReporterReconciler) updateStatus(
	ctx context.Context, logger logr.Logger, instance *api.IBMLicenseServiceReporter) error {

	podList := &corev1.PodList{}
	listOpts := []client.ListOption{
		client.InNamespace(instance.GetNamespace()),
		client.MatchingLabels(res.LabelsForPod(*instance)),
	}
	if err := r.Client.List(ctx, podList, listOpts...); err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	var podStatuses []corev1.PodStatus
	for _, pod := range podList.Items {
		if pod.Status.Conditions != nil {
			i := 0
			for _, podCondition := range pod.Status.Conditions {
				if (podCondition.LastProbeTime == metav1.Time{Time: time.Time{}}) {
					// Time{} is treated as null and causes error at status update so value need to be changed to some other default empty value
					pod.Status.Conditions[i].LastProbeTime = metav1.Time{
						Time: time.Unix(0, 1),
					}
				}
				i++
			}
		}
		podStatuses = append(podStatuses, pod.Status)
	}

	if !apieq.Semantic.DeepEqual(podStatuses, instance.Status.LicenseServiceReporterPods) {
		logger.Info("Updating IBMLicenseServiceReporter status")
		instance.Status.LicenseServiceReporterPods = podStatuses
		err := r.Client.Status().Update(ctx, instance)
		if err != nil {
			logger.Info("Failed to update pod status")
		}
	}

	return nil
}
