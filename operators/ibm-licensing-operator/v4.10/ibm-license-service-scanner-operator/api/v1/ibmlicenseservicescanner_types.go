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

package v1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IBMLicenseServiceScannerSpec defines the desired state of IBMLicenseServiceScanner
// +kubebuilder:pruning:PreserveUnknownFields
type IBMLicenseServiceScannerSpec struct {

	// Custom labels
	// +optional
	Labels map[string]string `json:"labels,omitempty"`

	// Custom annotations
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// Controls logger's verbosity, options: DEBUG, INFO
	// +kubebuilder:validation:Enum=DEBUG;INFO
	// +optional
	LogLevel string `json:"log-level,omitempty"`

	// Configuration of the scanner's cron job
	// +kubebuilder:validation:Required
	Scan IBMLicenseServiceScannerConfig `json:"scan"`

	// Enabling collection of Instana metrics
	// +optional
	EnableInstanaMetricCollection bool `json:"enableInstanaMetricCollection,omitempty"`

	// Reference pointing to the license service API secret with a valid url, token, and certificate
	LicenseServiceUploadSecret string `json:"license-service-upload-secret"`

	// Reference pointing to the secret enabling pulling images from the image registry
	RegistryPullSecret string `json:"registry-pull-secret"`

	// List of registries which method of authentication is other than license-service-upload-secret secret.
	// +optional
	Registries []RegistryDetails `json:"registries,omitempty"`

	// IBM License Service Scanner license acceptance
	// +kubebuilder:validation:Required
	License *License `json:"license"`

	// Container configuration of the operand (scanner)
	// +optional
	Container *Container `json:"container,omitempty"`
}

// Container configuration of the operand (scanner)
type Container struct {

	// Configure scanner's resource requirements
	// +optional
	Resources ResourceRequirementsNoClaims `json:"resources,omitempty"`

	// Set scanner's image pull policy
	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"image-pull-policy,omitempty"`

	// Set scanner's image pull secrets
	// +optional
	ImagePullSecrets []string `json:"image-pull-secrets,omitempty"`

	// Set scanner's image registry prefix
	// +optional
	ImagePullPrefix string `json:"image-pull-prefix,omitempty"`
}

/*
ResourceRequirementsNoClaims copied from corev1.ResourceRequirements, but without Claims support.

Claims support currently breaks bundle generation, and as such is not supported.
*/
type ResourceRequirementsNoClaims struct {
	// Limits describes the maximum amount of compute resources allowed.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Limits corev1.ResourceList `json:"limits,omitempty" protobuf:"bytes,1,rep,name=limits,casttype=ResourceList,castkey=ResourceName"`
	// Requests describes the minimum amount of compute resources required.
	// If Requests is omitted for a container, it defaults to Limits if that is explicitly specified,
	// otherwise to an implementation-defined value.
	// More info: https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// +optional
	Requests corev1.ResourceList `json:"requests,omitempty" protobuf:"bytes,2,rep,name=requests,casttype=ResourceList,castkey=ResourceName"`
}

// License must be accepted to allow creation of IBMLicenseServiceScanner instances
type License struct {
	// Accept the license terms: ibm.biz/lsvc-lic
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self == true", message="Please accept the license terms (ibm.biz/lsvc-lic) by setting the field \"spec.license.accept: true\""
	Accept bool `json:"accept"`
}

// RegistryDetails with host address, authentication method and auth credentials
type RegistryDetails struct {
	// Name of container registry
	// +kubebuilder:validation:Required
	Name string `json:"name"`

	// URL of registry host
	// +kubebuilder:validation:Required
	Host string `json:"host"`

	// Username used for login to container registry
	// +kubebuilder:validation:Required
	Username string `json:"username"`

	// Authentication method. For now, only supported method is VAULT
	// +kubebuilder:validation:Enum=VAULT
	// +kubebuilder:validation:Required
	AuthMethod string `json:"auth-method"`

	// Details for Vault authentication
	// +kubebuilder:validation:Optional
	VaultDetails VaultDetails `json:"vault"`
}

// VaultDetails required for authentication
type VaultDetails struct {
	// Vault API URL used to authenticate in Vault
	// +kubebuilder:validation:Required
	LoginURL string `json:"login-url"`

	// URL pointing to Vault secret which contains key, value pairs of registry pull secret data
	// +kubebuilder:validation:Required
	SecretURL string `json:"secret-url"`

	// Key under which registry pull secret is stored in Vault secret
	// +kubebuilder:validation:Required
	Key string `json:"key"`

	// Certificate to allow HTTPS (SSL secured) connection with Vault API
	// +kubebuilder:validation:Required
	Cert string `json:"cert"`

	// Role created in Vault with permission to read registry pull secret stored in Vault secret
	// +kubebuilder:validation:Required
	Role string `json:"role"`

	// Name of ServiceAccount with configured access to Vault
	// +kubebuilder:validation:Required
	ServiceAccount string `json:"service-account"`
}

// IBMLicenseServiceScannerConfig defines the configuration for scanning, directly affects the cron job creation
type IBMLicenseServiceScannerConfig struct {

	// Namespaces to be scanned
	// +kubebuilder:validation:Required
	// +kubebuilder:validation:XValidation:rule="self.size()>0", message="You must provide at least one namespace to be scanned"
	Namespaces []string `json:"namespaces"`

	// Frequency of the scans in the cron job format
	// +kubebuilder:validation:Pattern:=`(@(annually|yearly|monthly|weekly|daily|midnight|hourly))|((((\d+,)+\d+|(\d+(\/|-)\d+)|\d+|\*) ?){5,7})`
	// +optional
	Frequency string `json:"frequency,omitempty"`

	// Set to true to suspend the cron job
	// +optional
	Suspend bool `json:"suspend,omitempty"`

	// Set the "expiry" period of scheduled but not started jobs, in seconds
	// +optional
	StartingDeadlineSeconds int64 `json:"startingDeadlineSeconds,omitempty"`
}

// IBMLicenseServiceScannerStatus defines the observed state of IBMLicenseServiceScanner
type IBMLicenseServiceScannerStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// IBMLicenseServiceScanner is the Schema for the ibmlicenseservicescanners API
type IBMLicenseServiceScanner struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMLicenseServiceScannerSpec   `json:"spec,omitempty"`
	Status IBMLicenseServiceScannerStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// IBMLicenseServiceScannerList contains a list of IBMLicenseServiceScanner
type IBMLicenseServiceScannerList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBMLicenseServiceScanner `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IBMLicenseServiceScanner{}, &IBMLicenseServiceScannerList{})
}
