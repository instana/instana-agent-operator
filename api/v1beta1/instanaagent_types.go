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
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ZoneName string `json:"zoneName,omitempty"`

	// The name of your kubernetes cluster
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterName string `json:"clusterName,omitempty"`

	// Set your Instana agent key
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Agent Key",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	Key string `json:"key,omitempty"`

	// Set agent's endpoint
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Agent Endpoint",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Endpoint *InstanaAgentEndpoint `json:"endpoint,omitempty"`

	// Set environment vars
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Agent environment variables",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	Env []corev1.EnvVar `json:"env,omitempty"`

	// Configuration files
	//+operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Agent configuration files",xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:text"}
	ConfigFiles map[string]string `json:"configFiles,omitempty"`

	// Optional: Kubernetes Cluster Role Name
	// Defaults to instana-agent
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterRoleName string `json:"clusterRoleName,omitempty"`

	// Optional: Kubernetes Cluster Role Binding Name
	// Defaults to instana-agent
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ClusterRoleBindingName string `json:"clusterRoleBindingName,omitempty"`

	// Optional: Kubernetes Service Account Name
	// Defaults to instana-agent
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ServiceAccountName string `json:"serviceAccountName,omitempty"`

	// Optional: Kubernetes Secret Name
	// Defaults to instana-agent
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	SecretName string `json:"secretName,omitempty"`

	// Optional: Kubernetes Daemonset Name
	// Defaults to instana-agent-daemonset
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	DaemonSetName string `json:"daemonSetName,omitempty"`

	// Optional: Kubernetes ConfigMap Name
	// Defaults to instana-agent
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ConfigMapName string `json:"configMapName,omitempty"`

	// Optional: Rbac creation
	// Defaults to true
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
	RbacCreation bool `json:"rbacCreation,omitempty"`

	// Optional: Instana agent image
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	Image string `json:"image,omitempty"`

	// Optional: Kubernetes Image pull policy
	// Defaults to Always
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	ImagePullPolicy string `json:"imagePullPolicy,omitempty"`

	// Optional: Define resources requests and limits for single pods
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:advanced","urn:alm:descriptor:com.tectonic.ui:resourceRequirements"}
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Instana agent download key
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	DownloadKey string `json:"downloadKey,omitempty"`

	// Optional: Instana agent host repository
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:text"}
	HostRepository string `json:"hostRepository,omitempty"`

	// Optional: OpenTelemetry enabled
	// Defaults to false
	//+operator-sdk:csv:customresourcedefinitions:type=spec,xDescriptors={"urn:alm:descriptor:com.tectonic.ui:booleanSwitch"}
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
// +operator-sdk:csv:customresourcedefinitions:displayName="Instana Agent", resources={{DaemonSet,v1,instana-agent-daemonset},{Pod,v1,instana-agent-pod},{Service,v1,instana-agent-service}}
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
