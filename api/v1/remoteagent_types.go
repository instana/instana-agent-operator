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

func (in *RemoteAgent) Default(agent InstanaAgent) {
	// Compute desired values from the upstream agent spec with defaults.
	desiredEndpointHost := optional.Of(agent.Spec.Agent.EndpointHost).GetOrDefault("ingress-red-saas.instana.io")
	if in.Spec.Agent.EndpointHost != desiredEndpointHost {
		in.Spec.Agent.EndpointHost = desiredEndpointHost
	}

	desiredEndpointPort := optional.Of(agent.Spec.Agent.EndpointPort).GetOrDefault("443")
	if in.Spec.Agent.EndpointPort != desiredEndpointPort {
		in.Spec.Agent.EndpointPort = desiredEndpointPort
	}

	desiredImageName := optional.Of(agent.Spec.Agent.ImageSpec.Name).GetOrDefault("icr.io/instana/agent")
	if in.Spec.Agent.ImageSpec.Name != desiredImageName {
		in.Spec.Agent.ImageSpec.Name = desiredImageName
	}

	desiredImageTag := optional.Of(agent.Spec.Agent.ImageSpec.Tag).GetOrDefault("latest")
	if in.Spec.Agent.ImageSpec.Tag != desiredImageTag {
		in.Spec.Agent.ImageSpec.Tag = desiredImageTag
	}

	desiredPullPolicy := optional.Of(agent.Spec.Agent.ImageSpec.PullPolicy).GetOrDefault(corev1.PullAlways)
	if in.Spec.Agent.ImageSpec.PullPolicy != desiredPullPolicy {
		in.Spec.Agent.ImageSpec.PullPolicy = desiredPullPolicy
	}

	desiredRbac := optional.Of(agent.Spec.Rbac.Create).GetOrDefault(pointer.To(true))
	if !boolPointerEqual(in.Spec.Rbac.Create, desiredRbac) {
		in.Spec.Rbac.Create = desiredRbac
	}

	desiredSA := optional.Of(agent.Spec.ServiceAccountSpec.Create.Create).GetOrDefault(pointer.To(true))
	if !boolPointerEqual(in.Spec.ServiceAccountSpec.Create.Create, desiredSA) {
		in.Spec.ServiceAccountSpec.Create.Create = desiredSA
	}

	if in.Spec.Agent.ConfigurationYaml != in.Spec.ConfigurationYaml {
		in.Spec.Agent.ConfigurationYaml = in.Spec.ConfigurationYaml
	}

	if in.Spec.Cluster.Name != agent.Spec.Cluster.Name {
		in.Spec.Cluster.Name = agent.Spec.Cluster.Name
	}

	if in.Spec.Agent.Key != agent.Spec.Agent.Key {
		in.Spec.Agent.Key = agent.Spec.Agent.Key
	}

	if in.Spec.Agent.DownloadKey != agent.Spec.Agent.DownloadKey {
		in.Spec.Agent.DownloadKey = agent.Spec.Agent.DownloadKey
	}

	if in.Spec.Agent.KeysSecret != agent.Spec.Agent.KeysSecret {
		in.Spec.Agent.KeysSecret = agent.Spec.Agent.KeysSecret
	}

	if in.Spec.Agent.ListenAddress != agent.Spec.Agent.ListenAddress {
		in.Spec.Agent.ListenAddress = agent.Spec.Agent.ListenAddress
	}

	if in.Spec.Agent.MinReadySeconds != agent.Spec.Agent.MinReadySeconds {
		in.Spec.Agent.MinReadySeconds = agent.Spec.Agent.MinReadySeconds
	}

	if !reflect.DeepEqual(in.Spec.Agent.AdditionalBackends, agent.Spec.Agent.AdditionalBackends) {
		in.Spec.Agent.AdditionalBackends = agent.Spec.Agent.AdditionalBackends
	}

	if !reflect.DeepEqual(in.Spec.Agent.TlsSpec, agent.Spec.Agent.TlsSpec) {
		in.Spec.Agent.TlsSpec = agent.Spec.Agent.TlsSpec
	}

	if (agent.Spec.Agent.ExtendedImageSpec.ImageSpec.Name != "" || len(agent.Spec.Agent.ExtendedImageSpec.PullSecrets) > 0) &&
		(in.Spec.Agent.ExtendedImageSpec.ImageSpec.Name != agent.Spec.Agent.ExtendedImageSpec.ImageSpec.Name ||
			!reflect.DeepEqual(in.Spec.Agent.ExtendedImageSpec.PullSecrets, agent.Spec.Agent.ExtendedImageSpec.PullSecrets)) {
		in.Spec.Agent.ExtendedImageSpec = agent.Spec.Agent.ExtendedImageSpec
	}

	if in.Spec.Agent.ProxyHost != agent.Spec.Agent.ProxyHost {
		in.Spec.Agent.ProxyHost = agent.Spec.Agent.ProxyHost
	}

	if in.Spec.Agent.ProxyPassword != agent.Spec.Agent.ProxyPassword {
		in.Spec.Agent.ProxyPassword = agent.Spec.Agent.ProxyPassword
	}

	if in.Spec.Agent.ProxyPort != agent.Spec.Agent.ProxyPort {
		in.Spec.Agent.ProxyPort = agent.Spec.Agent.ProxyPort
	}

	if in.Spec.Agent.ProxyProtocol != agent.Spec.Agent.ProxyProtocol {
		in.Spec.Agent.ProxyProtocol = agent.Spec.Agent.ProxyProtocol
	}

	if in.Spec.Agent.ProxyUseDNS != agent.Spec.Agent.ProxyUseDNS {
		in.Spec.Agent.ProxyUseDNS = agent.Spec.Agent.ProxyUseDNS
	}

	if in.Spec.Agent.ProxyUser != agent.Spec.Agent.ProxyUser {
		in.Spec.Agent.ProxyUser = agent.Spec.Agent.ProxyUser
	}

	if !reflect.DeepEqual(in.Spec.Agent.Env, agent.Spec.Agent.Env) {
		in.Spec.Agent.Env = agent.Spec.Agent.Env
	}

	if in.Spec.Agent.RedactKubernetesSecrets != agent.Spec.Agent.RedactKubernetesSecrets {
		in.Spec.Agent.RedactKubernetesSecrets = agent.Spec.Agent.RedactKubernetesSecrets
	}

	if in.Spec.Agent.MvnRepoFeaturesPath != agent.Spec.Agent.MvnRepoFeaturesPath {
		in.Spec.Agent.MvnRepoFeaturesPath = agent.Spec.Agent.MvnRepoFeaturesPath
	}

	if in.Spec.Agent.MvnRepoSharedPath != agent.Spec.Agent.MvnRepoSharedPath {
		in.Spec.Agent.MvnRepoSharedPath = agent.Spec.Agent.MvnRepoSharedPath
	}

	if in.Spec.Agent.MvnRepoUrl != agent.Spec.Agent.MvnRepoUrl {
		in.Spec.Agent.MvnRepoUrl = agent.Spec.Agent.MvnRepoUrl
	}

	if in.Spec.Agent.MirrorReleaseRepoPassword != agent.Spec.Agent.MirrorReleaseRepoPassword {
		in.Spec.Agent.MirrorReleaseRepoPassword = agent.Spec.Agent.MirrorReleaseRepoPassword
	}

	if in.Spec.Agent.MirrorReleaseRepoUrl != agent.Spec.Agent.MirrorReleaseRepoUrl {
		in.Spec.Agent.MirrorReleaseRepoUrl = agent.Spec.Agent.MirrorReleaseRepoUrl
	}

	if in.Spec.Agent.MirrorReleaseRepoUsername != agent.Spec.Agent.MirrorReleaseRepoUsername {
		in.Spec.Agent.MirrorReleaseRepoUsername = agent.Spec.Agent.MirrorReleaseRepoUsername
	}

	if in.Spec.Agent.MirrorSharedRepoPassword != agent.Spec.Agent.MirrorSharedRepoPassword {
		in.Spec.Agent.MirrorSharedRepoPassword = agent.Spec.Agent.MirrorSharedRepoPassword
	}

	if in.Spec.Agent.MirrorSharedRepoUrl != agent.Spec.Agent.MirrorSharedRepoUrl {
		in.Spec.Agent.MirrorSharedRepoUrl = agent.Spec.Agent.MirrorSharedRepoUrl
	}

	if in.Spec.Agent.MirrorSharedRepoUsername != agent.Spec.Agent.MirrorSharedRepoUsername {
		in.Spec.Agent.MirrorSharedRepoUsername = agent.Spec.Agent.MirrorSharedRepoUsername
	}
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

func boolPointerEqual(a, b *bool) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return *a == *b
}
