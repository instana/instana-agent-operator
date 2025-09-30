/*
(c) Copyright IBM Corp. 2025

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

package v1

import (
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:openapi-gen=true

// InstanaAgentRemoteSpec defines the desired state of the Instana Agent Remote
type InstanaAgentRemoteSpec struct {
	// UseSecretMounts specifies whether to mount secrets as files instead of environment variables.
	// This is more secure as it prevents secrets from being exposed in the environment.
	// Default is true.
	// +kubebuilder:validation:Optional
	UseSecretMounts *bool `json:"useSecretMounts,omitempty"`

	// Agent deployment specific fields.
	// +kubebuilder:validation:Required
	Agent BaseAgentSpec `json:"agent"`

	// Name of the zone in which the host(s) will be displayed on the map. Required as we do not set cluster name.
	// +kubebuilder:validation:Required
	Zone Name `json:"zone"`

	// Specifies whether RBAC resources should be created.
	// +kubebuilder:validation:Optional
	Rbac Create `json:"rbac,omitempty"`

	// Specifies whether a ServiceAccount should be created (default `true`).
	// +kubebuilder:validation:Optional
	ServiceAccountSpec `json:"serviceAccount,omitempty"`

	// +kubebuilder:validation:Optional
	Hostname *Name `json:"hostname,omitempty"`
}

// +k8s:openapi-gen=true

// InstanaAgentRemoteOperatorState type representing the running state of the Agent Operator itself.
type InstanaAgentRemoteOperatorState string

const (
	RemoteOperatorStateRunning  InstanaAgentRemoteOperatorState = "Running"
	RemoteOperatorStateUpdating InstanaAgentRemoteOperatorState = "Updating"
	RemoteOperatorStateFailed   InstanaAgentRemoteOperatorState = "Failed"
)

// +k8s:openapi-gen=true

type InstanaAgentRemoteStatus struct {
	ConfigSecret       ResourceInfo       `json:"configsecret,omitempty"`
	Conditions         []metav1.Condition `json:"conditions,omitempty"`
	ObservedGeneration *int64             `json:"observedGeneration,omitempty"`
	OperatorVersion    *SemanticVersion   `json:"operatorVersion,omitempty"`
	Deployment         ResourceInfo       `json:"deployment,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=agentsremote,singular=agentremote,shortName=ar,scope=Namespaced,categories=monitoring;openshift-optional
// +kubebuilder:storageversion
// +operator-sdk:csv:customresourcedefinitions:displayName="Remote Instana Agent",resources={{Deployment,apps/v1,InstanaAgentRemote},{Pod,v1,InstanaAgentRemote},{Secret,v1,InstanaAgentRemote}}

type InstanaAgentRemote struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanaAgentRemoteSpec   `json:"spec,omitempty"`
	Status InstanaAgentRemoteStatus `json:"status,omitempty"`
}

func (in *InstanaAgentRemote) Default() {
	optional.ValueOrDefault(&in.Spec.Agent.EndpointHost, "ingress-red-saas.instana.io")
	optional.ValueOrDefault(&in.Spec.Agent.EndpointPort, "443")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.Name, "icr.io/instana/agent")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.Tag, "latest")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.PullPolicy, corev1.PullAlways)
	optional.ValueOrDefault(&in.Spec.Rbac.Create, pointer.To(true))
	optional.ValueOrDefault(&in.Spec.ServiceAccountSpec.Create.Create, pointer.To(true))

	// Set default value for useSecretMounts to true for all instances
	// This is more secure as it prevents secrets from being exposed in the environment
	optional.ValueOrDefault(&in.Spec.UseSecretMounts, pointer.To(true))
}

// +kubebuilder:object:root=true

type InstanaAgentRemoteList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanaAgentRemote `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanaAgentRemote{}, &InstanaAgentRemoteList{})
}
