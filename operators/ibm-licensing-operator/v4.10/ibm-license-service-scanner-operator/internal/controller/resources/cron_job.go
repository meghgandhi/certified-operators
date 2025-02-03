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
	"fmt"
	"os"
	"regexp"
	"slices"
	"strings"

	"k8s.io/utils/ptr"

	v1 "github.ibm.com/cloud-license-reporter/ibm-license-service-scanner-operator/api/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	cli "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	ScannerCronJobName               = resourceNamePrefix + "cron-job"
	scannerCronJobServiceAccountName = resourceNamePrefix + "operand-service-account"
	scannerCronJobContainerName      = resourceNamePrefix + "operand-container"
	scannerCronJobInitContainerName  = scannerCronJobContainerName + "-init-container"

	scanNamespacesEnvVarName   = "SCAN_NAMESPACES"
	logLevelEnvVarName         = "LOG_LEVEL"
	dockerConfigEnvVarName     = "DOCKER_CONFIG"
	instanaAgentHostEnvVarName = "INSTANA_AGENT_HOST"

	VaultAuthenticationMethodName = "VAULT"

	registryPullSecretVolumeName       = "auth"
	registryPullSecretVolumeMountPath  = "/opt/scanner/auth" // #nosec G101
	registryPullSecretVolumeConfigPath = "config.json"       // #nosec G101

	licenseServiceUploadSecretVolumeName      = "sender"
	licenseServiceUploadSecretVolumeMountPath = "/opt/scanner/sender" // #nosec G101

	tempDirectoryVolumeName      = "temp"
	tempDirectoryVolumeMountPath = "/tmp"

	vaultVolumeMountPathPrefix = "/opt/scanner/vault/"
	vaultAuthVolumeMountName   = "vault-auth"
	vaultScriptVolumeMountName = "vault-script"
	vaultAuthVolumeMountPath   = vaultVolumeMountPathPrefix + "auth"
	vaultScriptVolumeMountPath = vaultVolumeMountPathPrefix + "script"
	vaultScriptExecPath        = vaultScriptVolumeMountPath + "/script.sh"

	defaultSecretMode int32 = 420
	defaultScriptMode int32 = 0555

	defaultInitContainerRequestsResourceCPU              = "50m"
	defaultInitContainerRequestsResourceMemory           = "32Mi"
	defaultInitContainerRequestsResourceEphemeralStorage = "10Mi"

	defaultInitContainerLimitsResourceCPU              = "100m"
	defaultInitContainerLimitsResourceMemory           = "128Mi"
	defaultInitContainerLimitsResourceEphemeralStorage = "10Mi"
)

/*
ScannerCronJob is responsible for running the operand according to the schedule passed in the CR.

The scanning itself is performed entirely by the operand, the job of the operator is to configure the cron job
and pass resources needed by the operand to the container.
*/
type ScannerCronJob struct {
	BaseReconcilableResource
}

