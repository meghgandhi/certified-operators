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
)

var BasicAuthVolumeMounts = []corev1.VolumeMount{
	{
		Name:      "ibm-license-reporter-auth",
		MountPath: "/opt/oauth2-proxy/htpasswd",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-cookie-secret",
		MountPath: "/opt/oauth2-proxy/config/cookie-secret",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "license-reporter-https-certs",
		MountPath: "/etc/tls/private/",
		ReadOnly:  true,
	},
}

var OAuthVolumeMountsClientSecret = []corev1.VolumeMount{
	{
		Name:      "ibm-license-reporter-auth",
		MountPath: "/opt/oauth2-proxy/htpasswd",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-cookie-secret",
		MountPath: "/opt/oauth2-proxy/config/cookie-secret",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "license-reporter-https-certs",
		MountPath: "/etc/tls/private/",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-client-secret",
		MountPath: "/opt/oauth2-proxy/config/client-secret",
		SubPath:   "data",
		ReadOnly:  true,
	},
}

var OAuthVolumeMountsCA = []corev1.VolumeMount{
	{
		Name:      "ibm-license-reporter-auth",
		MountPath: "/opt/oauth2-proxy/htpasswd",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-cookie-secret",
		MountPath: "/opt/oauth2-proxy/config/cookie-secret",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "license-reporter-https-certs",
		MountPath: "/etc/tls/private/",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-provider-ca",
		MountPath: "/opt/oauth2-proxy/config/provider-ca",
		SubPath:   "ca.crt",
		ReadOnly:  true,
	},
}

var OAuthVolumeMountsCAClientSecret = []corev1.VolumeMount{
	{
		Name:      "ibm-license-reporter-auth",
		MountPath: "/opt/oauth2-proxy/htpasswd",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-cookie-secret",
		MountPath: "/opt/oauth2-proxy/config/cookie-secret",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "license-reporter-https-certs",
		MountPath: "/etc/tls/private/",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-client-secret",
		MountPath: "/opt/oauth2-proxy/config/client-secret",
		SubPath:   "data",
		ReadOnly:  true,
	},
	{
		Name:      "ibm-license-reporter-auth-provider-ca",
		MountPath: "/opt/oauth2-proxy/config/provider-ca",
		SubPath:   "ca.crt",
		ReadOnly:  true,
	},
}
