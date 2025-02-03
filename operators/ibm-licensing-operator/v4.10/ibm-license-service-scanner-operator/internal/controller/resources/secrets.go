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
	"fmt"
	odlm "github.com/IBM/operand-deployment-lifecycle-manager/api/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	licenseServiceUploadTokenName  = "ibm-licensing-upload-token"
	licenseServiceUploadConfigName = "ibm-licensing-upload-config"
)

/*
LicenseServiceUploadSecret enables uploading data from scanner to the License Service.

URL is the License Service's API url. The token and the certificate are directly related to authentication.
*/
type LicenseServiceUploadSecret struct {
	OperandRequest *LicenseServiceOperandRequest
	BaseReconcilableResource
}

func (s *LicenseServiceUploadSecret) Init() error {
	s.Logger.Info("Initializing resource")

	// Initialize secret with empty data values in case it doesn't yet exist
	s.ExpectedResource = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: s.Config.Scanner.APIVersion,
				Kind:       s.Config.Scanner.Kind,
				Name:       s.Config.Scanner.Name,
				UID:        s.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      s.GetBaseLabels(),
			Annotations: s.GetBaseAnnotations(),
		},
		StringData: map[string]string{
			"url":     "",
			"token":   "",
			"crt.pem": "",
		},
	}

	// Initialize empty secret to populate later
	s.ActualResource = &corev1.Secret{}

	return nil
}

func (s *LicenseServiceUploadSecret) Reconcile() (ctrl.Result, error) {
	s.Logger.Info("Reconciling secret")

	return ReconcileResource(s)
}

func (s *LicenseServiceUploadSecret) PopulateExpectedFromActual() {
	s.BaseReconcilableResource.PopulateExpectedFromActual()

	// Persist string data to avoid resetting already configured secrets on updates
	actualResourceData := s.ActualResource.(*corev1.Secret).Data
	for key, value := range actualResourceData {
		s.ExpectedResource.(*corev1.Secret).StringData[key] = string(value)
	}
}

/*
MarkShouldUpdate checks if an update is needed to configure a connection to License Service.

Once the License Service connection operand request is in the "running" state, the operator checks if the upload
secret's data is outdated and requests an update if needed.
*/
func (s *LicenseServiceUploadSecret) MarkShouldUpdate() error {
	if err := s.BaseReconcilableResource.MarkShouldUpdate(); err != nil {
		return err
	}

	// Return early if update set by parent function already
	if s.status.ShouldUpdate {
		return nil
	}

	// Return early if OperandRequest CRD is not present
	if s.OperandRequest == nil {
		return nil
	}

	if s.OperandRequest.GetRequestPhase() == odlm.ServiceRunning {
		currentData := s.ActualResource.(*corev1.Secret).Data

		token, err := s.getLicenseServiceUploadToken()
		if err != nil {
			return err
		}

		url, cert, err := s.getLicenseServiceUploadURLAndCert()
		if err != nil {
			return err
		}

		reconciledData := map[string]string{
			"url":     url,
			"token":   token,
			"crt.pem": cert,
		}

		if s.isDataOutdated(currentData, reconciledData) {
			s.ExpectedResource.(*corev1.Secret).StringData = reconciledData
			s.status.ShouldUpdate = true
		}
	}

	return nil
}

/*
Get string token from the License Service upload secret.

Required to configure the connection to License Service.
*/
func (s *LicenseServiceUploadSecret) getLicenseServiceUploadToken() (token string, err error) {
	secret := &corev1.Secret{}

	if err := s.Config.Client.Get(s.Config.Context, types.NamespacedName{
		Name:      licenseServiceUploadTokenName,
		Namespace: s.ExpectedResource.GetNamespace(),
	}, secret); err != nil {
		return "", err
	}

	tokenBytes, tokenExists := secret.Data["token-upload"]
	token = string(tokenBytes)

	if !tokenExists {
		return "", fmt.Errorf("token data missing from %s", licenseServiceUploadTokenName)
	}

	return token, nil
}