func (j *ScannerCronJob) Init() error {
	j.Logger.Info("Initializing resource")

	// Store unique certificate and service account references to avoid duplication later on (e.g. on volumes creation)
	// Maps will be used as Set, so value doesn't matter.
	registries := j.Config.Scanner.Spec.Registries
	certificates := map[string]bool{}
	serviceAccounts := map[string]bool{}
	for index := range registries {
		if registries[index].AuthMethod == VaultAuthenticationMethodName {
			// If property VaultDetails is not specified, early exit
			if registries[index].VaultDetails == (v1.VaultDetails{}) {
				return fmt.Errorf("missing property spec.registries.vault in registry: %s. "+
					"Please add missing data in Custom Resource YAML", registries[index].Name)
			}
			certificates[registries[index].VaultDetails.Cert] = true
			serviceAccounts[registries[index].VaultDetails.ServiceAccount] = true
		}
	}

	containers, err := j.getContainers()
	if err != nil {
		return fmt.Errorf("failed getting containers: %w", err)
	}

	initContainers, err := j.getScannerInitContainer(serviceAccounts, certificates)
	if err != nil {
		return fmt.Errorf("failed getting init containers: %w", err)
	}

	// Initialize the cron job -> the operand
	j.ExpectedResource = &batchv1.CronJob{
		ObjectMeta: metav1.ObjectMeta{
			Name:      j.Name,
			Namespace: j.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: j.Config.Scanner.APIVersion,
				Kind:       j.Config.Scanner.Kind,
				Name:       j.Config.Scanner.Name,
				UID:        j.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      j.GetBaseLabels(),
			Annotations: j.GetBaseAnnotations(),
		},
		Spec: batchv1.CronJobSpec{
			Schedule:                j.Config.Scanner.Spec.Scan.Frequency,
			Suspend:                 ptr.To(j.Config.Scanner.Spec.Scan.Suspend),
			StartingDeadlineSeconds: ptr.To(j.Config.Scanner.Spec.Scan.StartingDeadlineSeconds),
			ConcurrencyPolicy:       "Forbid",
			JobTemplate: batchv1.JobTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      j.GetBaseLabels(),
					Annotations: j.GetBaseAnnotations(),
				},
				Spec: batchv1.JobSpec{
					Template: corev1.PodTemplateSpec{
						ObjectMeta: metav1.ObjectMeta{
							Labels:      j.GetBaseLabels(),
							Annotations: j.GetBaseAnnotations(),
						},
						Spec: corev1.PodSpec{
							Volumes:            j.getVolumes(serviceAccounts, certificates),
							Containers:         containers,
							InitContainers:     initContainers,
							RestartPolicy:      "OnFailure",
							ServiceAccountName: scannerCronJobServiceAccountName,
							ImagePullSecrets:   j.getImagePullSecrets(),
						},
					},
				},
			},
		},
	}

	// Initialize an empty object to populate later
	j.ActualResource = &batchv1.CronJob{}

	return nil
}

func (j *ScannerCronJob) Reconcile() (ctrl.Result, error) {
	j.Logger.Info("Reconciling cron job")

	return ReconcileResource(j)
}

func (j *ScannerCronJob) MarkShouldUpdate() error {
	expectedContainers := j.ExpectedResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers
	actualContainers := j.ActualResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Containers
	expectedInitContainers := j.ExpectedResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.InitContainers
	actualInitContainers := j.ActualResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.InitContainers

	if err := j.BaseReconcilableResource.MarkShouldUpdate(); err != nil {
		return fmt.Errorf("failed marking should update within the base class: %w", err)
	}

	// Return early if base class already set the flag
	if j.status.ShouldUpdate {
		return nil
	}

	if !j.schedulesEqual() || !j.volumesEqual() || !j.suspendsEqual() || !j.expiresEqual() ||
		!j.containersEqual(expectedContainers, actualContainers) ||
		!j.containersEqual(expectedInitContainers, actualInitContainers) {
		j.status.ShouldUpdate = true
	}

	return nil
}

/*
Check if cron job's schedule is as expected.

Its value is controlled via the CR.
*/
func (j *ScannerCronJob) schedulesEqual() bool {
	expected := j.ExpectedResource.(*batchv1.CronJob)
	actual := j.ActualResource.(*batchv1.CronJob)

	return actual.Spec.Schedule == expected.Spec.Schedule
}

/*
Check if cron job's suspend flag is as expected.

Its value is controlled via the CR.
*/
func (j *ScannerCronJob) suspendsEqual() bool {
	expected := j.ExpectedResource.(*batchv1.CronJob)
	actual := j.ActualResource.(*batchv1.CronJob)

	return actual.Spec.Suspend == expected.Spec.Suspend
}

