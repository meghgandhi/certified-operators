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

package deployments

import (
	"reflect"

	"github.com/go-logr/logr"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apieq "k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

const OperandServiceAccount = "ibm-license-service-reporter"

func ReconcileDeployment(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	expectedDeployment, err := GetDeployment(config)
	if err != nil {
		return err
	}
	logger = AddResourceValuesToLog(logger, expectedDeployment)
	return ReconcileResource(
		config,
		expectedDeployment,
		&appsv1.Deployment{},
		true,
		nil,
		IsDeploymentInDesiredState,
		PatchDeploymentWithSpecLabelsAndAnnotations,
		MergeDeploymentIntoNew,
		logger,
		nil,
	)
}

func GetServiceAccountName() string {
	return OperandServiceAccount
}

var replicas = int32(1)

func GetDeployment(config IBMLicenseServiceReporterConfig) (*appsv1.Deployment, error) {
	instance := config.Instance
	metaLabels := LabelsForMeta(instance)
	selectorLabels := LabelsForSelector(instance.GetName())
	podLabels := LabelsForPod(instance)

	var imagePullSecrets []corev1.LocalObjectReference
	if instance.Spec.ImagePullSecrets != nil {
		for _, pullSecret := range instance.Spec.ImagePullSecrets {
			imagePullSecrets = append(imagePullSecrets, corev1.LocalObjectReference{Name: pullSecret})
		}
	}

	containerDB, err := GetDatabaseContainer(instance.Spec)
	if err != nil {
		return nil, err
	}

	containerReceiver, err := GetReceiverContainer(config)
	if err != nil {
		return nil, err
	}

	containerUI, err := GetReporterUIContainer(config)
	if err != nil {
		return nil, err
	}

	containerAuth, err := GetOAuthContainer(config)
	if err != nil {
		return nil, err
	}

	containers := []corev1.Container{containerDB, containerReceiver, containerUI, containerAuth}

	var seconds60 int64 = 60
	initContainers, err := getLicenseReporterInitContainers(config)
	if err != nil {
		return nil, err
	}

	volumes, err := GetLicenseServiceReporterVolumes(config)
	if err != nil {
		return &appsv1.Deployment{}, err
	}

	deployment := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:        GetDefaultResourceName(instance.GetName()),
			Namespace:   instance.GetNamespace(),
			Labels:      metaLabels,
			Annotations: GetSpecAnnotations(instance),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: selectorLabels,
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      podLabels,
					Annotations: AnnotationsForPod(instance),
				},
				Spec: corev1.PodSpec{
					Volumes:                       volumes,
					InitContainers:                initContainers,
					Containers:                    containers,
					TerminationGracePeriodSeconds: &seconds60,
					ServiceAccountName:            GetServiceAccountName(),
					ImagePullSecrets:              imagePullSecrets,
					Affinity: &corev1.Affinity{
						NodeAffinity: &corev1.NodeAffinity{
							RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
								NodeSelectorTerms: []corev1.NodeSelectorTerm{
									{
										MatchExpressions: []corev1.NodeSelectorRequirement{
											{
												Key:      "kubernetes.io/arch",
												Operator: corev1.NodeSelectorOpIn,
												Values:   []string{"amd64"},
											},
										},
									},
								},
							},
						},
					},
					Tolerations: []corev1.Toleration{
						{
							Key:      "dedicated",
							Operator: corev1.TolerationOpExists,
							Effect:   corev1.TaintEffectNoSchedule,
						},
						{
							Key:      "CriticalAddonsOnly",
							Operator: corev1.TolerationOpExists,
						},
					},
				},
			},
		},
	}
	return deployment, nil
}

func MergeDeploymentIntoNew(found FoundObject, expected ExpectedObject) (client.Object, bool, error) {
	foundDeployment := found.(*appsv1.Deployment)
	expectedDeployment := expected.(*appsv1.Deployment)

	foundDeployment.Spec = expectedDeployment.Spec
	foundDeployment.SetLabels(expectedDeployment.GetLabels())
	foundDeployment.SetAnnotations(expectedDeployment.GetAnnotations())

	return foundDeployment, false, nil
}

