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

package routes

import (
	"context"
	"errors"
	"fmt"
	"regexp"

	"github.com/go-logr/logr"
	routev1 "github.com/openshift/api/route/v1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func ReconcileRouteWithCertificates(logger logr.Logger, config IBMLicenseServiceReporterConfig) error {
	if config.IsRouteAPI && config.Instance.Spec.IsRouteEnabled() {
		instance := config.Instance
		logger.Info("Reconciling route with certificate")
		externalCertSecret := corev1.Secret{}
		var externalCertName string
		if instance.Spec.HTTPSCertsSource == secrets.CustomCertsSource {
			externalCertName = services.CustomExternalCertSecretName
		} else {
			externalCertName = secrets.ExternalCertName
		}

		externalNamespacedName := types.NamespacedName{Namespace: instance.GetNamespace(), Name: externalCertName}
		if err := config.Client.Get(context.TODO(), externalNamespacedName, &externalCertSecret); err != nil {
			return fmt.Errorf("cannot retrieve external certificate from secret: %w", err)
		}

		internalCertSecret := corev1.Secret{}
		internalNamespacedName := types.NamespacedName{Namespace: instance.GetNamespace(), Name: services.InternalCertSecretName}
		if err := config.Client.Get(context.TODO(), internalNamespacedName, &internalCertSecret); err != nil {
			return fmt.Errorf("cannot retrieve internal certificate from secret: %w", err)
		}

		cert, caCert, key, err := ProcessCertificateSecret(externalCertSecret)
		if err != nil {
			return fmt.Errorf("invalid external certificate format in secret: %w", err)
		}
		_, destinationCaCert, _, err := ProcessCertificateSecret(internalCertSecret)
		if err != nil {
			return fmt.Errorf("invalid internal certificate format in secret: %w", err)
		}

		tlsConfig := routev1.TLSConfig{
			Termination:                   routev1.TLSTerminationReencrypt,
			InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyNone,
			Certificate:                   cert,
			CACertificate:                 caCert,
			Key:                           key,
			DestinationCACertificate:      destinationCaCert,
		}
		expectedRoute := GetReporterRoute(instance, tlsConfig)
		return ReconcileRouteWithTLS(logger, config, tlsConfig, expectedRoute)
	}
	return nil
}

func ProcessCertificateSecret(secret corev1.Secret) (cert, caCert, key string, err error) {
	certChain := string(secret.Data["tls.crt"])
	key = string(secret.Data["tls.key"])
	re := regexp.MustCompile("(?s)-----BEGIN CERTIFICATE-----.*?-----END CERTIFICATE-----")
	externalCerts := re.FindAllString(certChain, -1)

	if len(externalCerts) == 0 {
		err = errors.New("invalid certificate format under tls.crt section")
		return
	}

	cert = externalCerts[0]

	if len(externalCerts) == 2 {
		caCert = externalCerts[1]
	} else {
		caCert = ""
	}
	return
}
