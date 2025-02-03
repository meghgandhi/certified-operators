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

package deployments

import (
	"fmt"
	"strings"

	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/ingress"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/routes"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/services"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	ReporterAuthHtpasswdVolumeName         = "ibm-license-reporter-auth"
	ReporterAuthCookieSecretVolumeName     = "ibm-license-reporter-auth-cookie-secret"
	ReporterAuthClientSecretFileVolumeName = "ibm-license-reporter-auth-client-secret"
	ReporterAuthProviderCAVolumeName       = "ibm-license-reporter-auth-provider-ca"
	AuthContainerName                      = "auth"
	OperandReporterAuthImageEnvVar         = "IBM_LICENSE_SERVICE_REPORTER_AUTH_IMAGE"
	// Files mounted in the auth container
	OAuth2ProxyDirPath                 = "/opt/oauth2-proxy"
	HtpasswdFilePath                   = OAuth2ProxyDirPath + "/htpasswd"
	OAuth2ProxyBasicTemplatesPath      = OAuth2ProxyDirPath + "/templates/useradmin"
	OAuth2ProxyOAuthBasicTemplatesPath = OAuth2ProxyDirPath + "/templates/oauth_useradmin"
	OAuth2ProxyCookieSecretPath        = OAuth2ProxyDirPath + "/config/cookie-secret"
	OAuth2ProxyClientSecretFilePath    = OAuth2ProxyDirPath + "/config/client-secret"
	OAuth2ProxyProviderCAFilePath      = OAuth2ProxyDirPath + "/config/provider-ca"
	// External secret flags
	OAuth2ProxyClientSecretNameFlag     = "--client-secret-name"
	OAuth2ProxyProviderCASecretNameFlag = "--provider-ca-secret-name"
)

func getAuthLivenessProbeHandler() corev1.ProbeHandler {
	return corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   "/ping",
			Port:   services.ReporterAuthTargetPort,
			Scheme: "HTTPS",
		},
	}
}

func getAuthReadinessProbeHandler() corev1.ProbeHandler {
	return corev1.ProbeHandler{
		HTTPGet: &corev1.HTTPGetAction{
			Path:   "/ready",
			Port:   services.ReporterAuthTargetPort,
			Scheme: "HTTPS",
		},
	}
}

func getAuthSecretVolumeMounts(config IBMLicenseServiceReporterConfig) []corev1.VolumeMount {
	spec := config.Instance.Spec

	volumeMounts := []corev1.VolumeMount{
		{
			Name:      ReporterAuthHtpasswdVolumeName,
			MountPath: HtpasswdFilePath,
			SubPath:   "data",
			ReadOnly:  true,
		},
		{
			Name:      ReporterAuthCookieSecretVolumeName,
			MountPath: OAuth2ProxyCookieSecretPath,
			SubPath:   "data",
			ReadOnly:  true,
		},
		{
			Name:      HTTPSCertsVolumeName,
			MountPath: "/etc/tls/private/",
			ReadOnly:  true,
		},
	}

	if spec.IsOAuthEnabled() {
		oauth := spec.Authentication.OAuth
		if _, found := oauth.FindOAuthParamValue(OAuth2ProxyClientSecretNameFlag); found {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      ReporterAuthClientSecretFileVolumeName,
				MountPath: OAuth2ProxyClientSecretFilePath,
				SubPath:   "data",
				ReadOnly:  true,
			})
		}
		if _, found := oauth.FindOAuthParamValue(OAuth2ProxyProviderCASecretNameFlag); found {
			volumeMounts = append(volumeMounts, corev1.VolumeMount{
				Name:      ReporterAuthProviderCAVolumeName,
				MountPath: OAuth2ProxyProviderCAFilePath,
				SubPath:   "ca.crt",
				ReadOnly:  true,
			})
		}
	}

	return volumeMounts
}

func getAuthEnvVariables(spec v1alpha1.IBMLicenseServiceReporterSpec) []corev1.EnvVar {
	if spec.EnableInstanaMetricCollection {
		return []corev1.EnvVar{
			{
				Name: "INSTANA_AGENT_HOST",
				ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "status.hostIP",
					},
				},
			},
		}
	}
	return nil
}

