/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1beta1

import (
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// InstanaAgentSpec defines the desired state of the Instana Agent
// +k8s:openapi-gen=true
type InstanaAgentSpec struct {
	Agent *BaseAgentSpec `json:"agent,omitempty"`

	// cluster.name represents the name that will be assigned to this cluster in Instana
	Cluster *Name `json:"cluster,omitempty"`

	OpenShift bool `json:"openshift,omitempty"`

	// Specifies whether RBAC resources should be created
	Rbac *Create `json:"rbac,omitempty"`
	// Specifies whether to create the instana-agent service to expose within the cluster the Prometheus remote-write, OpenTelemetry GRCP endpoint and other APIs
	// Note: Requires Kubernetes 1.17+, as it uses topologyKeys
	Service *Create `json:"service,omitempty"`
	// If true, it will also apply `service.create=true`
	OpenTelemetry *Enabled `json:"opentelemetry,omitempty"`

	Prometheus *Prometheus `json:"prometheus,omitempty"`
	// Specifies whether a ServiceAccount should be created
	// The name of the ServiceAccount to use.
	// If not set and `create` is true, a name is generated using the fullname template
	// name: instana-agent
	ServiceAccount *Create `json:"serviceAccount,omitempty"`

	PodSecurityPolicy *PodSecurityPolicySpec `json:"podSecurityPolicy,omitempty"`

	Zone *Name `json:"zone,omitempty"`

	Kuberentes *K8sSpec `json:"kubernetes,omitempty"`
}

// InstanaAgentStatus defines the observed state of InstanaAgent
// +k8s:openapi-gen=true
type InstanaAgentStatus struct {
	//Status of each config
	ConfigStatusses []appsv1.DaemonSetStatus `json:"configStatusses,omitempty"`
}

// Defines the desired specs and states for instana agent deployment.
// +kubebuilder:object:root=true
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
// +kubebuilder:resource:path=instanaagent,scope=Namespaced,categories=instana
// +operator-sdk:csv:customresourcedefinitions:displayName="Instana Agent", resources={{DaemonSet,v1,instana-agent},{Pod,v1,instana-agent},{Secret,v1,instana-agent}}
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
