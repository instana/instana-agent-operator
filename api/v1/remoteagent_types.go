/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +k8s:openapi-gen=true

// InstanaAgentSpec defines the desired state of the Instana Agent
type RemoteAgentSpec struct {
	// Agent deployment specific fields.
	// +kubebuilder:validation:Required
	Agent BaseAgentSpec `json:"agent"`

	Hostname Name `json:"hostname,omitempty"`

	Cluster Name `json:"cluster,omitempty"`

	// Name of the zone in which the host(s) will be displayed on the map. Optional, but then 'cluster.name' must be specified.
	Zone Name `json:"zone,omitempty"`

	// Specifies whether a ServiceAccount should be created (default `true`), and possibly the name to use.
	// +kubebuilder:validation:Optional
	ServiceAccountSpec `json:"serviceAccount,omitempty"`

	// Specifies whether to create the instana-agent `Service` to expose within the cluster. The Service can then be used e.g.
	// for the Prometheus remote-write, OpenTelemetry GRCP endpoint and other APIs.
	// Note: Requires Kubernetes 1.17+, as it uses topologyKeys.
	// +kubebuilder:validation:Optional
	Service Create `json:"service,omitempty"`
}

// +k8s:openapi-gen=true

// AgentOperatorState type representing the running state of the Agent Operator itself.
type RemoteAgentOperatorState string

const (
	// OperatorStateRunning the operator is running properly and all changes applied successfully.
	RemoteOperatorStateRunning RemoteAgentOperatorState = "Running"
	// OperatorStateUpdating the operator is running properly but is currently applying CR changes and getting the Agent in the correct state.
	RemoteOperatorStateUpdating RemoteAgentOperatorState = "Updating"
	// OperatorStateFailed the operator is not running properly and likely there were issues applying the CustomResource correctly.
	RemoteOperatorStateFailed RemoteAgentOperatorState = "Failed"
)

// +k8s:openapi-gen=true

// InstanaAgentStatus defines the observed state of InstanaAgent

// Deprecated: DeprecatedInstanaAgentStatus are the previous status fields that will be used to ensure backwards compatibility with any automation that may exist
type DeprecatedRemoteAgentStatus struct {
	Status     AgentOperatorState `json:"status,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	LastUpdate metav1.Time        `json:"lastUpdate,omitempty"`

	OldVersionsUpdated bool `json:"oldVersionsUpdated,omitempty"`

	ConfigMap       ResourceInfo            `json:"configmap,omitempty"` // no longer present, but keep it in the struct for backwards-compatibility
	Deployment      ResourceInfo            `json:"deployment,omitempty"`
	LeadingAgentPod map[string]ResourceInfo `json:"leadingAgentPod,omitempty"`
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
// +kubebuilder:resource:path=agents,singular=agent,shortName=ia,scope=Namespaced,categories=monitoring;openshift-optional
// +kubebuilder:storageversion
// +operator-sdk:csv:customresourcedefinitions:displayName="Instana Agent", resources={{DaemonSet,v1,instana-agent},{Pod,v1,instana-agent},{Secret,v1,instana-agent}}

// InstanaAgent is the Schema for the agents API
type RemoteAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   RemoteAgentSpec   `json:"spec,omitempty"`
	Status RemoteAgentStatus `json:"status,omitempty"`
}

func (in *RemoteAgent) Default(hostAgent InstanaAgent) {
	in.Spec.Agent.EndpointHost = hostAgent.Spec.Agent.EndpointHost
	in.Spec.Agent.EndpointPort = hostAgent.Spec.Agent.EndpointPort
	in.Spec.Agent.ImageSpec.Name = hostAgent.Spec.Agent.ImageSpec.Name
	in.Spec.Agent.ImageSpec.Tag = hostAgent.Spec.Agent.ImageSpec.Tag
	in.Spec.Agent.ImageSpec.PullPolicy = hostAgent.Spec.Agent.PullPolicy
	in.Spec.Agent.Key = hostAgent.Spec.Agent.Key
	in.Spec.Agent.DownloadKey = hostAgent.Spec.Agent.DownloadKey
	in.Spec.Agent.KeysSecret = hostAgent.Spec.Agent.KeysSecret
	in.Spec.Agent.ListenAddress = hostAgent.Spec.Agent.ListenAddress
	in.Spec.Agent.MinReadySeconds = hostAgent.Spec.Agent.MinReadySeconds
	in.Spec.Agent.AdditionalBackends = hostAgent.Spec.Agent.AdditionalBackends
	in.Spec.Agent.TlsSpec = hostAgent.Spec.Agent.TlsSpec
	in.Spec.Agent.ExtendedImageSpec = hostAgent.Spec.Agent.ExtendedImageSpec
	in.Spec.Agent.ProxyHost = hostAgent.Spec.Agent.ProxyHost
	in.Spec.Agent.ProxyPassword = hostAgent.Spec.Agent.ProxyPassword
	in.Spec.Agent.ProxyPort = hostAgent.Spec.Agent.ProxyPort
	in.Spec.Agent.ProxyProtocol = hostAgent.Spec.Agent.ProxyProtocol
	in.Spec.Agent.ProxyUseDNS = hostAgent.Spec.Agent.ProxyUseDNS
	in.Spec.Agent.ProxyUser = hostAgent.Spec.Agent.ProxyUser
	in.Spec.Agent.Env = hostAgent.Spec.Agent.Env
	in.Spec.Agent.RedactKubernetesSecrets = hostAgent.Spec.Agent.RedactKubernetesSecrets
	in.Spec.Agent.MvnRepoFeaturesPath = hostAgent.Spec.Agent.MvnRepoFeaturesPath
	in.Spec.Agent.MvnRepoSharedPath = hostAgent.Spec.Agent.MvnRepoSharedPath
	in.Spec.Agent.MvnRepoUrl = hostAgent.Spec.Agent.MvnRepoUrl
	in.Spec.Agent.MirrorReleaseRepoPassword = hostAgent.Spec.Agent.MirrorReleaseRepoPassword
	in.Spec.Agent.MirrorReleaseRepoUrl = hostAgent.Spec.Agent.MirrorReleaseRepoUrl
	in.Spec.Agent.MirrorReleaseRepoUsername = hostAgent.Spec.Agent.MirrorReleaseRepoUsername
	in.Spec.Agent.MirrorSharedRepoPassword = hostAgent.Spec.Agent.MirrorSharedRepoPassword
	in.Spec.Agent.MirrorSharedRepoUrl = hostAgent.Spec.Agent.MirrorSharedRepoUrl
	in.Spec.Agent.MirrorSharedRepoUsername = hostAgent.Spec.Agent.MirrorSharedRepoUsername
}

// +kubebuilder:object:root=true

// InstanaAgentList contains a list of InstanaAgent
type RemoteAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RemoteAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RemoteAgent{}, &RemoteAgentList{})
}
