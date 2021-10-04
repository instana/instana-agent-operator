/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1beta1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+k8s:openapi-gen=true

// InstanaAgentSpec defines the desired state of the Instana Agent
type InstanaAgentSpec struct {
	ConfigurationFiles          map[string]string `json:"config.files,omitempty"`
	AgentZoneName               string            `json:"agent.zone.name,omitempty"`
	AgentKey                    string            `json:"agent.key,omitempty"`
	AgentEndpointHost           string            `json:"agent.endpoint.host,omitempty"`
	AgentEndpointPort           uint16            `json:"agent.endpoint.port,omitempty"`
	AgentClusterRoleName        string            `json:"agent.clusterRoleName,omitempty"`
	AgentClusterRoleBindingName string            `json:"agent.clusterRoleBindingName,omitempty"`
	AgentServiceAccountName     string            `json:"agent.serviceAccountName,omitempty"`
	AgentSecretName             string            `json:"agent.secretName,omitempty"`
	AgentDaemonSetName          string            `json:"agent.daemonSetName,omitempty"`
	AgentConfigMapName          string            `json:"agent.configMapName,omitempty"`
	AgentRbacCreate             bool              `json:"agent.rbac.create,omitempty"`
	AgentImageName              string            `json:"agent.image,omitempty"`
	AgentImagePullPolicy        string            `json:"agent.imagePullPolicy,omitempty"`
	AgentCpuReq                 resource.Quantity `json:"agent.cpuReq,omitempty"`
	AgentCpuLim                 resource.Quantity `json:"agent.cpuLimit,omitempty"`
	AgentMemReq                 resource.Quantity `json:"agent.memReq,omitempty"`
	AgentMemLim                 resource.Quantity `json:"agent.memLimit,omitempty"`
	AgentDownloadKey            string            `json:"agent.downloadKey,omitempty"`
	AgentRepository             string            `json:"agent.host.repository,omitempty"`
	OpenTelemetryEnabled        bool              `json:"opentelemetry.enabled,omitempty"`
	ClusterName                 string            `json:"cluster.name,omitempty"`
	AgentEnv                    map[string]string `json:"agent.env,omitempty"`
}

//+k8s:openapi-gen=true

// ResourceInfo holds Name and UID to given object
type ResourceInfo struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
}

//+k8s:openapi-gen=true

// InstanaAgentStatus defines the observed state of InstanaAgent
type InstanaAgentStatus struct {
	ServiceAccount     ResourceInfo `json:"serviceaccount,omitempty"`
	ClusterRole        ResourceInfo `json:"clusterrole,omitempty"`
	ClusterRoleBinding ResourceInfo `json:"clusterrolebinding,omitempty"`
	Secret             ResourceInfo `json:"secret,omitempty"`
	ConfigMap          ResourceInfo `json:"configmap,omitempty"`
	DaemonSet          ResourceInfo `json:"daemonset,omitempty"`
	LeadingAgentPod    ResourceInfo `json:"leading.agent.pod,omitempty"`
}

//+kubebuilder:object:root=true
//+k8s:openapi-gen=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=agents,singular=agent,shortName=ia,scope=Namespaced,categories=monitoring;openshift-optional
//+operator-sdk:csv:customresourcedefinitions:displayName="Instana Agent", resources={{DaemonSet,v1,instana-agent},{Pod,v1,instana-agent},{Secret,v1,instana-agent}}

// InstanaAgent is the Schema for the agents API
type InstanaAgent struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   InstanaAgentSpec   `json:"spec,omitempty"`
	Status InstanaAgentStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// InstanaAgentList contains a list of InstanaAgent
type InstanaAgentList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []InstanaAgent `json:"items"`
}

func init() {
	SchemeBuilder.Register(&InstanaAgent{}, &InstanaAgentList{})
}
