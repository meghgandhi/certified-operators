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
	api "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
	. "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources"
	"github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/controllers/resources/secrets"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

const OcpCheckString = "ocp-check-secret"

func GetContainerBase(container api.Container) corev1.Container {
	return corev1.Container{
		Image:           container.GetFullImage(),
		ImagePullPolicy: container.ImagePullPolicy,
		SecurityContext: GetSecurityContext(),
		Resources:       container.Resources,
	}
}

func GetSecurityContext() *corev1.SecurityContext {
	var trueVar = true
	var falseVar = false
	securityContext := &corev1.SecurityContext{
		AllowPrivilegeEscalation: &falseVar,
		Privileged:               &falseVar,
		ReadOnlyRootFilesystem:   &trueVar,
		RunAsNonRoot:             &trueVar,
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{
				"ALL",
			},
		},
		SeccompProfile: &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		},
		ProcMount: ptr.To(corev1.DefaultProcMount),
	}
	return securityContext
}

func GetReadinessProbe(probeHandler corev1.ProbeHandler) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 60,
		TimeoutSeconds:      10,
		PeriodSeconds:       60,
	}
}

func GetLivenessProbe(probeHandler corev1.ProbeHandler) *corev1.Probe {
	return &corev1.Probe{
		ProbeHandler:        probeHandler,
		InitialDelaySeconds: 120,
		TimeoutSeconds:      10,
		PeriodSeconds:       300,
	}
}

func GetOCPSecretCheckScript() string {
	script := `while true; do
  echo "$(date): Checking for ocp secret"
  ls /opt/licensing/certs/* && break
  echo "$(date): Required ocp secret not found ... try again in 30s"
  sleep 30
done
echo "$(date): All required secrets exist"
`
	return script
}

func getLicenseReporterInitContainers(config IBMLicenseServiceReporterConfig) ([]corev1.Container, error) {
	containers := make([]corev1.Container, 0)
	if config.IsServiceCAAPI && secrets.IsOCPCertSource(config.Instance.Spec) {
		ocpSecretCheckContainer, err := GetReceiverContainer(config)
		if err != nil {
			return nil, err
		}
		ocpSecretCheckContainer.LivenessProbe = nil
		ocpSecretCheckContainer.ReadinessProbe = nil
		ocpSecretCheckContainer.Name = OcpCheckString
		ocpSecretCheckContainer.Command = []string{
			"sh",
			"-c",
			GetOCPSecretCheckScript(),
		}
		containers = append(containers, ocpSecretCheckContainer)
	}
	return containers, nil
}

func SetContainerFromEnv(container api.Container, envVar string) (api.Container, error) {
	var temp api.Container
	if err := temp.GetImageParametersFromEnv(envVar); err != nil {
		return temp, err
	}
	// If CR has at least one override, make sure all parts of the image are filled at least with default values c ENV
	if container.ImageName == "" {
		container.ImageName = temp.ImageName
	}
	if container.ImageRegistry == "" {
		container.ImageRegistry = temp.ImageRegistry
	}
	if container.ImageTagPostfix == "" {
		container.ImageTagPostfix = temp.ImageTagPostfix
	}
	return container, nil
}