func IsDeploymentInDesiredState(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
	expectedDeployment := expected.(*appsv1.Deployment)
	foundDeployment := found.(*appsv1.Deployment)

	if ShouldUpdateDeployment(&expectedDeployment.Spec.Template, &foundDeployment.Spec.Template, logger) {
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// In case patch sufficient in spec.labels check but not in spec.annotations check (can't return early)
	shouldReturnPatchSufficientAfterAnnotationsCheck := false

	// Spec.labels support for pods, call patch if only spec.labels mismatch
	if !MapHasAllPairsFromOther(foundDeployment.Spec.Template.GetLabels(), expectedDeployment.Spec.Template.GetLabels()) {
		if MapHasAllPairsFromOther(MergeMaps(foundDeployment.Spec.Template.GetLabels(), GetSpecLabels(config.Instance)), expectedDeployment.Spec.Template.GetLabels()) {
			shouldReturnPatchSufficientAfterAnnotationsCheck = true
		}
		logger.Info("Updating " + foundDeployment.GetName() + " due to having outdated pod labels")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// Spec.annotations support for pods, call patch if only spec.annotations mismatch
	if !MapHasAllPairsFromOther(foundDeployment.Spec.Template.GetAnnotations(), expectedDeployment.Spec.Template.GetAnnotations()) {
		if MapHasAllPairsFromOther(MergeMaps(foundDeployment.Spec.Template.GetAnnotations(), GetSpecAnnotations(config.Instance)), expectedDeployment.Spec.Template.GetAnnotations()) {
			return ResourceUpdateStatus{IsPatchSufficient: true}, nil
		}
		logger.Info("Updating " + foundDeployment.GetName() + " due to having outdated pod annotations")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// At this stage safe to return early without subsequent checks
	if shouldReturnPatchSufficientAfterAnnotationsCheck {
		return ResourceUpdateStatus{IsPatchSufficient: true}, nil
	}

	// Spec.labels support for resource updates
	if !MapHasAllPairsFromOther(foundDeployment.GetLabels(), GetSpecLabels(config.Instance)) {
		return ResourceUpdateStatus{IsPatchSufficient: true}, nil
	}

	// Spec.annotations support for resource updates
	if !MapHasAllPairsFromOther(foundDeployment.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
		return ResourceUpdateStatus{IsPatchSufficient: true}, nil
	}

	return ResourceUpdateStatus{IsInDesiredState: true}, nil
}

// PatchDeploymentWithSpecLabelsAndAnnotations attaches labels and annotations to the deployment and the template (pods)
func PatchDeploymentWithSpecLabelsAndAnnotations(
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
		"spec": map[string]interface{}{
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels":      GetSpecLabels(config.Instance),
					"annotations": GetSpecAnnotations(config.Instance),
				},
			},
		},
	}

	return MarshalAndPatch(data, found.GetResourceVersion())
}

// TODO: try optimize with collection operations
func ShouldUpdateDeployment(
	expectedSpec *corev1.PodTemplateSpec,
	foundSpec *corev1.PodTemplateSpec,
	logger logr.Logger,
) bool {
	if !apieq.Semantic.DeepEqual(foundSpec.Spec.Volumes, expectedSpec.Spec.Volumes) {
		logger.Info("Deployment has wrong volumes")
	} else if !apieq.Semantic.DeepEqual(foundSpec.Spec.Affinity, expectedSpec.Spec.Affinity) {
		logger.Info("Deployment has wrong affinity")
	} else if foundSpec.Spec.ServiceAccountName != expectedSpec.Spec.ServiceAccountName {
		logger.Info("Deployment wrong service account name")
	} else if !equalContainerLists(foundSpec.Spec.Containers, expectedSpec.Spec.Containers, logger) {
		logger.Info("Deployment wrong containers")
	} else if !equalContainerLists(foundSpec.Spec.InitContainers, expectedSpec.Spec.InitContainers, logger) {
		logger.Info("Deployment wrong init containers")
	} else {
		return false
	}
	return true
}

