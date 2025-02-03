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
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"math/big"
	"time"

	"github.com/go-logr/logr"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

// OcpCertsSource means application will use cert manager
const OcpCertsSource string = "ocp"

// ExternalCertsSource means operand will use certificate from a volume mounted to a container
const ExternalCertsSource = "external"

// SelfSignedCertsSource means application will create certificate by itself and use it
const SelfSignedCertsSource = "self-signed"

// CustomCertsSource means application will use certificate created by user
const CustomCertsSource = "custom"

const ExternalCertName = "ibm-license-service-reporter-cert"

func IsOCPCertSource(spec v1alpha1.IBMLicenseServiceReporterSpec) bool {
	return spec.HTTPSCertsSource == OcpCertsSource || spec.HTTPSCertsSource == ""
}

func GenerateSelfSignedCertSecret(instance v1alpha1.IBMLicenseServiceReporter, namespacedName types.NamespacedName, dns []string) (*corev1.Secret, error) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, err
	}

	// Generate a pem block with the private key
	keyPem := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: x509.MarshalPKCS1PrivateKey(key),
	})

	commonName := ""
	if len(dns) > 0 {
		commonName = dns[0]
	}

	// need to generate a different serial number each execution
	serialNumber, _ := rand.Int(rand.Reader, big.NewInt(1000000))

	tml := x509.Certificate{
		NotBefore:    time.Now(),
		NotAfter:     time.Now().AddDate(1, 0, 0),
		SerialNumber: serialNumber,
		Subject: pkix.Name{
			CommonName:   commonName,
			Organization: []string{"IBM"},
		},
		BasicConstraintsValid: true,
	}

	if dns != nil {
		tml.DNSNames = dns
	}

	cert, err := x509.CreateCertificate(rand.Reader, &tml, &tml, &key.PublicKey, key)
	if err != nil {
		return nil, err
	}

	certPem := pem.EncodeToMemory(&pem.Block{
		Type:  "CERTIFICATE",
		Bytes: cert,
	})

	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:        namespacedName.Name,
			Namespace:   namespacedName.Namespace,
			Labels:      LabelsForMeta(instance),
			Annotations: GetSpecAnnotations(instance),
		},
		Data: map[string][]byte{
			"tls.crt": certPem,
			"tls.key": keyPem,
		},
		Type: corev1.SecretTypeTLS,
	}, nil
}

func parseCertificate(rawCertData []byte, logger logr.Logger) (*x509.Certificate, error) {
	block, rest := pem.Decode(rawCertData)

	if len(rest) > 0 {
		logger.Info("Extraneous data read from the TLS certificate - ignoring it and attempting decoding of the valid PEM certificate block.", "rest", rest)
	}

	if block != nil {
		return x509.ParseCertificate(block.Bytes)
	}

	return nil, errors.New("unable to decode PEM certificate block")
}

func getSelfSignedCertWithOwnerReference(
	logger logr.Logger,
	config IBMLicenseServiceReporterConfig,
	namespacedName types.NamespacedName,
	dns []string,
) (*corev1.Secret, error) {

	secret, err := GenerateSelfSignedCertSecret(config.Instance, namespacedName, dns)
	if err != nil {
		logger.Error(err, "Error when generating self signed certificate")
	}
	err = controllerutil.SetControllerReference(&config.Instance, secret, config.Scheme)
	if err != nil {
		logger.Error(err, "Failed to set owner reference in secret")
		return nil, err
	}
	return secret, nil
}

func IsSecretCertificateInDesiredState(
	config IBMLicenseServiceReporterConfig,
	logger logr.Logger,
	foundSecret corev1.Secret,
	expectedSecret corev1.Secret,
	hosts []string,
) (ResourceUpdateStatus, error) {
	cert, err := parseCertificate(foundSecret.Data["tls.crt"], logger)
	if err != nil {
		logger.Error(err, "improper x509 certificate in secret")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	if cert.NotAfter.Before(time.Now().AddDate(0, 0, 90)) {
		logger.Error(err, "self-signed certificate has expired")
		return ResourceUpdateStatus{IsInDesiredState: false}, nil
	}

	for _, hostName := range hosts {
		if err = cert.VerifyHostname(hostName); err != nil {
			logger.Error(err, "certificate not issued to the proper hostname")
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
}
