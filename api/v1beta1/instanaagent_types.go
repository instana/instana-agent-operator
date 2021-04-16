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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// // ImageSpec describes an image name and tag to use
// type ImageSpec struct {
// 	//Docker image name to use
// 	Name string `json:"name"`

// 	//Docker image tag to use
// 	Tag string `json:"tag"`
// }

// //ConfigSpec describes the config template
// type ConfigSpec struct {
// 	//Number of replicas, will default to 1
// 	Replicas *int32 `json:"replicas,omitempty"`

// 	//Docker image to use
// 	Image ImageSpec `json:"image"`

// 	//Port, will default to 8000
// 	Port *int32 `json:"port,omitempty"`

// 	//Memory to allocate
// 	Memory resource.Quantity `json:"memory"`

// 	// Node group to have affinity toward
// 	NodeAffinityLabel string `json:"nodeAffinityLabel,omitempty"`
// }

// InstanaAgentSpec defines the desired state of the Instana Agent
// type InstanaAgentSpec struct {
// 	//Config spec
// 	Config ConfigSpec `json:"config"`

// }

type InstanaAgentEndpoint struct {
	Host string `json:"host,omitempty"`
	Port string `json:"port,omitempty"`
}

// InstanaAgentSpec defines the desired state of the Instana Agent
// +k8s:openapi-gen=true
type InstanaAgentSpec struct {
	// Optional: Set the zone of the host
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ZoneName string `json:"zone.name,omitempty"`

	// your Instana agent key
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent Key"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Key string `json:"key,omitempty"`

	// set agent's endpoint
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent Endpoint"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Endpoint *InstanaAgentEndpoint `json:"endpoint,omitempty"`

	// Set environment vars
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent environment variable"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env string `json:"env,omitempty"`
}

// InstanaAgentStatus defines the observed state of InstanaAgent
type InstanaAgentStatus struct {
	//Status of each config
	ConfigStatusses []appsv1.DeploymentStatus `json:"configStatusses,omitempty"`
}

// +kubebuilder:object:root=true

// Defines the desired specs and states for instana agent deployment.
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Daemonset,v1,"instana-agent-operator"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Pod,v1,"instana-agent-operator"
// +operator-sdk:gen-csv:customresourcedefinitions.resources=`Service,v1,"instana-agent-operator"
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
