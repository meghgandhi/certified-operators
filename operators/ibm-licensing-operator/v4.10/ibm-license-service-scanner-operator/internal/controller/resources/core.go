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
	"errors"
	"fmt"
	"github.com/go-logr/logr"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/api/v1"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/version"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	applicationName      = "ibm-licensing-scanner"
	resourceNamePrefix   = "ibm-licensing-scanner-"
	operatorResourceName = resourceNamePrefix + "operator"

	annotationProductID     = "105fa377cada4660a213f99e02c53782"
	annotationProductName   = "IBM License Service Scanner"
	annotationProductMetric = "FREE"
)

/*
Reconcilable objects provide methods required to reconcile a resource using the ReconcileResource function.

The methods are as follows:
- GetReconcileStatus: Get reconciliation status, for example to determine if resource update is needed
- GetResource: Fetch the resource from the cluster
- MarkShouldCreate: If the resource doesn't exist, mark if it should be created
- MarkShouldUpdate: If the resource does exist, mark if it should be updated
- CreateResource: Create a resource on the cluster
- UpdateResource: Update the cluster resource
*/
type Reconcilable interface {
	GetReconcileStatus() ReconcileStatus
	GetResource() error
	MarkShouldCreate() error
	MarkShouldUpdate() error
	CreateResource() error
	UpdateResource() error
	PopulateExpectedFromActual()
}

/*
ReconcileStatus is used within Reconcilable objects (and, consequently, in ReconcileResource).
*/
type ReconcileStatus struct {
	ShouldCreate bool
	ShouldUpdate bool
}

/*
ReconcileResource reconciles a Reconcilable object.

This function is used within BaseReconcilableResource and can act as an example implementation of Resource.Reconcile.
*/
func ReconcileResource(reconcilable Reconcilable) (ctrl.Result, error) {
	if err := reconcilable.GetResource(); err != nil {
		if apierrors.IsNotFound(err) {
			if err = reconcilable.MarkShouldCreate(); err != nil {
				return ctrl.Result{}, fmt.Errorf("failed marking the create flag: %w", err)
			}
		} else {
			return ctrl.Result{}, fmt.Errorf("failed getting the resource: %w", err)
		}
	} else {
		reconcilable.PopulateExpectedFromActual()
		if err = reconcilable.MarkShouldUpdate(); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed marking the update flag: %w", err)
		}
	}

	// Create or update/patch/recreate resource
	if reconcilable.GetReconcileStatus().ShouldCreate {
		if err := reconcilable.CreateResource(); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed creating the resource: %w", err)
		}
	} else if reconcilable.GetReconcileStatus().ShouldUpdate {
		if err := reconcilable.UpdateResource(); err != nil {
			return ctrl.Result{}, fmt.Errorf("failed updating the resource: %w", err)
		}
	}

	return ctrl.Result{}, nil
}

/*
Resource objects are used within the main reconcile loop. Before being reconciled, they must be initialized and
their state after initialisation checked. The steps exist to make sure that reconciliation itself is done on properly
prepared objects.

The base set of methods used within the main reconcile loop is:
- Init: initialize the resource
- CheckInit: verify if the resource has been initialized properly
- Reconcile: reconcile the resource

See BaseReconcilableResource for more details and default implementation.
*/
type Resource interface {
	Init() error
	CheckInit() error
	Reconcile() (ctrl.Result, error)
}

/*
BaseReconcilableResource provides the default implementation of the Reconcilable and Resource interfaces.

Each "inheriting" struct should:
- Implement Resource.Init
- Extend CheckInit to verify state of resource-specific fields
- Adjust MarkShouldCreate and MarkShouldUpdate methods if the default implementation is not sufficient
- Provide a basic implementation of Resource.Reconcile, most likely using the ReconcileResource function
- Shadow any other methods to achieve required custom functionalities

To learn more about the default implementation of each method, please refer to the docs provided alongside.
*/
type BaseReconcilableResource struct {
	// Base parameters required for configuration or access to specific methods and fields
	Name   string
	Config ScannerConfig
	Logger logr.Logger

	// Required for most API operations
	ExpectedResource client.Object
	ActualResource   client.Object

	// Status is private as it's exposed via getter, required for the reconcile flow
	status ReconcileStatus
}