func GetOAuthContainer(config IBMLicenseServiceReporterConfig) (corev1.Container, error) {
	instance := config.Instance
	spec := instance.Spec
	authContainer, err := SetContainerFromEnv(spec.AuthContainer, OperandReporterAuthImageEnvVar)
	if err != nil {
		return corev1.Container{}, err
	}
	cpu100m := resource.NewMilliQuantity(100, resource.DecimalSI)
	memory50Mi := resource.NewQuantity(50*1024*1024, resource.BinarySI)
	ephStorage, _ := resource.ParseQuantity("256Mi")
	authContainer.InitResourcesIfNil()
	authContainer.SetImagePullPolicyIfNotSet()
	authContainer.SetResourceLimitMemoryIfNotSet(*memory50Mi)
	authContainer.SetResourceRequestMemoryIfNotSet(*memory50Mi)
	authContainer.SetResourceLimitCPUIfNotSet(*cpu100m)
	authContainer.SetResourceRequestCPUIfNotSet(*cpu100m)
	authContainer.SetResourceRequestEphemeralStorageIfNotSet(ephStorage)
	container := GetContainerBase(authContainer)
	container.Env = getAuthEnvVariables(spec)

	containerArgs, err := parseOauth2ProxyArgs(config)
	if err != nil {
		return corev1.Container{}, err
	}

	container.Args = containerArgs
	container.VolumeMounts = getAuthSecretVolumeMounts(config)
	container.Name = AuthContainerName
	container.Ports = []corev1.ContainerPort{
		{
			Name:          services.ReporterAuthTargetPortName,
			ContainerPort: services.ReporterAuthTargetPort.IntVal,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	container.LivenessProbe = GetLivenessProbe(getAuthLivenessProbeHandler())
	container.ReadinessProbe = GetReadinessProbe(getAuthReadinessProbeHandler())

	return container, nil
}

func parseOauth2ProxyArgs(config IBMLicenseServiceReporterConfig) ([]string, error) {
	spec := config.Instance.Spec
	redirectUrl, err := getRedirectUrl(config, spec)
	if err != nil {
		return []string{}, err
	}

	containerArgs := []string{
		// Core setup, non-configurable
		"--https-address=:" + services.ReporterAuthServicePort.String(),
		"--proxy-prefix=" + ingress.DefaultConsolePath + "/oauth2",
		"--upstream=http://localhost:3001/",
		"--tls-cert-file=/etc/tls/private/tls.crt",
		"--tls-key-file=/etc/tls/private/tls.key",
		"--htpasswd-file=" + HtpasswdFilePath,
		"--redirect-url=" + redirectUrl,
	}

	// Basic auth only
	if spec.IsBasicAuthEnabled() && !spec.IsOAuthEnabled() {
		basicAuthArgs := []string{
			"--display-htpasswd-form=true",
			"--custom-templates-dir=" + OAuth2ProxyBasicTemplatesPath,
			"--provider-display-name=lsr-useradmin",
			"--oidc-issuer-url=https://www.lsr-useradmin.com",
			"--oidc-jwks-url=https://www.lsr-useradmin.com",
			"--client-id=lsr-useradmin",
			"--client-secret=lsr-useradmin",
			"--skip-oidc-discovery=true",
		}

		containerArgs = append(containerArgs, basicAuthArgs...)

		// Basic auth + OAuth
	} else if spec.IsBasicAuthEnabled() && spec.IsOAuthEnabled() {
		oauthBasicArgs := []string{
			"--display-htpasswd-form=true",
			"--custom-templates-dir=" + OAuth2ProxyOAuthBasicTemplatesPath,
		}

		containerArgs = append(containerArgs, oauthBasicArgs...)

		// OAuth only
	} else if !spec.IsBasicAuthEnabled() && spec.IsOAuthEnabled() {
		oauthArgs := []string{
			"--display-htpasswd-form=false",
			"--custom-templates-dir=" + OAuth2ProxyOAuthBasicTemplatesPath,
		}

		containerArgs = append(containerArgs, oauthArgs...)
	}

	authSpec := config.Instance.Spec.Authentication

	// Add all additional oauth2-proxy parameters to be passed to the binary.
	// Parameters in the CR must be passed in the following format: --param=value
	// These ARE NOT validated in any other way so user takes all the responsibility of their correctness.
	if authSpec.OAuth.Enabled && authSpec.OAuth.Parameters != nil && len(authSpec.OAuth.Parameters) > 0 {
		coreParams := getCoreParams()
		extSecretsPaths := getExtSecretsMountPaths()
		trueDefaultParams := getTrueDefaultParams()
		for _, param := range authSpec.OAuth.Parameters {
			splitParam := strings.Split(param, "=")
			if len(splitParam) != 2 {
				return []string{}, fmt.Errorf("parameter " + param + " has wrong format. It should be --param=value")
			}
			paramName := splitParam[0]
			// Filter out parameters not meant to be changed by user
			if _, found := coreParams[paramName]; found {
				continue
			}
			if path, found := extSecretsPaths[paramName]; found {
				containerArgs = append(containerArgs, path)
				continue
			}
			// If user specified custom value for a given param, don't use the default one
			delete(trueDefaultParams, paramName)

			containerArgs = append(containerArgs, param)
		}

		// Params left in the map are ones to be passed with default values
		for param, defVal := range trueDefaultParams {
			containerArgs = append(containerArgs, param+"="+defVal)
		}
	}

	return containerArgs, nil
}

func getRedirectUrl(config IBMLicenseServiceReporterConfig, spec v1alpha1.IBMLicenseServiceReporterSpec) (string, error) {
	if config.IsRouteAPI && spec.IsRouteEnabled() {
		consoleRoute, err := routes.GetExistingRoute(config.Client, routes.ReporterConsoleRouteName, config.Instance.Namespace)
		if err != nil {
			return "", fmt.Errorf("cannot parse oauth2-proxy parameters due to missing %s route: %v", routes.ReporterConsoleRouteName, err)
		}
		return "https://" + consoleRoute.Spec.Host + consoleRoute.Spec.Path + "/oauth2/callback", nil
	} else if spec.IngressEnabled {
		if spec.IngressOptions != nil && spec.IngressOptions.CommonOptions.Host != nil {
			return "https://" + *spec.IngressOptions.CommonOptions.Host + ingress.DefaultConsolePath + "/oauth2/callback", nil
		} else {
			return "", fmt.Errorf("ingress is enabled but it does not have host configured. Configure ingress host in IBMLicenseServiceReporter")
		}
	} else {
		// customer is supposed to create ingress on their own if automatic creation is disabled
		consoleIngress, err := ingress.GetExistingIngress(config.Client, ingress.ConsoleIngressName, config.Instance.Namespace)
		if err != nil {
			return "", fmt.Errorf("cannot get the customer created ingress %s - make sure it's created or enable automatic ingress creation: %v", ingress.ConsoleIngressName, err)
		}
		if consoleIngress == nil || consoleIngress.Spec.Rules == nil || len(consoleIngress.Spec.Rules) == 0 || consoleIngress.Spec.Rules[0].Host == "" {
			return "", fmt.Errorf("customer created ingress should specify exactly one rule in the spec and it needs to contain the host value")
		}
		return "https://" + consoleIngress.Spec.Rules[0].Host + ingress.DefaultConsolePath + "/oauth2/callback", nil
	}
}

func getCoreParams() map[string]string {
	return map[string]string{
		"--https-address":         "",
		"--proxy-prefix":          "",
		"--upstream":              "",
		"--tls-cert-file":         "",
		"--tls-key-file":          "",
		"--htpasswd-file":         "",
		"--display-htpasswd-form": "",
		"--custom-templates-dir":  "",
		"--redirect-url":          "",
		"--client-secret-file":    "",
		"--provider-ca-file":      "",
	}
}

func getExtSecretsMountPaths() map[string]string {
	return map[string]string{
		OAuth2ProxyClientSecretNameFlag:     "--client-secret-file=" + OAuth2ProxyClientSecretFilePath,
		OAuth2ProxyProviderCASecretNameFlag: "--provider-ca-file=" + OAuth2ProxyProviderCAFilePath,
	}
}

// Returns map of params with true default values (values only present if user does not specify customs)
func getTrueDefaultParams() map[string]string {
	return map[string]string{
		"--email-domain": "*",
	}
}
