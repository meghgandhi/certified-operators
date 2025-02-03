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

package v1alpha1

import (
	"errors"
	"fmt"
	"os"
	"strings"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type Container struct {
	// IBM License Service Reporter docker Image Registry, will override default value and disable image env value in operator deployment
	ImageRegistry string `json:"imageRegistry,omitempty"`
	// IBM License Service Reporter docker Image Name, will override default value and disable image env value in operator deployment
	ImageName string `json:"imageName,omitempty"`
	// IBM License Service Reporter docker Image Tag or Digest, will override default value and disable image env value in operator deployment
	ImageTagPostfix string `json:"imageTagPostfix,omitempty"`

	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// +kubebuilder:validation:Enum=Always;IfNotPresent;Never
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`
}

func (container *Container) GetFullImage() string {
	// If there is ":" in image tag then we use "@" for digest as only digest can have it
	if strings.ContainsAny(container.ImageTagPostfix, ":") {
		return container.ImageRegistry + "/" + container.ImageName + "@" + container.ImageTagPostfix
	}
	return container.ImageRegistry + "/" + container.ImageName + ":" + container.ImageTagPostfix
}

// getImageParametersFromEnv get image info from full image reference
func (container *Container) GetImageParametersFromEnv(envVariableName string) error {
	fullImageName := os.Getenv(envVariableName)
	// First get imageName, to do that we need to split FullImage like path
	imagePathSplitted := strings.Split(fullImageName, "/")
	if len(imagePathSplitted) < 2 {
		text := fmt.Sprintf("ENV variable: %s should have registry and image separated with \"/\" symbol", envVariableName)
		return errors.New(text)
	}
	imageWithTag := imagePathSplitted[len(imagePathSplitted)-1]
	var imageWithTagSplitted []string
	// Check if digest and split into Image Name and TagPostfix
	if strings.Contains(imageWithTag, "@") {
		imageWithTagSplitted = strings.Split(imageWithTag, "@")
		if len(imageWithTagSplitted) != 2 {
			text := fmt.Sprintf("ENV variable: %s in operator deployment should have digest and image name separated by only one \"@\" symbol", envVariableName)
			return errors.New(text)
		}
	} else {
		imageWithTagSplitted = strings.Split(imageWithTag, ":")
		if len(imageWithTagSplitted) != 2 {
			text := fmt.Sprintf("ENV variable: %s in operator deployment should have image tag and image name separated by only one \":\" symbol", envVariableName)
			return errors.New(text)
		}
	}
	container.ImageTagPostfix = imageWithTagSplitted[1]
	container.ImageName = imageWithTagSplitted[0]
	container.ImageRegistry = strings.Join(imagePathSplitted[:len(imagePathSplitted)-1], "/")
	return nil
}

func (container *Container) InitResourcesIfNil() {
	if container.Resources.Limits == nil {
		container.Resources.Limits = corev1.ResourceList{}
	}
	if container.Resources.Requests == nil {
		container.Resources.Requests = corev1.ResourceList{}
	}
}

func (container *Container) SetResourceLimitCPUIfNotSet(value resource.Quantity) {
	if container.Resources.Limits.Cpu().IsZero() {
		container.Resources.Limits[corev1.ResourceCPU] = value
	}
}

func (container *Container) SetResourceRequestCPUIfNotSet(value resource.Quantity) {
	if container.Resources.Requests.Cpu().IsZero() {
		container.Resources.Requests[corev1.ResourceCPU] = value
	}
}

func (container *Container) SetResourceLimitMemoryIfNotSet(value resource.Quantity) {
	if container.Resources.Limits.Memory().IsZero() {
		container.Resources.Limits[corev1.ResourceMemory] = value
	}
}

func (container *Container) SetResourceRequestMemoryIfNotSet(value resource.Quantity) {
	if container.Resources.Requests.Memory().IsZero() {
		container.Resources.Requests[corev1.ResourceMemory] = value
	}
}

func (container *Container) SetResourceRequestEphemeralStorageIfNotSet(value resource.Quantity) {
	if container.Resources.Requests.StorageEphemeral().IsZero() {
		container.Resources.Requests[corev1.ResourceEphemeralStorage] = value
	}
}

func (container *Container) SetImagePullPolicyIfNotSet() {
	if container.ImagePullPolicy == "" {
		container.ImagePullPolicy = corev1.PullIfNotPresent
	}
}