/*
Check if cron job's startingDeadlineSeconds field is as expected.

Its value is controlled via the CR.
*/
func (j *ScannerCronJob) expiresEqual() bool {
	expected := j.ExpectedResource.(*batchv1.CronJob)
	actual := j.ActualResource.(*batchv1.CronJob)

	return actual.Spec.StartingDeadlineSeconds == expected.Spec.StartingDeadlineSeconds
}

/*
Check if containers are equal.

The following checks are performed:
  - Compare env vars, to see if namespaces to scan and log level are as expected
  - Compare volume mounts, to see if secrets are mounted as expected

The values are controlled via the CR.
*/
func (j *ScannerCronJob) containersEqual(expectedContainers, actualContainers []corev1.Container) bool {
	// Exit early if count of containers is mismatched -> definitely need to update
	if len(expectedContainers) != len(actualContainers) {
		return false
	}

	// Make sure all expected containers can be found, then check if they match the expected state
	for index := range expectedContainers {
		actualContainer := j.findContainerByName(expectedContainers[index].Name, actualContainers)
		if actualContainer == nil || !containersDeepEqual(actualContainer, &expectedContainers[index]) {
			return false
		}
	}

	return true
}

/*
Check if the two given containers are in the same state.
*/
func containersDeepEqual(actual, expected *corev1.Container) bool {
	return equality.Semantic.DeepEqual(expected.Env, actual.Env) &&
		equality.Semantic.DeepEqual(expected.VolumeMounts, actual.VolumeMounts) &&
		equality.Semantic.DeepEqual(expected.Resources, actual.Resources) &&
		expected.ImagePullPolicy == actual.ImagePullPolicy
}

/*
Identify a container by name in the provided list of containers.

Container is returned as nil if it doesn't exist.
*/
func (j *ScannerCronJob) findContainerByName(name string, containers []corev1.Container) *corev1.Container {
	for index := range containers {
		if containers[index].Name == name {
			return &containers[index]
		}
	}

	return nil
}

/*
Check if volumes are equal.

Volume sources are compared, to see if the resource references are matching the expected state
*/
func (j *ScannerCronJob) volumesEqual() bool {
	expectedVolumes := j.ExpectedResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Volumes
	actualVolumes := j.ActualResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Volumes

	// Exit early if count of volumes is mismatched -> definitely need to update
	if len(expectedVolumes) != len(actualVolumes) {
		return false
	}

	// Make sure all expected volumes can be found, then check if they match the expected state
	for index := range expectedVolumes {
		actualVolume := j.findActualVolumeByName(expectedVolumes[index].Name)
		if actualVolume == nil || !volumesDeepEqual(actualVolume, &expectedVolumes[index]) {
			return false
		}
	}

	return true
}

/*
Check if the two given volumes are in the same state.
*/
func volumesDeepEqual(actual, expected *corev1.Volume) bool {
	return equality.Semantic.DeepEqual(actual.VolumeSource, expected.VolumeSource)
}

/*
Identify a volume by name in the cluster cron job resource.

Volume is returned as nil if it doesn't exist.
*/
func (j *ScannerCronJob) findActualVolumeByName(name string) *corev1.Volume {
	actualVolumes := j.ActualResource.(*batchv1.CronJob).Spec.JobTemplate.Spec.Template.Spec.Volumes

	for index := range actualVolumes {
		if actualVolumes[index].Name == name {
			return &actualVolumes[index]
		}
	}

	return nil
}

/*
Get volumes required by the operand.
*/
func (j *ScannerCronJob) getVolumes(serviceAccounts, certificates map[string]bool) []corev1.Volume {
	volumes := []corev1.Volume{
		{
			Name: licenseServiceUploadSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  j.Config.Scanner.Spec.LicenseServiceUploadSecret,
					DefaultMode: ptr.To(defaultSecretMode),
				},
			},
		},
		{
			Name: registryPullSecretVolumeName,
			VolumeSource: corev1.VolumeSource{
				Secret: &corev1.SecretVolumeSource{
					SecretName:  j.Config.Scanner.Spec.RegistryPullSecret,
					DefaultMode: ptr.To(defaultSecretMode),
					Items: []corev1.KeyToPath{{
						Key:  corev1.DockerConfigJsonKey,
						Path: registryPullSecretVolumeConfigPath,
					}},
				},
			},
		},
		{
			Name: tempDirectoryVolumeName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
	}

	return append(volumes, getVaultVolumes(serviceAccounts, certificates)...)
}