/*
ScannerConfig encapsulates important values that may be used within each function.
*/
type ScannerConfig struct {
	Scanner v1.IBMLicenseServiceScanner
	Context context.Context
	Client  client.Client
}

/*
CheckInit to verify if required fields are provided.
*/
func (b *BaseReconcilableResource) CheckInit() error {
	b.Logger.Info("Checking initialisation succeeded")

	// Check resources are not nil
	if b.ExpectedResource == nil {
		return errors.New("expected resource is nil")
	}
	if b.ActualResource == nil {
		return errors.New("actual resource is nil")
	}
	if b.Config.Context == nil {
		return errors.New("context is nil")
	}
	if b.Config.Client == nil {
		return errors.New("client is nil")
	}

	// Check name is not empty
	if b.Name == "" {
		return errors.New("resource name is empty")
	}

	// Check expected resource has minimal data included
	if b.ExpectedResource.GetName() == "" {
		return errors.New("expected resource's name is empty")
	}
	if b.ExpectedResource.GetNamespace() == "" {
		return errors.New("expected resource's namespaces is empty")
	}

	return nil
}

/*
GetReconcileStatus returns the internal status field.
*/
func (b *BaseReconcilableResource) GetReconcileStatus() ReconcileStatus {
	return b.status
}

/*
GetResource fetches the resource from the cluster and checks for next actions depending on the API call's result.
*/
func (b *BaseReconcilableResource) GetResource() error {
	b.Logger.Info("Getting resource from cluster")

	return b.Config.Client.Get(b.Config.Context, types.NamespacedName{
		Name:      b.ExpectedResource.GetName(),
		Namespace: b.ExpectedResource.GetNamespace(),
	}, b.ActualResource)
}

/*
MarkShouldCreate resource if it doesn't exist; always create by default.
*/
func (b *BaseReconcilableResource) MarkShouldCreate() error {
	b.Logger.Info("Checking if resource should be created")

	b.status.ShouldCreate = true

	return nil
}

/*
MarkShouldUpdate resource if it doesn't exist; never update by default.
*/
func (b *BaseReconcilableResource) MarkShouldUpdate() error {
	b.Logger.Info("Checking if resource should be updated")

	if !b.labelsEqual() || !b.annotationsEqual() {
		b.status.ShouldUpdate = true
	}

	return nil
}

/*
Check if expected and actual resource's labels match.

Consider labels equal even if there are labels present in the actual, cluster resource, which are missing from the
expected resource. This is to avoid triggering unnecessary updates, because as long as the actual resource has all
required labels from the expected resource, there is no need to reconcile.
*/
func (b *BaseReconcilableResource) labelsEqual() bool {
	actualResourceLabels := b.ActualResource.GetLabels()

	// Check if all labels from expected resource present in actual resource (then check if values match)
	for key, expectedValue := range b.ExpectedResource.GetLabels() {
		if actualValue, ok := actualResourceLabels[key]; !ok || expectedValue != actualValue {
			return false
		}
	}

	return true
}

/*
Check if expected and actual resource's annotations match.

Consider annotations equal even if there are annotations present in the actual, cluster resource, which are missing
from the expected resource. This is to avoid triggering unnecessary updates, because as long as the actual resource
has all required annotations from the expected resource, there is no need to reconcile.
*/
func (b *BaseReconcilableResource) annotationsEqual() bool {
	actualResourceAnnotations := b.ActualResource.GetAnnotations()

	// Check if all annotations from expected resource present in actual resource (then check if values match)
	for key, expectedValue := range b.ExpectedResource.GetAnnotations() {
		if actualValue, ok := actualResourceAnnotations[key]; !ok || expectedValue != actualValue {
			return false
		}
	}

	return true
}