/*
Get string url and certificate from the License Service upload config map.

Required to configure the connection to License Service.
*/
func (s *LicenseServiceUploadSecret) getLicenseServiceUploadURLAndCert() (url, cert string, err error) {
	configMap := &corev1.ConfigMap{}

	if err := s.Config.Client.Get(s.Config.Context, types.NamespacedName{
		Name:      licenseServiceUploadConfigName,
		Namespace: s.ExpectedResource.GetNamespace(),
	}, configMap); err != nil {
		return "", "", err
	}

	url, urlExists := configMap.Data["url"]
	cert, certExists := configMap.Data["crt.pem"]

	if !(urlExists && certExists) {
		return "", "", fmt.Errorf("url and/or certificate data missing from %s", licenseServiceUploadConfigName)
	}

	return url, cert, nil
}

/*
Check if the License Service connection data present in the upload secret is outdated.
*/
func (s *LicenseServiceUploadSecret) isDataOutdated(
	currentData map[string][]byte,
	reconciledData map[string]string,
) bool {
	return !(string(currentData["token"]) == reconciledData["token"] &&
		string(currentData["url"]) == reconciledData["url"] &&
		string(currentData["crt.pem"]) == reconciledData["crt.pem"])
}

/*
RegistryPullSecret enables pulling images from the image registry.

The secret's value is meant to be populated from a JSON file.
*/
type RegistryPullSecret struct {
	BaseReconcilableResource
}

func (s *RegistryPullSecret) Init() error {
	s.Logger.Info("Initializing resource")

	// Initialize secret with empty data values in case it doesn't yet exist
	s.ExpectedResource = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: s.Config.Scanner.APIVersion,
				Kind:       s.Config.Scanner.Kind,
				Name:       s.Config.Scanner.Name,
				UID:        s.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      s.GetBaseLabels(),
			Annotations: s.GetBaseAnnotations(),
		},
		StringData: map[string]string{
			corev1.DockerConfigJsonKey: "{}",
		},
		Type: corev1.SecretTypeDockerConfigJson,
	}

	// Initialize empty secret to populate later
	s.ActualResource = &corev1.Secret{}

	return nil
}

func (s *RegistryPullSecret) Reconcile() (ctrl.Result, error) {
	s.Logger.Info("Reconciling secret")

	return ReconcileResource(s)
}

func (s *RegistryPullSecret) PopulateExpectedFromActual() {
	s.BaseReconcilableResource.PopulateExpectedFromActual()

	// Persist string data to avoid resetting already configured secrets on updates
	actualResourceData := s.ActualResource.(*corev1.Secret).Data
	for key, value := range actualResourceData {
		s.ExpectedResource.(*corev1.Secret).StringData[key] = string(value)
	}
}

/*
VaultServiceAccountTokenSecret stores JWT token for ServiceAccount with access to VaultAPI.

Token is Kubernetes provided ServiceAccount JWT, bound to specific Service Account.
*/
type VaultServiceAccountTokenSecret struct {
	ServiceAccountName string
	BaseReconcilableResource
}

func (s *VaultServiceAccountTokenSecret) Init() error {
	s.Logger.Info("Initializing resource")

	// Get annotations and bind secret with Service Account
	annotations := s.GetBaseAnnotations()
	annotations[corev1.ServiceAccountNameKey] = s.ServiceAccountName

	// Initialize secret
	s.ExpectedResource = &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.Name,
			Namespace: s.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: s.Config.Scanner.APIVersion,
				Kind:       s.Config.Scanner.Kind,
				Name:       s.Config.Scanner.Name,
				UID:        s.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      s.GetBaseLabels(),
			Annotations: annotations,
		},
		Type: corev1.SecretTypeServiceAccountToken,
	}
	// Initialize empty secret to populate later
	s.ActualResource = &corev1.Secret{}

	return nil
}

func (s *VaultServiceAccountTokenSecret) Reconcile() (ctrl.Result, error) {
	s.Logger.Info("Reconciling secret")

	return ReconcileResource(s)
}