/*
Configure operand container.

The namespaces to scan are exposed via an env var, while the other values are exposed via resource volume mounts.
*/
func (j *ScannerCronJob) getContainers() ([]corev1.Container, error) {
	image, err := j.getOperandImage()
	if err != nil {
		return nil, fmt.Errorf("failed getting operand image: %w", err)
	}

	// Default use secret for registries auth or vault (currently the only supported non-secret auth, change in future)
	dockerConfigMountPath := registryPullSecretVolumeMountPath
	if len(j.Config.Scanner.Spec.Registries) != 0 {
		dockerConfigMountPath = vaultAuthVolumeMountPath
	}

	scanNamespaces, err := j.getScanNamespaces()
	if err != nil {
		return nil, fmt.Errorf("could not process scan namespaces from the CR: %w", err)
	}

	if len(scanNamespaces) == 0 {
		return nil, fmt.Errorf("no valid namespaces to scan found")
	}

	containerEnvVars := []corev1.EnvVar{
		{
			Name:  logLevelEnvVarName,
			Value: j.getLogLevel(),
		},
		{
			Name:  scanNamespacesEnvVarName,
			Value: strings.Join(scanNamespaces, ","),
		},
		{
			Name:  dockerConfigEnvVarName,
			Value: dockerConfigMountPath,
		},
	}

	if j.Config.Scanner.Spec.EnableInstanaMetricCollection {
		containerEnvVars = append(
			containerEnvVars,
			corev1.EnvVar{
				Name: instanaAgentHostEnvVarName,
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.hostIP",
					},
				},
			})
	}

	return []corev1.Container{
		{
			Name:  scannerCronJobContainerName,
			Image: image,
			Env:   containerEnvVars,
			SecurityContext: &corev1.SecurityContext{
				AllowPrivilegeEscalation: ptr.To(false),
				Privileged:               ptr.To(false),
				ReadOnlyRootFilesystem:   ptr.To(true),
				RunAsNonRoot:             ptr.To(true),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
				SeccompProfile: &corev1.SeccompProfile{
					Type: corev1.SeccompProfileTypeRuntimeDefault,
				},
				ProcMount: ptr.To(corev1.DefaultProcMount),
			},
			Resources: corev1.ResourceRequirements{
				Requests: j.Config.Scanner.Spec.Container.Resources.Requests,
				Limits:   j.Config.Scanner.Spec.Container.Resources.Limits,
			},
			VolumeMounts:    j.getContainerVolumeMounts(),
			ImagePullPolicy: j.Config.Scanner.Spec.Container.ImagePullPolicy,
		},
	}, nil
}

/*
Get operand image from an env var.

No extra processing is done to verify the image is an actual docker image. An invalid value should just result in
failing to download the image and run the container.
*/
func (j *ScannerCronJob) getOperandImage() (string, error) {
	const operandImageEnvVar = "IBM_LICENSE_SERVICE_SCANNER_OPERAND_IMAGE"

	image, found := os.LookupEnv(operandImageEnvVar)
	if !found || image == "" {
		return "", fmt.Errorf("%s env var must be set and not empty", operandImageEnvVar)
	}

	// Replace image registry if specified in the Scanner CR
	if prefix := j.Config.Scanner.Spec.Container.ImagePullPrefix; prefix != "" {
		parts := strings.Split(image, "/")
		parts[0] = prefix
		image = strings.Join(parts, "/")
	}

	return image, nil
}

