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

package secrets

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/go-logr/logr"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	ReporterCredentialsSecretName = "ibm-license-service-reporter-credentials"
	ReporterHtpasswdSecretName    = "ibm-license-service-reporter-credentials-htpasswd"
	ReporterAuthCookieSecret      = "ibm-license-service-reporter-auth-cookie"
)

func GetReporterCredentialsSecret(namespace string, metaLabels, metaAnnotations map[string]string) (corev1.Secret, error) {
	username, password, err := generateUserAndPassword()
	if err != nil {
		return corev1.Secret{}, err
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ReporterCredentialsSecretName,
			Namespace:   namespace,
			Labels:      metaLabels,
			Annotations: metaAnnotations,
		},
		Type: corev1.SecretTypeBasicAuth,
		Data: map[string][]byte{"username": []byte(username), "password": []byte(password)},
	}

	return secret, nil
}

func GetReporterHtpasswdSecret(namespace string, metaLabels, metaAnnotations map[string]string, credsSecret corev1.Secret) (corev1.Secret, error) {
	rawUsername := credsSecret.Data["username"]
	rawPassword := credsSecret.Data["password"]

	htpasswd, err := generateHtpasswd(string(rawUsername), string(rawPassword))
	if err != nil {
		return corev1.Secret{}, err
	}

	secret := corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ReporterHtpasswdSecretName,
			Namespace:   namespace,
			Labels:      metaLabels,
			Annotations: metaAnnotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"data": []byte(htpasswd)},
	}

	return secret, nil
}

// Reconcile credentials and htpasswd secrets
func ReconcileCredentialsSecrets(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	instance := config.Instance

	expectedCredsSecret, err := GetReporterCredentialsSecret(instance.GetNamespace(), LabelsForMeta(instance), GetSpecAnnotations(instance))
	if err != nil {
		return err
	}

	foundCredsSecret := corev1.Secret{}

	// Reconcile credentials secret
	credsLogger := AddResourceValuesToLog(logger, &expectedCredsSecret)
	if err := ReconcileResource(
		config,
		&expectedCredsSecret,
		&foundCredsSecret,
		true,
		nil,
		// We do not want to overwrite user's choice of credentials
		func(config IBMLicenseServiceReporterConfig, found FoundObject, _ ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
			foundSecret := found.(*corev1.Secret)

			if val, ok := foundSecret.Data["username"]; !ok || bytes.Equal(val, make([]byte, 0)) {
				credsLogger.Info("Updating " + ReporterCredentialsSecretName + " due to missing username")
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}
			if val, ok := foundSecret.Data["password"]; !ok || bytes.Equal(val, make([]byte, 0)) {
				credsLogger.Info("Updating " + ReporterCredentialsSecretName + " due to missing password")
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}

			// Spec.labels support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetLabels(), GetSpecLabels(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			// Spec.annotations support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			return ResourceUpdateStatus{IsInDesiredState: true}, nil
		},
		PatchFoundWithSpecLabelsAndAnnotations,
		OverrideFoundWithExpected,
		credsLogger,
		nil,
	); err != nil {
		return err
	}

	// Wait for credentials secret to be fully created
	retry := 3
	foundCredsSecret = corev1.Secret{}
	for retry > 0 {
		err = config.Client.Get(context.TODO(), types.NamespacedName{Name: ReporterCredentialsSecretName, Namespace: instance.Namespace}, &foundCredsSecret)
		if err != nil {
			if apierrors.IsNotFound(err) {
				retry = retry - 1
				time.Sleep(5 * time.Second)
				continue
			}
			return fmt.Errorf("could not get "+ReporterCredentialsSecretName+" secret: %w", err)
		}
		break
	}

	// If error persists after 3 retires, return it
	if err != nil {
		return fmt.Errorf("could not create "+ReporterCredentialsSecretName+" secret after 3 retires: %w", err)
	}

	expectedPassword := string(foundCredsSecret.Data["password"])
	expectedUsername := string(foundCredsSecret.Data["username"])

	// Reconcile htpasswd secret based on already reconciled credentials secret
	expectedHtpasswdSecret, err := GetReporterHtpasswdSecret(instance.GetNamespace(), LabelsForMeta(instance), GetSpecAnnotations(instance), foundCredsSecret)
	if err != nil {
		return err
	}

	htpasswdLogger := AddResourceValuesToLog(logger, &expectedHtpasswdSecret)
	return ReconcileResource(
		config,
		&expectedHtpasswdSecret,
		&corev1.Secret{},
		true,
		nil,
		func(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
			foundSecret := found.(*corev1.Secret)
			expectedSecret := expected.(*corev1.Secret)

			if foundSecret.Type != expectedSecret.Type {
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}
			if !MapHasAllKeysFromOther(foundSecret.Data, expectedSecret.Data) {
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}

			data := string(foundSecret.Data["data"])
			splitData := strings.Split(data, ":")

			if len(splitData) != 2 {
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			} else {
				foundUsername := splitData[0]
				hashedPassword := []byte(splitData[1])

				if foundUsername != expectedUsername {
					return ResourceUpdateStatus{IsInDesiredState: false}, nil
				}

				if err := bcrypt.CompareHashAndPassword(hashedPassword, []byte(expectedPassword)); err != nil {
					return ResourceUpdateStatus{IsInDesiredState: false}, nil
				}
			}

			// Spec.labels support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetLabels(), GetSpecLabels(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			// Spec.annotations support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			return ResourceUpdateStatus{IsInDesiredState: true}, nil
		},
		PatchFoundWithSpecLabelsAndAnnotations,
		OverrideFoundWithExpected,
		htpasswdLogger,
		RestartReporterOperandPod,
	)
}

