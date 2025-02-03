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
	"os"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
)

const (
	CacheConfigMapName  = resourceNamePrefix + "cache"
	ScriptConfigMapName = resourceNamePrefix + "vault"
)

/*
CacheConfigMap stores results from previously scanned images, to avoid having to pull and scan them again.
*/
type CacheConfigMap struct {
	BaseReconcilableResource
}

func (m *CacheConfigMap) Init() error {
	m.Logger.Info("Initializing resource")

	// Initialize config map with empty data values in case it doesn't yet exist
	m.ExpectedResource = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: m.Config.Scanner.APIVersion,
				Kind:       m.Config.Scanner.Kind,
				Name:       m.Config.Scanner.Name,
				UID:        m.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      m.GetBaseLabels(),
			Annotations: m.GetBaseAnnotations(),
		},
	}

	// Initialize empty config map to populate later
	m.ActualResource = &corev1.ConfigMap{}

	return nil
}

func (m *CacheConfigMap) Reconcile() (ctrl.Result, error) {
	m.Logger.Info("Reconciling config map")

	return ReconcileResource(m)
}

func (m *CacheConfigMap) PopulateExpectedFromActual() {
	m.BaseReconcilableResource.PopulateExpectedFromActual()

	// Persist data to avoid resetting existing cache on updates
	actualResourceData := m.ActualResource.(*corev1.ConfigMap).Data
	if actualResourceData != nil {
		m.ExpectedResource.(*corev1.ConfigMap).Data = actualResourceData
	}
}

type VaultScriptConfigMap struct {
	BaseReconcilableResource
}

func (m *VaultScriptConfigMap) Init() error {
	m.Logger.Info("Initializing resource")

	script, err := getVaultConnectorScript()
	if err != nil {
		return fmt.Errorf("failed initializing vault script config map: %w", err)
	}

	// Initialize config map with empty data values in case it doesn't yet exist
	m.ExpectedResource = &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      m.Name,
			Namespace: m.Config.Scanner.GetNamespace(),
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion: m.Config.Scanner.APIVersion,
				Kind:       m.Config.Scanner.Kind,
				Name:       m.Config.Scanner.Name,
				UID:        m.Config.Scanner.UID,
				Controller: ptr.To(true),
			}},
			Labels:      m.GetBaseLabels(),
			Annotations: m.GetBaseAnnotations(),
		},
		Data: map[string]string{
			"script.sh": script,
		},
	}

	// Initialize empty config map to populate later
	m.ActualResource = &corev1.ConfigMap{}

	return nil
}

func (m *VaultScriptConfigMap) Reconcile() (ctrl.Result, error) {
	m.Logger.Info("Reconciling config map")

	return ReconcileResource(m)
}

func (m *VaultScriptConfigMap) PopulateExpectedFromActual() {
	m.BaseReconcilableResource.PopulateExpectedFromActual()

	// Persist data to avoid resetting existing cache on updates
	actualResourceData := m.ActualResource.(*corev1.ConfigMap).Data
	if actualResourceData != nil {
		m.ExpectedResource.(*corev1.ConfigMap).Data = actualResourceData
	}
}

/*
Read vault connector script from the file present in the Dockerfile.

It will be used to populate the vault script config map by default.
*/
func getVaultConnectorScript() (string, error) {
	// Check if Vault Connector location is provided
	const vaultScriptEnvVar = "VAULT_CONNECTOR_SCRIPT_PATH"
	vaultScriptFile, found := os.LookupEnv(vaultScriptEnvVar)
	if !found {
		return "", fmt.Errorf("couldn't find environment variable %s,"+
			"please set the variable to path of the vault connection script", vaultScriptEnvVar)
	}

	vaultScript, err := os.ReadFile(vaultScriptFile)
	if err != nil {
		return "", err
	}

	return string(vaultScript), nil
}