/*
CreateResource if it doesn't exist.
*/
func (b *BaseReconcilableResource) CreateResource() error {
	b.Logger.Info("Creating resource")

	if err := b.Config.Client.Create(b.Config.Context, b.ExpectedResource); err != nil {
		return fmt.Errorf("failed creating resource: %w", err)
	}

	return nil
}

/*
UpdateResource if the update flag is set to true.
*/
func (b *BaseReconcilableResource) UpdateResource() error {
	if err := b.patchResource(); err != nil {
		b.Logger.Info("Couldn't use PATCH, trying UPDATE")

		if err := b.updateResource(); err != nil {
			b.Logger.Info("Couldn't use UPDATE, trying DELETE & CREATE")

			if err := b.deleteResource(); err != nil {
				return fmt.Errorf("all available resource update flows failed: %w", err)
			}

			return b.CreateResource()
		}
	}

	return nil
}

/*
Patch a resource.
*/
func (b *BaseReconcilableResource) patchResource() error {
	b.Logger.Info("Patching resource")

	patch := client.StrategicMergeFrom(b.ActualResource)
	if err := b.Config.Client.Patch(b.Config.Context, b.ExpectedResource, patch); err != nil {
		return fmt.Errorf("failed patching resource: %w", err)
	}

	return nil
}

/*
Update a resource.
*/
func (b *BaseReconcilableResource) updateResource() error {
	b.Logger.Info("Updating resource")

	if err := b.Config.Client.Update(b.Config.Context, b.ExpectedResource); err != nil {
		return fmt.Errorf("failed updating resource: %w", err)
	}

	return nil
}

/*
Delete a resource.
*/
func (b *BaseReconcilableResource) deleteResource() error {
	b.Logger.Info("Deleting resource")

	if err := b.Config.Client.Delete(b.Config.Context, b.ActualResource); err != nil {
		return fmt.Errorf("failed deleting resource: %w", err)
	}

	return nil
}

/*
PopulateExpectedFromActual adds data to the expected resource based on the actual, cluster resource.

Useful to keep data persistence, for example in case of custom labels and annotations.
*/
func (b *BaseReconcilableResource) PopulateExpectedFromActual() {
	b.populateExpectedWithActualLabels()
	b.populateExpectedWithActualAnnotations()
}

/*
Add labels from the actual resource to the expected resource, for persistence.

Labels (keys) already present in the expected resource are not overridden to allow for updates.
*/
func (b *BaseReconcilableResource) populateExpectedWithActualLabels() {
	expectedLabels := b.ExpectedResource.GetLabels()

	for key, value := range b.ActualResource.GetLabels() {
		_, ok := expectedLabels[key]
		if !ok {
			expectedLabels[key] = value
		}
	}
}

/*
Add annotations from the actual resource to the expected resource, for persistence.

Annotations (keys) already present in the expected resource are not overridden to allow for updates.
*/
func (b *BaseReconcilableResource) populateExpectedWithActualAnnotations() {
	expectedAnnotations := b.ExpectedResource.GetAnnotations()

	for key, value := range b.ActualResource.GetAnnotations() {
		_, ok := expectedAnnotations[key]
		if !ok {
			expectedAnnotations[key] = value
		}
	}
}

/*
GetBaseLabels returns predefined + custom labels for a resource.
*/
func (b *BaseReconcilableResource) GetBaseLabels() map[string]string {
	labels := map[string]string{
		"app.kubernetes.io/name":       applicationName,
		"app.kubernetes.io/version":    version.Version,
		"app.kubernetes.io/component":  b.Name,
		"app.kubernetes.io/managed-by": operatorResourceName,
	}

	for key, value := range b.Config.Scanner.Spec.Labels {
		labels[key] = value
	}

	return labels
}

/*
GetBaseAnnotations returns predefined + custom annotations for a resource.
*/
func (b *BaseReconcilableResource) GetBaseAnnotations() map[string]string {
	annotations := map[string]string{
		"productID":     annotationProductID,
		"productName":   annotationProductName,
		"productMetric": annotationProductMetric,
	}

	for key, value := range b.Config.Scanner.Spec.Annotations {
		annotations[key] = value
	}

	return annotations
}
