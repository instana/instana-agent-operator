/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1beta1

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

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
	ZoneName string `json:"zoneName,omitempty"`

	// The name of your kubernetes cluster
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ClusterName string `json:"clusterName,omitempty"`

	// Set your Instana agent key
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent Key"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:io.kubernetes:Secret"
	Key string `json:"key,omitempty"`

	// Set agent's endpoint
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent Endpoint"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Endpoint *InstanaAgentEndpoint `json:"endpoint,omitempty"`

	// Set environment vars
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent environment variables"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Configuration files
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.displayName="Agent configuration files"
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:advanced,urn:alm:descriptor:com.tectonic.ui:text"
	ConfigFiles map[string]string `json:"configFiles,omitempty"`

	// Kubernetes Cluster Role Name
	// Defaults to instana-agent
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ClusterRoleName string `json:"clusterRoleName,omitempty"`

	// Kubernetes Cluster Role Binding Name
	// Defaults to instana-agent
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ClusterRoleBindingName string `json:"clusterRoleBindingName,omitempty"`

	// Kubernetes Service Account Name
	// Defaults to instana-agent
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Kubernetes Secret Name
	// Defaults to instana-agent
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	SecretName string `json:"secretName,omitempty"`

	// Kubernetes Daemonset Name
	// Defaults to instana-agent
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	DaemonSetName string `json:"daemonSetName,omitempty"`

	// Kubernetes ConfigMap Name
	// Defaults to instana-agent
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ConfigMapName string `json:"configMapName,omitempty"`

	// Rbac creation
	// Defaults to true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	RbacCreation bool `json:"rbacCreation,omitempty"`

	// Instana agent image
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	Image string `json:"image,omitempty"`

	// Kubernetes Image pull policy
	// Defaults to Always
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// CpuReq
	// Defaults to 0.5
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	CpuReq string `json:"cpuReq,omitempty"`

	// CpuLimit
	// Defaults to 1.5
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	CpuLimit string `json:"cpuLimit,omitempty"`

	// MemReq
	// Defaults to 512
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:number"
	MemReq byte `json:"memReq,omitempty"`

	// MemLimit
	// Defaults to 512
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:number"
	MemLimit byte `json:"memLimit,omitempty"`

	// Instana agent download key
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	DownloadKey string `json:"downloadKey,omitempty"`

	// Instana agent host repository
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:text"
	HostRepository string `json:"hostRepository,omitempty"`

	// OpenTelemetry enabled
	// Defaults to false
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors=true
	// +operator-sdk:gen-csv:customresourcedefinitions.specDescriptors.x-descriptors="urn:alm:descriptor:com.tectonic.ui:booleanSwitch"
	OpenTelemetryEnabled bool `json:"opentelemetryEnabled,omitempty"`
}

// InstanaAgentStatus defines the observed state of InstanaAgent
// +k8s:openapi-gen=true
type InstanaAgentStatus struct {
	//Status of each config
	ConfigStatusses []appsv1.DeploymentStatus `json:"configStatusses,omitempty"`
}

// +kubebuilder:object:root=true

// Defines the desired specs and states for instana agent deployment.
// +k8s:openapi-gen=true
// +kubebuilder:resource:path=instanaagent,scope=Namespaced,categories=instana
// +operator-sdk:gen-csv:customresourcedefinitions.displayName="Instana Agent"
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
