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

func GetBasicAuthOnlyParams(prefixPath, fullPath string, additionalParams []string) []string {
	proxyPrefix := "--proxy-prefix=/license-service-reporter/oauth2"
	redirectUri := "--redirect-url=https://ibm-lsr-console-ibm-licensing.apps.jp2.cp.fyre.ibm.com/license-service-reporter/oauth2/callback"

	if prefixPath != "" {
		proxyPrefix = "--proxy-prefix=" + prefixPath + "/oauth2"
	}
	if fullPath != "" {
		redirectUri = "--redirect-url=" + fullPath + "/oauth2/callback"
	}

	args := []string{
		"--https-address=:8888",
		proxyPrefix,
		"--upstream=http://localhost:3001/",
		"--tls-cert-file=/etc/tls/private/tls.crt",
		"--tls-key-file=/etc/tls/private/tls.key",
		"--htpasswd-file=/opt/oauth2-proxy/htpasswd",
		redirectUri,
		"--display-htpasswd-form=true",
		"--custom-templates-dir=/opt/oauth2-proxy/templates/useradmin",
		"--provider-display-name=lsr-useradmin",
		"--oidc-issuer-url=https://www.lsr-useradmin.com",
		"--oidc-jwks-url=https://www.lsr-useradmin.com",
		"--client-id=lsr-useradmin",
		"--client-secret=lsr-useradmin",
		"--skip-oidc-discovery=true",
	}

	return append(args, additionalParams...)
}

func GetBasicAuthOAuthParams(additionalParams []string) []string {
	args := []string{
		"--https-address=:8888",
		"--proxy-prefix=/license-service-reporter/oauth2",
		"--upstream=http://localhost:3001/",
		"--tls-cert-file=/etc/tls/private/tls.crt",
		"--tls-key-file=/etc/tls/private/tls.key",
		"--htpasswd-file=/opt/oauth2-proxy/htpasswd",
		"--redirect-url=https://ibm-lsr-console-ibm-licensing.apps.jp2.cp.fyre.ibm.com/license-service-reporter/oauth2/callback",
		"--display-htpasswd-form=true",
		"--custom-templates-dir=/opt/oauth2-proxy/templates/oauth_useradmin",
		"--email-domain=*", // will appear only when any custom param is passed -> target scenario
	}
	return append(args, additionalParams...)
}

func GetOAuthOnlyParams(prefixPath, fullPath string, additionalParams []string) []string {
	proxyPrefix := "--proxy-prefix=/license-service-reporter/oauth2"
	redirectUri := "--redirect-url=https://ibm-lsr-console-ibm-licensing.apps.jp2.cp.fyre.ibm.com/license-service-reporter/oauth2/callback"

	if prefixPath != "" {
		proxyPrefix = "--proxy-prefix=" + prefixPath + "/oauth2"
	}
	if fullPath != "" {
		redirectUri = "--redirect-url=" + fullPath + "/oauth2/callback"
	}

	args := []string{
		"--https-address=:8888",
		proxyPrefix,
		"--upstream=http://localhost:3001/",
		"--tls-cert-file=/etc/tls/private/tls.crt",
		"--tls-key-file=/etc/tls/private/tls.key",
		"--htpasswd-file=/opt/oauth2-proxy/htpasswd",
		redirectUri,
		"--display-htpasswd-form=false",
		"--custom-templates-dir=/opt/oauth2-proxy/templates/oauth_useradmin",
		"--email-domain=*", // will appear only when any custom param is passed -> target scenario
	}
	return append(args, additionalParams...)
}
