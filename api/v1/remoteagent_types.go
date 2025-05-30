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
	"reflect"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:openapi-gen=true

// RemoteAgentSpec defines the desired state of the Remote Agent
type RemoteAgentSpec struct {
	// Agent deployment specific fields.
	// +kubebuilder:validation:Optional
	Agent BaseAgentSpec `json:"agent"`

	//Hostname Name `json:"hostname,omitempty"`

	Cluster Name `json:"cluster,omitempty"` //inherit from main agent

	// Name of the zone in which the host(s) will be displayed on the map. Optional, but then 'cluster.name' must be specified.
	Zone Name `json:"zone,omitempty"`

	// Specifies whether RBAC resources should be created.
	// +kubebuilder:validation:Optional
	Rbac Create `json:"rbac,omitempty"`

	// Specifies whether a ServiceAccount should be created (default `true`), and possibly the name to use.
	// +kubebuilder:validation:Optional
	ServiceAccountSpec `json:"serviceAccount,omitempty"`

	// +kubebuilder:validation:Optional
	ResourceRequirements `json:",omitempty"`

	// Supply Agent configuration e.g. for configuring certain Sensors.
	// +kubebuilder:validation:Optional
	ConfigurationYaml string `json:"remote_configuration_yaml,omitempty"`
}

// +k8s:openapi-gen=true

// RemoteAgentOperatorState type representing the running state of the Agent Operator itself.
type RemoteAgentOperatorState string

const (
	// RemoteOperatorStateRunning the operator is running properly and all changes applied successfully.
	RemoteOperatorStateRunning RemoteAgentOperatorState = "Running"
	// RemoteOperatorStateUpdating the operator is running properly but is currently applying CR changes and getting the Agent in the correct state.
	RemoteOperatorStateUpdating RemoteAgentOperatorState = "Updating"
	// RemoteOperatorStateFailed the operator is not running properly and likely there were issues applying the CustomResource correctly.
	RemoteOperatorStateFailed RemoteAgentOperatorState = "Failed"
)

// +k8s:openapi-gen=true

// RemoteAgentStatus defines the observed state of RemoteAgent

