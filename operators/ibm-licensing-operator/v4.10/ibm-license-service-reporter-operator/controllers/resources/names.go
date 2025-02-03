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

package resources

import (
	"os"

	apiv1alpha1 "github.ibm.com/cloud-license-reporter/ibm-license-service-reporter-operator/api/v1alpha1"
)

const LicenseReporterReleaseName = "ibm-license-service-reporter"
const LicenseReporterResourceBase = LicenseReporterReleaseName
const LicenseReporterComponentName = LicenseReporterReleaseName + "-svc"
const LicenseReporterOperatorName = "ibm-license-service-reporter-operator"

const CachingLabelKey = "app.kubernetes.io/instance"
const CachingLabelValue = LicenseReporterReleaseName

// Important product values needed for annotations
const LicensingProductName = "IBM Cloud Platform Common Services"
const LicensingProductID = "068a62892a1e4db39641342e592daa25"
const LicensingProductMetric = "FREE"

func GetDefaultResourceName(instanceName string) string {
	return LicenseReporterResourceBase + "-" + instanceName
}

func LabelsForSelector(instanceName string) map[string]string {
	return map[string]string{"app": GetDefaultResourceName(instanceName), "component": LicenseReporterComponentName, "licensing_cr": instanceName}
}

func LabelsForMeta(instance apiv1alpha1.IBMLicenseServiceReporter) map[string]string {
	metaLabels := map[string]string{
		"app.kubernetes.io/name":       GetDefaultResourceName(instance.GetName()),
		"app.kubernetes.io/component":  LicenseReporterComponentName,
		"app.kubernetes.io/managed-by": "operator",
		CachingLabelKey:                CachingLabelValue,
		"release":                      LicenseReporterReleaseName,
	}

	return MergeMaps(metaLabels, GetSpecLabels(instance))
}

func LabelsForPod(instance apiv1alpha1.IBMLicenseServiceReporter) map[string]string {
	podLabels := LabelsForMeta(instance)

	selectorLabels := LabelsForSelector(instance.GetName())
	for key, value := range selectorLabels {
		podLabels[key] = value
	}
	return podLabels
}

func AnnotationsForPod(instance apiv1alpha1.IBMLicenseServiceReporter) map[string]string {
	annotations := map[string]string{
		"productName":   LicensingProductName,
		"productID":     LicensingProductID,
		"productMetric": LicensingProductMetric,
	}

	return MergeMaps(annotations, GetSpecAnnotations(instance))
}

func GetDatabaseUsername() (string, error) {
	content, err := os.ReadFile("/tmp/POSTGRES_USER")
	if err != nil {
		return "", err
	}
	return string(content)[:len(string(content))-1], nil
}

func GetReporterUsername() (string, error) {
	content, err := os.ReadFile("/tmp/REPORTER_USER")
	if err != nil {
		return "", err
	}
	return string(content)[:len(string(content))-1], nil
}
