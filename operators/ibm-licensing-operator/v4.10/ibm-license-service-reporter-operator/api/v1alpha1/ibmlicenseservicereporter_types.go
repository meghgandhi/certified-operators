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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// IBMLicenseServiceReporterSpec defines the desired state of IBMLicenseServiceReporter
// +kubebuilder:pruning:PreserveUnknownFields
type IBMLicenseServiceReporterSpec struct {
	// Environment variable setting
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Environment variable setting",xDescriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	// +optional
	EnvVariable map[string]string `json:"envVariable,omitempty"`
	// Labels to be copied into all relevant resources
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Labels",xDescriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	// +optional
	Labels map[string]string `json:"labels,omitempty"`
	// Annotations to be copied into all relevant resources
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations",xDescriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
	// Receiver Settings
	ReceiverContainer Container `json:"receiverContainer,omitempty"`
	// Receiver Settings
	ReporterUIContainer Container `json:"reporterUIContainer,omitempty"`
	// Database Settings
	DatabaseContainer Container `json:"databaseContainer,omitempty"`
	// Authentication Image Settings
	AuthContainer Container `json:"authContainer,omitempty"`
	// License Service Reporter oauth2-proxy configuration
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Authentication",xDescriptors="urn:alm:descriptor:com.tectonic.ui:hidden"
	Authentication Authentication `json:"authentication,omitempty"`
	// IBM License Service Reporter license acceptance.
	License *License `json:"license"`
	// Should application pod show additional information, options: DEBUG, INFO, VERBOSE
	// +kubebuilder:validation:Enum=DEBUG;INFO;VERBOSE
	LogLevel string `json:"logLevel,omitempty"`
	// Secret name used to store application token, either one that exists, or one that will be created
	APISecretToken string `json:"apiSecretToken,omitempty"`
	// Array of pull secrets which should include existing at InstanceNamespace secret to allow pulling IBM License Service Reporter image
	ImagePullSecrets []string `json:"imagePullSecrets,omitempty"`
	// options: self-signed or custom
	// +kubebuilder:validation:Enum=self-signed;custom;ocp
	HTTPSCertsSource string `json:"httpsCertsSource,omitempty"`
	// Information whether to create the Routes automatically by the operator (available only on OpenShift) to expose the IBM License Service Reporter Console and API
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Route Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	RouteEnabled *bool `json:"routeEnabled,omitempty"`
	// Route parameters
	RouteOptions *IBMLicenseServiceRouteOptions `json:"routeOptions,omitempty"`
	// Version
	Version string `json:"version,omitempty"`
	// Storage class used by database to provide persistency
	StorageClass string `json:"storageClass,omitempty"`
	// Persistent Volume Claim Capacity
	Capacity resource.Quantity `json:"capacity,omitempty" protobuf:"bytes,2,opt,name=capacity"`

	// Enable workloads-related processing and reportering in the License Service Reporter
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Enable workloads processing",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch", "urn:alm:descriptor:com.tectonic.ui:hidden"}
	// +optional
	EnableWorkloadsProcessing bool `json:"enableWorkloadsProcessing,omitempty"`

	// Number of days that the deleted custom column values will be stored in database before removing them permamently. Default value is 90.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Custom columns retention period in days",xDescriptors="urn:alm:descriptor:com.tectonic.ui:number"
	// +optional
	WorkloadsCustomColumnsRetencyDays *int `json:"workloadsCustomColumnsRetencyDays,omitempty"`

	// Enabling collection of Instana metrics
	// +optional
	EnableInstanaMetricCollection bool `json:"enableInstanaMetricCollection,omitempty"`

	// Information whether to create the Ingress automatically by the operator to expose the IBM License Service Reporter Console and API (not needed if Routes creation is enabled, disable if you want to create Ingresses manually)
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingress Enabled",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	IngressEnabled bool `json:"ingressEnabled,omitempty"`

	// If ingress is enabled, you can set its parameters
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Ingress Options",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	IngressOptions *IBMLicenseServiceReporterIngressOptions `json:"ingressOptions,omitempty"`
}

