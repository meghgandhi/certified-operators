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

package mocks

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	CookieSecretNameMock = "ibm-license-service-reporter-auth-cookie"
)

var CookieSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      CookieSecretNameMock,
		Namespace: "test",
		Labels:    GetLabelsForMeta(CookieSecretNameMock),
	},
	Type: corev1.SecretTypeOpaque,
	Data: map[string][]byte{"data": []byte("Wkrh_97IIbroIMxddk5mONsw8wGGDNC-Po_a0olD82k")},
}

var ExternalCertSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ibm-license-service-reporter-cert",
		Namespace: "test",
	},
	Type: corev1.SecretTypeOpaque,
	Data: map[string][]byte{"tls.crt": []byte("-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----")},
}

var InternalCertSecret = corev1.Secret{
	ObjectMeta: metav1.ObjectMeta{
		Name:      "ibm-license-service-reporter-cert-internal",
		Namespace: "test",
	},
	Type: corev1.SecretTypeOpaque,
	Data: map[string][]byte{"tls.crt": []byte("-----BEGIN CERTIFICATE-----test-----END CERTIFICATE-----")},
}