func equalContainerLists(containers1, containers2 []corev1.Container, logger logr.Logger) bool {
	if len(containers1) != len(containers2) {
		logger.Info("Deployment has wrong amount of containers")
		return false
	}
	if len(containers1) == 0 {
		return true
	}

	containersToBeChecked := map[*corev1.Container]*corev1.Container{}

	//map container with same names
	for i, container1 := range containers1 {
		foundContainer2 := false
		for j, container2 := range containers2 {
			if container1.Name == container2.Name {
				containersToBeChecked[&containers1[i]] = &containers2[j]
				foundContainer2 = true
				break
			}
		}
		if !foundContainer2 {
			return false
		}
	}

	potentialDifference := false
	// DeepEqual requires same order of items, which results in false negatives, so we use custom comparison functions to verify same contents of slices
	for foundContainer, expectedContainer := range containersToBeChecked {
		if potentialDifference {
			break
		}
		potentialDifference = true
		if foundContainer.Image != expectedContainer.Image {
			logger.Info("Container " + foundContainer.Name + " has wrong container image")
		} else if foundContainer.ImagePullPolicy != expectedContainer.ImagePullPolicy {
			logger.Info("Container " + foundContainer.Name + " has wrong image pull policy")
		} else if !apieq.Semantic.DeepEqual(foundContainer.Command, expectedContainer.Command) {
			logger.Info("Container " + foundContainer.Name + " has wrong container command")
		} else if !apieq.Semantic.DeepEqual(foundContainer.Ports, expectedContainer.Ports) {
			logger.Info("Container " + foundContainer.Name + " has wrong containers ports")
		} else if !apieq.Semantic.DeepEqual(foundContainer.VolumeMounts, expectedContainer.VolumeMounts) {
			logger.Info("Container " + foundContainer.Name + " has wrong VolumeMounts in container")
		} else if !equalEnvVars(foundContainer.Env, expectedContainer.Env) {
			logger.Info("Container " + foundContainer.Name + " has wrong env variables in container")
		} else if !UnorderedEqualSlice(foundContainer.Args, expectedContainer.Args) {
			logger.Info("Container " + foundContainer.Name + " has wrong arguments")
		} else if !apieq.Semantic.DeepEqual(foundContainer.SecurityContext, expectedContainer.SecurityContext) {
			logger.Info("Container " + foundContainer.Name + " has wrong container security context")
		} else if (foundContainer.Resources.Limits == nil) || (foundContainer.Resources.Requests == nil) { // We must have default requests and limits set -> no nils allowed
			logger.Info("Container " + foundContainer.Name + " has wrong container Resources")
		} else if !apieq.Semantic.DeepEqual(expectedContainer.Resources.Limits, foundContainer.Resources.Limits) {
			logger.Info("Container " + foundContainer.Name + " has wrong container resources limits")
		} else if !apieq.Semantic.DeepEqual(expectedContainer.Resources.Requests, foundContainer.Resources.Requests) {
			logger.Info("Container " + foundContainer.Name + " has wrong container resources requests")
		} else if !equalProbes(foundContainer.ReadinessProbe, expectedContainer.ReadinessProbe) {
			logger.Info("Container " + foundContainer.Name + " has wrong container Readiness Probe")
		} else if !equalProbes(foundContainer.LivenessProbe, expectedContainer.LivenessProbe) {
			logger.Info("Container " + foundContainer.Name + " has wrong container Liveness Probe")
		} else {
			potentialDifference = false
		}
	}
	return !potentialDifference
}

func equalProbes(probe1 *corev1.Probe, probe2 *corev1.Probe) bool {
	if probe1 == nil {
		return probe2 == nil
	} else if probe2 == nil {
		return false
	}
	// need to set thresholds for not set values
	if probe1.SuccessThreshold == 0 {
		probe1.SuccessThreshold = probe2.SuccessThreshold
	} else if probe2.SuccessThreshold == 0 {
		probe2.SuccessThreshold = probe1.SuccessThreshold
	}
	if probe1.FailureThreshold == 0 {
		probe1.FailureThreshold = probe2.FailureThreshold
	} else if probe2.FailureThreshold == 0 {
		probe2.FailureThreshold = probe1.FailureThreshold
	}
	return apieq.Semantic.DeepEqual(probe1, probe2)
}

func equalEnvVars(envVarArr1, envVarArr2 []corev1.EnvVar) bool {
	if len(envVarArr1) != len(envVarArr2) {
		return false
	}
	for _, env1 := range envVarArr1 {
		contains := false
		for _, env2 := range envVarArr2 {
			if env1.Name == env2.Name && env1.Value == env2.Value {
				contains = true
				break
			}
		}
		if !contains {
			return contains
		}
	}
	return true
}
