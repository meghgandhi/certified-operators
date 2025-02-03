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
	routev1 "github.com/openshift/api/route/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var ConsoleRoute = routev1.Route{
	ObjectMeta: metav1.ObjectMeta{
		Name:        "ibm-lsr-console",
		Namespace:   "test",
		Labels:      GetLabelsForMeta("instance"),
		Annotations: map[string]string{"haproxy.router.openshift.io/timeout": "90s"},
	},
	Spec: routev1.RouteSpec{
		Host: "ibm-lsr-console-ibm-licensing.apps.jp2.cp.fyre.ibm.com",
		Path: "/license-service-reporter",
		To: routev1.RouteTargetReference{
			Kind: "Service",
			Name: "ibm-license-service-reporter",
		},
		Port: &routev1.RoutePort{
			TargetPort: intstr.FromString("8888"),
		},
	},
}