func (j *ScannerCronJob) getLogLevel() string {
	if j.Config.Scanner.Spec.LogLevel == "DEBUG" {
		return "DEBUG"
	}
	return "INFO"
}

/*
Returns an alphabetically sorted slice of namespaces, based on the input from the Scanner's CR.

For prefix wildcards (e.g. ns-*) all namespaces from the K8s API will be listed and all matching ones will be added
to the result.

Note: Currently only prefix wildcards with '*' are supported, e.g. ns-*, or sole '*'
*/
func (j *ScannerCronJob) getScanNamespaces() ([]string, error) {
	if len(j.Config.Scanner.Spec.Scan.Namespaces) == 0 {
		return nil, nil
	}

	// Guard regexp to quickly catch unsupported wildcards.
	guardRe := regexp.MustCompile(`\*`)
	var wildcards []string

	// map serves as a set - we don't want duplicate namespaces due to e.g. accidental input
	scanNamespacesSet := map[string]any{}

	for _, ns := range j.Config.Scanner.Spec.Scan.Namespaces {
		// Separate plain namespaces from supported wildcard(s).
		// If user provides special characters without '*', it'll be passed as is.
		if strings.Contains(ns, "*") {
			// Asterisk must be at the end and only one is allowed
			if !strings.HasSuffix(ns, "*") || len(guardRe.FindAllString(ns, -1)) > 1 {
				return []string{}, fmt.Errorf("unsupported wildcard detected: %s", ns)
			}
			wildcards = append(wildcards, ns)
		} else {
			scanNamespacesSet[ns] = nil // onle keys matter
		}
	}

	var scanNamespaces []string
	var err error // declared not to overshadow scanNamespaces with the declaration in the block below
	// Obtain namespaces based on found wildcards
	if len(wildcards) > 0 {
		scanNamespaces, err = getNamespacesFromWildcards(j.Config.Client, wildcards)
		if err != nil {
			return nil, fmt.Errorf("could not obtain namespaces to scan, based on provided wildcards: %w", err)
		}
	}

	// Having sorted results is crucial for proper reconciliation and easier testing.
	// Random order might cause false-positive reconciles, as the result is later concatenated to a single string.
	scanNamespaces = append(scanNamespaces, getAllMapKeys(scanNamespacesSet)...)
	slices.Sort(scanNamespaces)

	return scanNamespaces, nil
}

/*
Returns all namespaces from the cluster that have been matched with provided wildcards.
Function to be used only internally.

Note: Passed wildcards must end with '*'.
*/
func getNamespacesFromWildcards(client cli.Client, wildcards []string) ([]string, error) {
	namespaces := corev1.NamespaceList{}
	err := client.List(context.Background(), &namespaces) // get all namespaces from the cluster
	if err != nil {
		return []string{}, fmt.Errorf("could not list namespaces from the cluster: %w", err)
	}

	scanNamespacesSet := map[string]any{} // used as a set to avoid duplicate namespaces
	// currently only prefix wildcards ending with '*' are supported, e.g. ns-*, or sole '*'
	for _, wildcard := range wildcards {
		// Last character should be '*'.
		// Wildcards need to be escaped from unsupported special characters.
		// Then, supported characters are placed at the beginning and end of the string.
		// This way we make sure only supported wildcards will be used for regexp.
		processedWildcard := "^" + regexp.QuoteMeta(wildcard[:len(wildcard)-1]) + ".*"
		regExpression, err := regexp.Compile(processedWildcard)
		if err != nil {
			return []string{}, fmt.Errorf("namespace wildcard pattern compilation failed: %w", err)
		}

		for idx := range namespaces.Items {
			namespace := namespaces.Items[idx]
			if regExpression.FindString(namespace.Name) != "" {
				scanNamespacesSet[namespace.Name] = nil // only keys matter
			}
		}
	}

	return getAllMapKeys(scanNamespacesSet), nil
}