func generateHtpasswd(username, password string) (string, error) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s:%s\n", username, string(hashedPassword)), nil
}

func generateUserAndPassword() (string, string, error) {
	username, err := GetReporterUsername()
	if err != nil {
		return "", "", err
	}

	const passwdLength = 16
	randomBytes := make([]byte, passwdLength)
	_, err = rand.Read(randomBytes)
	if err != nil {
		return "", "", err
	}
	randStringPasswd := base64.URLEncoding.EncodeToString(randomBytes)

	return username, randStringPasswd, err
}

func GetCookieSecret(namespace string, metaLabels, metaAnnotations map[string]string) corev1.Secret {
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        ReporterAuthCookieSecret,
			Namespace:   namespace,
			Labels:      metaLabels,
			Annotations: metaAnnotations,
		},
		Type: corev1.SecretTypeOpaque,
		Data: map[string][]byte{"data": []byte(generateHash())},
	}
}

func ReconcileCookieSecret(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {

	instance := config.Instance
	expectedCookieSecret := GetCookieSecret(instance.Namespace, LabelsForMeta(instance), GetSpecAnnotations(instance))
	cookieSecretLogger := AddResourceValuesToLog(logger, &expectedCookieSecret)

	return ReconcileResource(
		config,
		&expectedCookieSecret,
		&corev1.Secret{},
		true,
		nil,
		func(config IBMLicenseServiceReporterConfig, found FoundObject, expected ExpectedObject, logger logr.Logger) (ResourceUpdateStatus, error) {
			foundSecret := found.(*corev1.Secret)
			expectedSecret := expected.(*corev1.Secret)

			if foundSecret.Type != expectedSecret.Type {
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}

			if foundSecret.Data == nil || len(foundSecret.Data) == 0 {
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}

			if val, found := foundSecret.Data["data"]; !found || len(val) == 0 {
				return ResourceUpdateStatus{IsInDesiredState: false}, nil
			}

			// Spec.labels support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetLabels(), GetSpecLabels(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			// Spec.annotations support for resource updates
			if !MapHasAllPairsFromOther(foundSecret.GetAnnotations(), GetSpecAnnotations(config.Instance)) {
				return ResourceUpdateStatus{IsPatchSufficient: true}, nil
			}

			return ResourceUpdateStatus{IsInDesiredState: true}, nil
		},
		PatchFoundWithSpecLabelsAndAnnotations,
		OverrideFoundWithExpected,
		cookieSecretLogger,
		nil,
	)
}

func generateHash() string {
	randomBytes := make([]byte, 32)
	_, err := rand.Read(randomBytes)
	if err != nil {
		panic(err)
	}

	encodedBytes := base64.StdEncoding.EncodeToString(randomBytes)
	encodedBytes = strings.TrimRight(encodedBytes, "=")
	encodedBytes = strings.ReplaceAll(encodedBytes, "+", "-")
	encodedBytes = strings.ReplaceAll(encodedBytes, "/", "_")

	return encodedBytes
}
