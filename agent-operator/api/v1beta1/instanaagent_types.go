/*
Copyright 2021 Instana.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1beta1

import (
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ImageSpec describes an image name and tag to use
type ImageSpec struct {
	//Docker image name to use
	Name string `json:"name"`

	//Docker image tag to use
	Tag string `json:"tag"`
}

//ConfigSpec describes the config template
type ConfigSpec struct {
	//Number of replicas, will default to 1
	Replicas *int32 `json:"replicas,omitempty"`

	//Docker image to use
	Image ImageSpec `json:"image"`

	//Port, will default to 8000
	Port *int32 `json:"port,omitempty"`

	//Memory to allocate
	Memory resource.Quantity `json:"memory"`

	// Node group to have affinity toward
	NodeAffinityLabel string `json:"nodeAffinityLabel,omitempty"`
}

// InstanaAgentSpec defines the desired state of the Instana Agent
type InstanaAgentSpec struct {
	//Config spec
	Config ConfigSpec `json:"config"`
}

// InstanaAgentStatus defines the observed state of InstanaAgent
type InstanaAgentStatus struct {
	//Status of each config
	ConfigStatusses []appsv1.DeploymentStatus `json:"configStatusses,omitempty"`
}

// +kubebuilder:object:root=true

// InstanaAgent is the Schema for the beeinstanas API
type InstanaAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanaAgentSpec   `json:"spec,omitempty"`
	Status InstanaAgentStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// InstanaAgentList contains a list of InstanaAgent
type InstanaAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanaAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanaAgent{}, &InstanaAgentList{})
}