// Deprecated: DeprecatedRemoteAgentStatus are the previous status fields that will be used to ensure backwards compatibility with any automation that may exist
type DeprecatedRemoteAgentStatus struct {
	Status     AgentOperatorState `json:"status,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	LastUpdate metav1.Time        `json:"lastUpdate,omitempty"`

	OldVersionsUpdated bool `json:"oldVersionsUpdated,omitempty"`

	Deployment ResourceInfo `json:"deployment,omitempty"`
}

type RemoteAgentStatus struct {
	ConfigSecret                ResourceInfo `json:"configsecret,omitempty"`
	DeprecatedRemoteAgentStatus `json:",inline"`
	// +patchMergeKey=type
	// +patchStrategy=merge
	// +listType=map
	// +listMapKey=type
	Conditions []metav1.Condition `json:"conditions,omitempty"`
	// +kubebuilder:validation:Minimum=0
	ObservedGeneration *int64           `json:"observedGeneration,omitempty"`
	OperatorVersion    *SemanticVersion `json:"operatorVersion,omitempty"`
}

// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=remoteagents,singular=remoteagent,shortName=ra,scope=Namespaced,categories=monitoring;
// +kubebuilder:storageversion
// +operator-sdk:csv:customresourcedefinitions:displayName="Remote Instana Agent",resources={{Deployment,apps/v1,RemoteAgent},{Pod,v1,RemoteAgent},{Secret,v1,RemoteAgent}}

// RemoteAgent is the Schema for the agents API
type RemoteAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteAgentSpec   `json:"spec,omitempty"`
	Status RemoteAgentStatus `json:"status,omitempty"`
}

func (in *RemoteAgent) DefaultWithHost(agent InstanaAgent) {
	// Get desired values from the host agent spec with defaults.
	desiredEndpointHost := optional.Of(agent.Spec.Agent.EndpointHost).GetOrDefault("ingress-red-saas.instana.io")
	inherit(&in.Spec.Agent.EndpointHost, &desiredEndpointHost)

	desiredEndpointPort := optional.Of(agent.Spec.Agent.EndpointPort).GetOrDefault("443")
	inherit(&in.Spec.Agent.EndpointPort, &desiredEndpointPort)

	desiredImageName := optional.Of(agent.Spec.Agent.ImageSpec.Name).GetOrDefault("icr.io/instana/agent")
	inherit(&in.Spec.Agent.ImageSpec.Name, &desiredImageName)

	desiredImageTag := optional.Of(agent.Spec.Agent.ImageSpec.Tag).GetOrDefault("latest")
	inherit(&in.Spec.Agent.ImageSpec.Tag, &desiredImageTag)

	desiredPullPolicy := optional.Of(agent.Spec.Agent.ImageSpec.PullPolicy).GetOrDefault(corev1.PullAlways)
	inherit(&in.Spec.Agent.ImageSpec.PullPolicy, &desiredPullPolicy)

	desiredRbac := optional.Of(agent.Spec.Rbac.Create).GetOrDefault(pointer.To(true))
	inherit(&in.Spec.Rbac.Create, &desiredRbac)

	desiredSA := optional.Of(agent.Spec.ServiceAccountSpec.Create.Create).GetOrDefault(pointer.To(true))
	inherit(&in.Spec.ServiceAccountSpec.Create.Create, &desiredSA)

	//Get desired values from the host agent spec
	inherit(&in.Spec.Agent.ConfigurationYaml, &in.Spec.ConfigurationYaml)
	inherit(&in.Spec.Cluster.Name, &agent.Spec.Cluster.Name)
	inherit(&in.Spec.Agent.Key, &agent.Spec.Agent.Key)
	inherit(&in.Spec.Agent.DownloadKey, &agent.Spec.Agent.DownloadKey)
	inherit(&in.Spec.Agent.KeysSecret, &agent.Spec.Agent.KeysSecret)
	inherit(&in.Spec.Agent.ListenAddress, &agent.Spec.Agent.ListenAddress)
	inherit(&in.Spec.Agent.MinReadySeconds, &agent.Spec.Agent.MinReadySeconds)
	inherit(&in.Spec.Agent.ProxyHost, &agent.Spec.Agent.ProxyHost)
	inherit(&in.Spec.Agent.ProxyPassword, &agent.Spec.Agent.ProxyPassword)
	inherit(&in.Spec.Agent.ProxyPort, &agent.Spec.Agent.ProxyPort)
	inherit(&in.Spec.Agent.ProxyProtocol, &agent.Spec.Agent.ProxyProtocol)
	inherit(&in.Spec.Agent.ProxyUseDNS, &agent.Spec.Agent.ProxyUseDNS)
	inherit(&in.Spec.Agent.ProxyUser, &agent.Spec.Agent.ProxyUser)
	inherit(&in.Spec.Agent.RedactKubernetesSecrets, &agent.Spec.Agent.RedactKubernetesSecrets)
	inherit(&in.Spec.Agent.MvnRepoFeaturesPath, &agent.Spec.Agent.MvnRepoFeaturesPath)
	inherit(&in.Spec.Agent.MvnRepoSharedPath, &agent.Spec.Agent.MvnRepoSharedPath)
	inherit(&in.Spec.Agent.MvnRepoUrl, &agent.Spec.Agent.MvnRepoUrl)
	inherit(&in.Spec.Agent.MirrorReleaseRepoPassword, &agent.Spec.Agent.MirrorReleaseRepoPassword)
	inherit(&in.Spec.Agent.MirrorReleaseRepoUrl, &agent.Spec.Agent.MirrorReleaseRepoUrl)
	inherit(&in.Spec.Agent.MirrorReleaseRepoUsername, &agent.Spec.Agent.MirrorReleaseRepoUsername)
	inherit(&in.Spec.Agent.MirrorSharedRepoPassword, &agent.Spec.Agent.MirrorSharedRepoPassword)
	inherit(&in.Spec.Agent.MirrorSharedRepoUrl, &agent.Spec.Agent.MirrorSharedRepoUrl)
	inherit(&in.Spec.Agent.MirrorSharedRepoUsername, &agent.Spec.Agent.MirrorSharedRepoUsername)

	if !reflect.DeepEqual(in.Spec.Agent.AdditionalBackends, agent.Spec.Agent.AdditionalBackends) {
		in.Spec.Agent.AdditionalBackends = agent.Spec.Agent.AdditionalBackends
	}

	if !reflect.DeepEqual(in.Spec.Agent.TlsSpec, agent.Spec.Agent.TlsSpec) {
		in.Spec.Agent.TlsSpec = agent.Spec.Agent.TlsSpec
	}

	if !reflect.DeepEqual(in.Spec.Agent.Pod.ResourceRequirements, in.Spec.ResourceRequirements) {
		in.Spec.Agent.Pod.ResourceRequirements = in.Spec.ResourceRequirements
	}
}

func (in *RemoteAgent) Default() {
	optional.ValueOrDefault(&in.Spec.Agent.ConfigurationYaml, in.Spec.ConfigurationYaml)
	optional.ValueOrDefault(&in.Spec.Agent.EndpointHost, "ingress-red-saas.instana.io")
	optional.ValueOrDefault(&in.Spec.Agent.EndpointPort, "443")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.Name, "icr.io/instana/agent")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.Tag, "latest")
	optional.ValueOrDefault(&in.Spec.Agent.ImageSpec.PullPolicy, corev1.PullAlways)
	optional.ValueOrDefault(&in.Spec.Rbac.Create, pointer.To(true))
	optional.ValueOrDefault(&in.Spec.ServiceAccountSpec.Create.Create, pointer.To(true))
	optional.ValueOrDefault(&in.Spec.Agent.Pod.ResourceRequirements, in.Spec.ResourceRequirements)
}

// +kubebuilder:object:root=true

// RemoteAgentList contains a list of RemoteAgent
type RemoteAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteAgent{}, &RemoteAgentList{})
}

func inherit[T comparable](target *T, source *T) {
	if *source != zeroValue[T]() && *target != *source {
		*target = *source
	}
}

func zeroValue[T any]() T {
	var zero T
	return zero
}
