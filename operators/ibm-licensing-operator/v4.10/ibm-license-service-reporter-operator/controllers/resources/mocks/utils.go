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

const (
	ReporterBaseName = "ibm-license-service-reporter"
)

// Copy from controllers/resources/names.go
func GetLabelsForMeta(instanceName string) map[string]string {
	return map[string]string{"app.kubernetes.io/name": ReporterBaseName + "-" + instanceName, "app.kubernetes.io/component": ReporterBaseName + "-svc",
		"app.kubernetes.io/managed-by": "operator", "app.kubernetes.io/instance": ReporterBaseName, "release": ReporterBaseName}

}
