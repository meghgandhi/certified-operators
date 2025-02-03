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

package resources

import (
	"context"
	"fmt"
	"reflect"

	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/util/json"

	"github.com/go-logr/logr"
	"github.com/pkg/errors"
	apiv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

type FoundObject client.Object
type ExpectedObject client.Object

// ResourceUpdateStatus declares if a resource is in its desired state, and if not, is a patch sufficient to fix it?
// Should only matter if IsInDesiredState evaluates to false (so resource needs update/patch)
// By default both values are evaluated as false (zero-values of booleans)
type ResourceUpdateStatus struct {
	IsInDesiredState  bool
	IsPatchSufficient bool
}

// IsFoundInDesiredStateIfExistsFunc returns: (is-desired state; error)
type IsFoundInDesiredStateIfExistsFunc func(
	config IBMLicenseServiceReporterConfig,
	found FoundObject,
	expected ExpectedObject,
	logger logr.Logger,
) (ResourceUpdateStatus, error)

//goland:noinspection GoUnusedExportedFunction
func FoundIsAlwaysInDesiredStateWhenExists(
	_ IBMLicenseServiceReporterConfig,
	_ FoundObject,
	_ ExpectedObject,
	_ logr.Logger,
) (ResourceUpdateStatus, error) {
	return ResourceUpdateStatus{true, false}, nil
}

// ApplyPatchToFoundObjectFunc returns: (Patched object, error)
type ApplyPatchToFoundObjectFunc func(
	config IBMLicenseServiceReporterConfig,
	found FoundObject,
	logger logr.Logger,
) (client.Patch, error)

// PatchFoundWithSpecLabelsAndAnnotations attaches labels and annotations to an object and calls the Patch API request
func PatchFoundWithSpecLabelsAndAnnotations(
	config IBMLicenseServiceReporterConfig,
	found FoundObject,
	logger logr.Logger,
) (client.Patch, error) {
	logger.Info("Patching " + reflect.TypeOf(found).String() + " " + found.GetName() + " due to having outdated spec.labels and/or spec.annotations")

	data := map[string]interface{}{
		"metadata": map[string]interface{}{
			"labels":      GetSpecLabels(config.Instance),
			"annotations": GetSpecAnnotations(config.Instance),
		},
	}

	return MarshalAndPatch(data, found.GetResourceVersion())
}

// MarshalAndPatch converts data to a string while ensuring optimistic locking and returns a Patch object
func MarshalAndPatch(data map[string]interface{}, resourceVersion string) (client.Patch, error) {
	if data["metadata"] != nil {
		data["metadata"].(map[string]interface{})["resourceVersion"] = resourceVersion
	} else {
		data["metadata"] = map[string]interface{}{"resourceVersion": resourceVersion}
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return client.RawPatch(types.MergePatchType, dataBytes), nil
}

// MergeMaps combines two maps together, copying key, value pairs from new to the original (and returning a copy)
func MergeMaps(original, new map[string]string) map[string]string {
	if new == nil {
		new = make(map[string]string)
	}

	// Copy to avoid overriding passed data
	originalCopy := make(map[string]string)
	for key, value := range original {
		originalCopy[key] = value
	}
	for key, value := range new {
		originalCopy[key] = value
	}

	return originalCopy
}

// MergeIntoFoundAsNewObjectFunc returns: (merged object, shouldIgnoreUpdateErr bool, error)
type MergeIntoFoundAsNewObjectFunc func(found FoundObject, expected ExpectedObject) (client.Object, bool, error)

func OverrideFoundWithExpected(_ FoundObject, expected ExpectedObject) (client.Object, bool, error) {
	shouldIgnoreUpdateErr := true
	return expected, shouldIgnoreUpdateErr, nil
}

type NotFoundExpectedOverrideFunc func(expected ExpectedObject) (ExpectedObject, error)

type RunActionsAfterReconcile func(shouldRun bool, config IBMLicenseServiceReporterConfig, logger logr.Logger) error

type IBMLicenseServiceReporterConfig struct {
	Instance apiv1alpha1.IBMLicenseServiceReporter
	client.Client
	*runtime.Scheme
	IsServiceCAAPI bool
	IsRouteAPI     bool
}

func ReconcileResource(
	config IBMLicenseServiceReporterConfig,
	expected client.Object,
	found client.Object,
	shouldCreateIfNotExists bool,
	overrideExpected NotFoundExpectedOverrideFunc,
	checkFound IsFoundInDesiredStateIfExistsFunc,
	patchFound ApplyPatchToFoundObjectFunc,
	mergeIntoNew MergeIntoFoundAsNewObjectFunc,
	logger logr.Logger,
	runAdditionalActions RunActionsAfterReconcile,
) error {
	logger.Info("Reconciling resource")
	EnsureCachingLabel(expected)
	expectedNamespacedName := types.NamespacedName{
		Name:      expected.GetName(),
		Namespace: expected.GetNamespace(),
	}
	err := config.Get(context.TODO(), expectedNamespacedName, found)
	if err != nil {
		if apierrors.IsNotFound(err) {
			if shouldCreateIfNotExists {
				logger.Info("Resource does not exist, trying creating new one")
				if overrideExpected != nil {
					expected, err = overrideExpected(expected)
					if err != nil {
						return err
					}
					if expected.GetNamespace() != expectedNamespacedName.Namespace ||
						expected.GetName() != expectedNamespacedName.Name {
						err := errors.New("overrideExpected cannot change the name or namespace of expected")
						logger.Error(err, "Returning error cause creating resource could fail if it exists. Override should only provide inner content changes.")
						return err
					}
				}
				err = CreateResource(&config.Instance, expected, config.Client, config.Scheme, logger)
				if err != nil {
					if apierrors.IsAlreadyExists(err) {
						// Wrap the error with additional information about a possible cache issue
						return fmt.Errorf("failed to create resource: %w. Most likely configured client has such cache'ing that does not allow reading this resource, check k8s client configuration", err)
					}
					return err
				}
				return nil
			}
			logger.Info("Resource does not exist, creation not managed by the operator")
			return nil
		}
		return fmt.Errorf("failed to get resource: %w", err)
	}

	updateStatus, err := checkFound(config, found, expected, logger)
	shouldRunAdditionalActions := !updateStatus.IsInDesiredState
	if err != nil {
		return err
	}
	if updateStatus.IsInDesiredState {
		if found.GetLabels()[CachingLabelKey] != CachingLabelValue {
			updateStatus.IsInDesiredState = false
			updateStatus.IsPatchSufficient = false
		}
	}
	if !updateStatus.IsInDesiredState {
		if updateStatus.IsPatchSufficient && patchFound != nil {
			patch, err := patchFound(config, found, logger)
			if err != nil {
				return err
			}
			if err = config.Patch(context.TODO(), found, patch); err != nil {
				return err
			}
		} else if mergeIntoNew != nil {
			merged, ignoreUpdateErr, err := mergeIntoNew(found, expected)
			if err != nil {
				return err
			}
			EnsureCachingLabel(merged)
			err = TryUpdateOrRecreateResource(&config.Instance, merged, config.Client, config.Scheme, logger, ignoreUpdateErr)
			if err != nil {
				return err
			}

			// This is only for ReconcileCredentialsSecrets at the moment, only runs when the secret's data was changed
			if runAdditionalActions != nil {
				if err := runAdditionalActions(shouldRunAdditionalActions, config, logger); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

func EnsureCachingLabel(o client.Object) {
	if o.GetLabels() == nil {
		o.SetLabels(map[string]string{})
	}
	o.GetLabels()[CachingLabelKey] = CachingLabelValue
}

func CreateResource(owner metav1.Object, expected client.Object, c client.Client, scheme *runtime.Scheme, logger logr.Logger) error {
	logger = logger.WithValues("CreateResource", "performingAction")
	err := controllerutil.SetControllerReference(owner, expected, scheme)
	if err != nil {
		return fmt.Errorf("failed to set controller reference on expected resource: %w", err)
	}
	err = c.Create(context.TODO(), expected)
	if err != nil {
		return fmt.Errorf("failed to create new resource: %w", err)
	}
	logger.Info("Created resource successfully")
	return nil
}

func TryUpdateOrRecreateResource(owner metav1.Object, res client.Object, c client.Client, scheme *runtime.Scheme, logger logr.Logger, ignoreUpdateErr bool) error {
	logger = logger.WithValues("TryUpdateOrRecreateResource", "performingAction")
	// There was an issue when htpasswd secret was loosing owner ref after second reconciliation
	if len(res.GetOwnerReferences()) == 0 {
		if err := controllerutil.SetControllerReference(owner, res, scheme); err != nil {
			return fmt.Errorf("failed to set controller reference on expected resource: %w", err)
		}
	}
	err := c.Update(context.TODO(), res)
	if err != nil {
		if !ignoreUpdateErr {
			logger.Error(err, "Error while performing client update.")
		}
		err = DeleteResource(c, res, logger)
		if err != nil {
			return err
		}
		err = CreateResource(owner, res, c, scheme, logger)
		if err != nil {
			return err
		}
	}
	logger.Info("Updated resource")
	return nil
}

func DeleteResource(c client.Client, res client.Object, logger logr.Logger) error {
	logger = logger.WithValues("DeleteResource", "performingAction")
	err := c.Delete(context.TODO(), res)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return fmt.Errorf("failed to delete resource: %w", err)
		}
		return nil
	}
	logger.Info("Deleted resource successfully")
	return nil
}

func GetSpecLabels(instance apiv1alpha1.IBMLicenseServiceReporter) map[string]string {
	if instance.Spec.Labels != nil {
		return instance.Spec.Labels
	}

	return make(map[string]string)
}

func GetSpecAnnotations(instance apiv1alpha1.IBMLicenseServiceReporter) map[string]string {
	if instance.Spec.Annotations != nil {
		return instance.Spec.Annotations
	}

	return make(map[string]string)
}

/*
RestartReporterOperandPod finds all the running LSR operand pods and deletes them, forcing the deployment to
run the new operand pod. Useful if due to other resources changes the operand pod needs to be recreated to
pick up the latest changes during initialization, even thought the deployment itself doesn't need to change
*/
func RestartReporterOperandPod(shouldRun bool, config IBMLicenseServiceReporterConfig, logger logr.Logger) error {
	if !shouldRun {
		return nil
	}

	podList := corev1.PodList{}
	opts := client.ListOptions{
		LabelSelector: labels.SelectorFromSet(LabelsForSelector(config.Instance.Name)),
	}

	if err := config.Client.List(context.TODO(), &podList, &opts); err != nil {
		if apierrors.IsNotFound(err) {
			logger.Info("no Reporter pods found for restart")
			return nil
		}
		return fmt.Errorf("could not list License Service Reporter pods for restart: %w", err)
	}

	if len(podList.Items) == 0 {
		return nil
	}

	logger.Info("restarting Reporter pod")

	reporterPod := podList.Items[0]
	if err := config.Client.Delete(context.TODO(), &reporterPod); err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return fmt.Errorf("could not delete %s pod: %w", reporterPod.Name, err)
	}

	return nil
}