type IBMLicenseServiceReporterIngressOptions struct {
	// Options common to both API and Console Ingresses
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API Ingress Options",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	CommonOptions *IBMLicenseServiceReporterIngressCommonOptions `json:"commonOptions,omitempty"`

	// Options specific to API Ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="API Ingress Options",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	ApiOptions *IBMLicenseServiceReporterIngressSpecificOptions `json:"apiOptions,omitempty"`

	// Options specific to Console Ingress
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Console Ingress Options",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	ConsoleOptions *IBMLicenseServiceReporterIngressSpecificOptions `json:"consoleOptions,omitempty"`
}

type IBMLicenseServiceReporterIngressCommonOptions struct {
	// If you use non-default host include it here
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Host",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	Host *string `json:"host,omitempty"`

	// IngressClassName defines ingress class name option to be passed to the ingress spec field
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="IngressClassName",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	IngressClassName *string `json:"ingressClassName,omitempty"`
}

type IBMLicenseServiceReporterIngressSpecificOptions struct {
	// Endpoint under which the application will be available e.g. if you specify this path as /ibm-license-service-reporter-api then the application will be available under https://<hostname>:<port>/ibm-license-service-reporter-api
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Path",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	Path *string `json:"path,omitempty"`

	// Additional annotations that should include r.g. ingress class if using non-default ingress controller
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Annotations",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// TLS options to enable secure connection
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="TLS",xDescriptors="urn:alm:descriptor:com.tectonic.ui:text"
	// +optional
	TLS []networkingv1.IngressTLS `json:"tls,omitempty"`
}

// IBMLicenseServiceReporterStatus defines the observed state of IBMLicenseServiceReporter
type IBMLicenseServiceReporterStatus struct {
	// Important: Run "operator-sdk generate k8s" to regenerate code after modifying this file
	// Add custom validation using kubebuilder tags: https://book-v1.book.kubebuilder.io/beyond_basics/generating_crd.html
	// +optional
	LicenseServiceReporterPods []corev1.PodStatus `json:"LicenseServiceReporterPods,omitempty"`

	// Property for compatibility with LicenseService LTSR
	// +optional
	LicensingReporterPods []corev1.PodStatus `json:"LicensingReporterPods,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

/*
	IBM License Service Reporter is a singleton which may be used in a single or multi-cluster environment. It aggregates data pushed from IBM License Services, deployed on clusters and from ILMT.

	Documentation: https://ibm.biz/lsvc-rprtr.

	License: Please refer to the IBM Terms website (ibm.biz/lsvc-lic) to find the license terms for the particular IBM product for which you are deploying this component.

	IBM License Service Reporter is a free, optionally installed add-on – one of services of Cloud Pak Foundational Services. Thanks to the IBM License Service Reporter customer can:

		- see IBM software deployments on a single dashboard, verify and maintain license compliance and avoid audit infractions,

		- see historical data on IBM software deployments to support making informed decisions for future purchases,

		- see details of software deployments, including source (ILMT for VMs, License Service for containers), cluster and quantities of licenses deployed,

		- identify which software deployments are VMs vs. containers and can use this information to evaluate workloads for modernization.
*/
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=ibmlicenseservicereporters,scope=Namespaced
type IBMLicenseServiceReporter struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   IBMLicenseServiceReporterSpec   `json:"spec,omitempty"`
	Status IBMLicenseServiceReporterStatus `json:"status,omitempty"`
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// IBMLicenseServiceReporterList contains a list of IBMLicenseServiceReporter
type IBMLicenseServiceReporterList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []IBMLicenseServiceReporter `json:"items"`
}

func init() {
	SchemeBuilder.Register(&IBMLicenseServiceReporter{}, &IBMLicenseServiceReporterList{})
}

func (spec *IBMLicenseServiceReporterSpec) IsDebug() bool {
	return spec.LogLevel == "DEBUG"
}

func (spec *IBMLicenseServiceReporterSpec) IsVerbose() bool {
	return spec.LogLevel == "VERBOSE"
}

func (spec *IBMLicenseServiceReporterSpec) IsBasicAuthEnabled() bool {
	return spec.Authentication.Useradmin.Enabled
}

func (spec *IBMLicenseServiceReporterSpec) IsOAuthEnabled() bool {
	return spec.Authentication.OAuth.Enabled
}

func (spec *IBMLicenseServiceReporterSpec) IsRouteEnabled() bool {
	return spec.RouteEnabled != nil && *spec.RouteEnabled
}