/*
Return all map keys as a slice.
*/
func getAllMapKeys[K comparable, V any](m map[K]V) []K {
	var keys []K
	for k := range m {
		keys = append(keys, k)
	}

	return keys
}

/*
Get volume mounts for operand container.

Function returns array of volumes that should be mounted.
*/
func (j *ScannerCronJob) getContainerVolumeMounts() []corev1.VolumeMount {
	volumeMounts := []corev1.VolumeMount{
		{
			Name:      registryPullSecretVolumeName,
			MountPath: registryPullSecretVolumeMountPath,
			ReadOnly:  true,
		},
		{
			Name:      licenseServiceUploadSecretVolumeName,
			MountPath: licenseServiceUploadSecretVolumeMountPath,
			ReadOnly:  true,
		},
		{
			Name:      tempDirectoryVolumeName,
			MountPath: tempDirectoryVolumeMountPath,
			ReadOnly:  false,
		},
	}

	// If registries auth is needed, this is where init container would create auth files (so only read-only is needed)
	if len(j.Config.Scanner.Spec.Registries) != 0 {
		volumeMounts = append(volumeMounts, corev1.VolumeMount{
			Name:      vaultAuthVolumeMountName,
			MountPath: vaultAuthVolumeMountPath,
			ReadOnly:  true,
		})
	}

	return volumeMounts
}

/*
Create init container specification for License Service Scanner.

Accepts ServiceAccounts and certificates that should be mounted as volumes to specified init container.
*/
func (j *ScannerCronJob) getScannerInitContainer(
	serviceAccounts, certificates map[string]bool,
) ([]corev1.Container, error) {
	if len(serviceAccounts) == 0 && len(certificates) == 0 {
		return nil, nil
	}

	image, err := j.getOperandImage()
	if err != nil {
		return nil, fmt.Errorf("failed getting operand image: %w", err)
	}

	initContainer := []corev1.Container{
		{
			Name:  scannerCronJobInitContainerName,
			Image: image,
			SecurityContext: &corev1.SecurityContext{
				Privileged:               ptr.To(false),
				RunAsNonRoot:             ptr.To(true),
				ReadOnlyRootFilesystem:   ptr.To(true),
				AllowPrivilegeEscalation: ptr.To(false),
				Capabilities: &corev1.Capabilities{
					Drop: []corev1.Capability{
						"ALL",
					},
				},
			},
			Resources: corev1.ResourceRequirements{
				Requests: getResourcesRequest(),
				Limits:   getResourcesLimits(),
			},
			Command:         []string{vaultScriptExecPath},
			VolumeMounts:    getInitContainersVolumeMounts(serviceAccounts, certificates),
			ImagePullPolicy: j.Config.Scanner.Spec.Container.ImagePullPolicy,
			Env:             getInitContainerEnvVars(j.Config.Scanner.Spec.Registries),
		},
	}
	return initContainer, nil
}

func getInitContainerEnvVars(registries []v1.RegistryDetails) []corev1.EnvVar {
	var envVars []corev1.EnvVar

	for index := range registries {
		saTokenPath := vaultVolumeMountPathPrefix + registries[index].VaultDetails.ServiceAccount
		caCertPath := vaultVolumeMountPathPrefix + registries[index].VaultDetails.Cert
		envVars = append(envVars,
			corev1.EnvVar{Name: fmt.Sprintf("REGISTRY_NAME_%d", index), Value: registries[index].Host},
			corev1.EnvVar{Name: fmt.Sprintf("REGISTRY_USER_%d", index), Value: registries[index].Username},
			corev1.EnvVar{Name: fmt.Sprintf("VAULT_LOGIN_ADDR_%d", index), Value: registries[index].VaultDetails.LoginURL},
			corev1.EnvVar{Name: fmt.Sprintf("VAULT_SECRET_ADDR_%d", index), Value: registries[index].VaultDetails.SecretURL},
			corev1.EnvVar{Name: fmt.Sprintf("SA_TOKEN_%d", index), Value: saTokenPath},
			corev1.EnvVar{Name: fmt.Sprintf("VAULT_ROLE_%d", index), Value: registries[index].VaultDetails.Role},
			corev1.EnvVar{Name: fmt.Sprintf("CERT_PATH_%d", index), Value: caCertPath},
			corev1.EnvVar{Name: fmt.Sprintf("SECRET_KEY_%d", index), Value: registries[index].VaultDetails.Key},
		)
	}

	return envVars
}

