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

package pvc

import (
	"context"
	"fmt"

	"github.com/go-logr/logr"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	storagev1 "k8s.io/api/storage/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const PersistentVolumeClaimName = "license-service-reporter-pvc"

func ReconcilePersistentVolumeClaim(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	expectedPVC, err := GetPersistentVolumeClaim(logger, config)
	if err != nil {
		return err
	}
	logger = AddResourceValuesToLog(logger, &expectedPVC)

	isOnlyRequestChanged := false
	isOnlyRequestChangedPtr := &isOnlyRequestChanged

	return ReconcileResource(
		config,
		&expectedPVC,
		&corev1.PersistentVolumeClaim{},
		true,
		nil,
		func(_ IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, _ logr.Logger) (ResourceUpdateStatus, error) {
			isDesired, isOnlyRequestChanged, err := IsPVCInDesiredState(found, expected)
			*isOnlyRequestChangedPtr = isOnlyRequestChanged
			if !isDesired || err != nil {
				return ResourceUpdateStatus{IsInDesiredState: false}, err
			}

			// Spec.labels support for resource updates
			if !MapHasAllPairsFromOther(found.GetLabels(), GetSpecLabels(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			// Spec.annotations support for resource updates
			if !MapHasAllPairsFromOther(found.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			return ResourceUpdateStatus{IsInDesiredState: true}, nil
		},
		PatchFoundWithSpecLabelsAndAnnotations,
		func(found FoundObject, expected ExpectedObject) (client.Object, bool, error) {
			return MergePvcIntoNew(found, expected, isOnlyRequestChangedPtr, logger, config)
		},
		logger,
		nil,
	)
}

func MergePvcIntoNew(
	found FoundObject,
	expected ExpectedObject,
	isOnlyRequestChangedPtr *bool,
	logger logr.Logger,
	config IBMLicenseServiceReporterConfig,
) (client.Object, bool, error) {
	foundPvc := found.(*corev1.PersistentVolumeClaim)
	expectedPvc := expected.(*corev1.PersistentVolumeClaim)

	if *isOnlyRequestChangedPtr {
		foundPvc.Spec.Resources.Requests = expectedPvc.Spec.Resources.Requests
	} else {
		foundPvc.Spec = expectedPvc.Spec
	}
	foundPvc.SetLabels(expectedPvc.GetLabels())
	foundPvc.SetAnnotations(expectedPvc.GetAnnotations())

	deployment := v1.Deployment{}
	deployment.SetName(GetDefaultResourceName(config.Instance.GetName()))
	deployment.SetNamespace(foundPvc.GetNamespace())
	err := DeleteResource(config.Client, &deployment, AddResourceValuesToLog(logger, &deployment))
	return foundPvc, false, err
}

func getStorageClass(logger logr.Logger, r client.Reader) (string, error) {
	var defaultSC []string

	scList := &storagev1.StorageClassList{}
	logger.Info("getStorageClass")
	err := r.List(context.TODO(), scList)
	if err != nil {
		return "", err
	}
	if len(scList.Items) == 0 {
		return "", fmt.Errorf("could not find storage class in the cluster")
	}

	for _, sc := range scList.Items {
		if sc.Provisioner == "kubernetes.io/no-provisioner" {
			continue
		}
		if sc.ObjectMeta.GetAnnotations()["storageclass.kubernetes.io/is-default-class"] == "true" {
			defaultSC = append(defaultSC, sc.GetName())
			continue
		}
	}

	if len(defaultSC) != 0 {
		logger.Info("StorageClass configuration", "Name", defaultSC[0])
		return defaultSC[0], nil
	}

	return "", fmt.Errorf("could not find dynamic provisioner default storage class in the cluster")
}

func GetPersistentVolumeClaim(logger logr.Logger, config IBMLicenseServiceReporterConfig) (corev1.PersistentVolumeClaim, error) {
	instance := config.Instance
	spec := instance.Spec
	capacity := spec.Capacity
	if capacity.IsZero() {
		size1Gi := resource.NewQuantity(1024*1024*1024, resource.BinarySI)
		capacity = *size1Gi
	}
	storageClass := spec.StorageClass
	if storageClass == "" {
		var err error
		storageClass, err = getStorageClass(logger, config.Client)
		if err != nil {
			logger.Error(err, "Failed to get StorageCLass for IBM License Service Reporter")
			return corev1.PersistentVolumeClaim{}, err
		}
	}
	return corev1.PersistentVolumeClaim{
		ObjectMeta: metav1.ObjectMeta{
			Name:        PersistentVolumeClaimName,
			Namespace:   instance.GetNamespace(),
			Labels:      LabelsForMeta(instance),
			Annotations: GetSpecAnnotations(instance),
		},
		Spec: corev1.PersistentVolumeClaimSpec{
			StorageClassName: &storageClass,
			AccessModes: []corev1.PersistentVolumeAccessMode{
				corev1.ReadWriteOnce,
			},
			Resources: corev1.ResourceRequirements{
				Requests: corev1.ResourceList{
					corev1.ResourceStorage: capacity,
				},
			},
		},
	}, nil
}

// IsPVCInDesiredState returns isDesired, isOnlyRequestChanged, error
// TODO: Add spec.labels support (check works as expected with both LSR and LS)
func IsPVCInDesiredState(found FoundObject, expected ExpectedObject) (bool, bool, error) {
	foundPVC := found.(*corev1.PersistentVolumeClaim)
	expectedPVC := expected.(*corev1.PersistentVolumeClaim)

	// Compare StorageClassName
	if foundPVC.Spec.StorageClassName == nil || expectedPVC.Spec.StorageClassName == nil ||
		*foundPVC.Spec.StorageClassName != *expectedPVC.Spec.StorageClassName {
		return false, false, nil
	}

	// Compare AccessModes
	if !UnorderedEqualSlice(expectedPVC.Spec.AccessModes, foundPVC.Spec.AccessModes) {
		return false, false, nil
	}

	// Compare Resource Requests
	for resourceName, expectedQuantity := range expectedPVC.Spec.Resources.Requests {
		foundQuantity, ok := foundPVC.Spec.Resources.Requests[resourceName]
		if !ok || !expectedQuantity.Equal(foundQuantity) {
			return false, true, nil
		}
	}

	return true, true, nil
}
