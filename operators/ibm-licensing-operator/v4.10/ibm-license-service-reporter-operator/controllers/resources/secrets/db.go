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
	"github.com/go-logr/logr"
	api "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const DatabaseConfigSecretName = "license-service-reporter-hub-db-config"
const PostgresPasswordKey = "POSTGRES_PASSWORD" // #nosec
const PostgresUserKey = "POSTGRES_USER"
const PostgresDatabaseNameKey = "POSTGRES_DB"
const PostgresPgDataKey = "POSTGRES_PGDATA"

const DatabaseName = "postgres"
const DatabaseMountPath = "/var/lib/postgresql"
const PgData = DatabaseMountPath + "/pgdata"

func ReconcileDatabaseSecret(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	instance := config.Instance
	expectedSecret, err := GetDatabaseSecret(instance)
	if err != nil {
		logger.Error(err, "Failed to get expected secret")
		return err
	}
	logger = AddResourceValuesToLog(logger, expectedSecret)
	return ReconcileResource(
		config,
		expectedSecret,
		&corev1.Secret{},
		true,
		nil,
		IsDatabaseSecretInDesiredState,
		PatchFoundWithSpecLabelsAndAnnotations,
		OverrideFoundWithExpected,
		logger,
		nil,
	)
}

func GetDatabaseSecret(instance api.IBMLicenseServiceReporter) (*corev1.Secret, error) {
	metaLabels := LabelsForMeta(instance)
	randString, err := RandString(8)
	if err != nil {
		return nil, err
	}
	dbUser, err := GetDatabaseUsername()
	if err != nil {
		return nil, err
	}
	expectedSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        DatabaseConfigSecretName,
			Namespace:   instance.GetNamespace(),
			Labels:      metaLabels,
			Annotations: GetSpecAnnotations(instance),
		},
		Type: corev1.SecretTypeOpaque,
		StringData: map[string]string{
			PostgresPasswordKey:     randString,
			PostgresUserKey:         dbUser,
			PostgresDatabaseNameKey: DatabaseName,
			PostgresPgDataKey:       PgData,
		},
	}
	return expectedSecret, nil
}

func IsDatabaseSecretInDesiredState(
	config IBMLicenseServiceReporterConfig,
	found FoundObject,
	expected ExpectedObject,
	logger logr.Logger,
) (ResourceUpdateStatus, error) {
	foundSecret := found.(*corev1.Secret)
	expectedSecret := expected.(*corev1.Secret)

	// Check Secret type
	if foundSecret.Type != expectedSecret.Type {
		logger.Info("Updating " + foundSecret.GetName() + " due to type mismatch")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// Check keys presence
	if !MapHasAllKeysFromOther(foundSecret.Data, expectedSecret.StringData) {
		logger.Info("Updating " + foundSecret.GetName() + " due to data mismatch")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// In case patch sufficient in spec.labels check but not in spec.annotations check (can't return early)
	shouldReturnPatchSufficientAfterAnnotationsCheck := false

	// Check labels, call patch if only spec.labels mismatch
	if !MapHasAllPairsFromOther(foundSecret.GetLabels(), expectedSecret.GetLabels()) {
		if MapHasAllPairsFromOther(MergeMaps(foundSecret.GetLabels(), GetSpecLabels(config.Instance)), expectedSecret.GetLabels()) {
			shouldReturnPatchSufficientAfterAnnotationsCheck = true
		}
		logger.Info("Updating " + foundSecret.GetName() + " due to having outdated labels")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	// Check annotations, call patch if only spec.annotations mismatch
	if !MapHasAllPairsFromOther(foundSecret.GetAnnotations(), expectedSecret.GetAnnotations()) {
		if MapHasAllPairsFromOther(MergeMaps(foundSecret.GetAnnotations(), GetSpecAnnotations(config.Instance)), expectedSecret.GetAnnotations()) {
			return ResourceUpdateStatus{IsPatchSufficient: true}, nil
		}
		logger.Info("Updating " + foundSecret.GetName() + " due to having outdated annotations")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	if shouldReturnPatchSufficientAfterAnnotationsCheck {
		return ResourceUpdateStatus{IsPatchSufficient: true}, nil
	}

	return ResourceUpdateStatus{IsInDesiredState: true}, nil
}