func getInitContainersVolumeMounts(serviceAccounts, certificates map[string]bool) []corev1.VolumeMount {
	mounts := []corev1.VolumeMount{
		{
			Name:      vaultAuthVolumeMountName,
			MountPath: vaultAuthVolumeMountPath,
			ReadOnly:  false,
		},
		{
			Name:      vaultScriptVolumeMountName,
			MountPath: vaultScriptVolumeMountPath,
			ReadOnly:  true,
		},
		{
			Name:      registryPullSecretVolumeName,
			MountPath: registryPullSecretVolumeMountPath,
			ReadOnly:  true,
		},
	}

	for certificate := range certificates {
		mounts = append(mounts,
			corev1.VolumeMount{
				Name:      certificate,
				MountPath: vaultVolumeMountPathPrefix + certificate,
				ReadOnly:  true,
			},
		)
	}

	for serviceAccount := range serviceAccounts {
		mounts = append(mounts,
			corev1.VolumeMount{
				Name:      serviceAccount,
				MountPath: vaultVolumeMountPathPrefix + serviceAccount,
				ReadOnly:  true,
			},
		)
	}

	return mounts
}

func getVaultVolumes(serviceAccounts, certificates map[string]bool) []corev1.Volume {
	if len(serviceAccounts) == 0 && len(certificates) == 0 {
		return []corev1.Volume{}
	}

	volumes := []corev1.Volume{
		{
			Name: vaultAuthVolumeMountName,
			VolumeSource: corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			},
		},
		{
			Name: vaultScriptVolumeMountName,
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: ScriptConfigMapName,
					},
					DefaultMode: ptr.To(defaultScriptMode),
				},
			},
		},
	}

	for certificate := range certificates {
		volumes = append(volumes,
			corev1.Volume{
				Name: certificate,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  certificate,
						DefaultMode: ptr.To(defaultSecretMode),
					},
				},
			},
		)
	}

	for serviceAccount := range serviceAccounts {
		volumes = append(volumes,
			corev1.Volume{
				Name: serviceAccount,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  serviceAccount,
						DefaultMode: ptr.To(defaultScriptMode),
					},
				},
			},
		)
	}
	return volumes
}

func getResourcesRequest() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse(defaultInitContainerRequestsResourceCPU),
		corev1.ResourceMemory:           resource.MustParse(defaultInitContainerRequestsResourceMemory),
		corev1.ResourceEphemeralStorage: resource.MustParse(defaultInitContainerRequestsResourceEphemeralStorage),
	}
}

func getResourcesLimits() corev1.ResourceList {
	return corev1.ResourceList{
		corev1.ResourceCPU:              resource.MustParse(defaultInitContainerLimitsResourceCPU),
		corev1.ResourceMemory:           resource.MustParse(defaultInitContainerLimitsResourceMemory),
		corev1.ResourceEphemeralStorage: resource.MustParse(defaultInitContainerLimitsResourceEphemeralStorage),
	}
}

/*
Retrieve image pull secrets as a list of local object references rather than a list of strings.
*/
func (j *ScannerCronJob) getImagePullSecrets() []corev1.LocalObjectReference {
	var references []corev1.LocalObjectReference

	for _, secret := range j.Config.Scanner.Spec.Container.ImagePullSecrets {
		references = append(references, corev1.LocalObjectReference{Name: secret})
	}

	return references
}
