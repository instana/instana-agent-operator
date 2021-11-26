/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */

package v1

import (
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

//+k8s:openapi-gen=true

// InstanaAgentSpec defines the desired state of the Instana Agent
type InstanaAgentSpec struct {
	// Agent deployment specific fields.
	// +kubebuilder:validation:Required
	Agent BaseAgentSpec `json:"agent"`

	// Name of the cluster, that will be assigned to this cluster in Instana. Either specifying the 'cluster.name' or 'zone.name'
	// is mandatory.
	Cluster Name `json:"cluster,omitempty"`

	// Name of the zone in which the host(s) will be displayed on the map. Optional, but then 'cluster.name' must be specified.
	Zone Name `json:"zone,omitempty"`

	// Set to `True` to indicate the Operator is being deployed in a OpenShift cluster. Provides a hint so that RBAC etc is
	// configured correctly.
	// +kubebuilder:validation:Optional
	OpenShift bool `json:"openshift,omitempty"`

	// Specifies whether RBAC resources should be created.
	// +kubebuilder:validation:Optional
	Rbac Create `json:"rbac,omitempty"`

	// Specifies whether to create the instana-agent `Service` to expose within the cluster. The Service can then be used e.g.
	// for the Prometheus remote-write, OpenTelemetry GRCP endpoint and other APIs.
	// Note: Requires Kubernetes 1.17+, as it uses topologyKeys.
	// +kubebuilder:validation:Optional
	Service Create `json:"service,omitempty"`

	// Enables the OpenTelemetry gRPC endpoint on the Agent. If true, it will also apply `service.create: true`.
	// +kubebuilder:validation:Optional
	OpenTelemetry Enabled `json:"opentelemetry,omitempty"`

	// Enables the Prometheus endpoint on the Agent. If true, it will also apply `service.create: true`.
	// +kubebuilder:validation:Optional
	Prometheus `json:"prometheus,omitempty"`

	// Specifies whether a ServiceAccount should be created (default `true`), and possibly the name to use.
	// +kubebuilder:validation:Optional
	ServiceAccountSpec `json:"serviceAccount,omitempty"`

	// Specify a PodSecurityPolicy for the Instana Agent Pods. If enabled requires `rbac.create: true`.
	// +kubebuilder:validation:Optional
	PodSecurityPolicySpec `json:"podSecurityPolicy,omitempty"`

	// Allows for installment of the Kubernetes Sensor as separate pod. Which allows for better tailored resource settings
	// (mainly memory) both for the Agent pods and the Kubernetes Sensor pod.
	// +kubebuilder:validation:Optional
	KubernetesSpec `json:"kubernetes,omitempty"`

	// Specifying the PinnedChartVersion allows for 'pinning' the Helm Chart used by the Operator for installing the Agent
	// DaemonSet. Normally the Operator will always install and update to the latest Helm Chart version.
	// The Operator will check and make sure no 'unsupported' Chart versions can be selected.
	// +kubebuilder:validation:Optional
	PinnedChartVersion string `json:"pinnedChartVersion,omitempty"`

	//
	// OLD v1beta1 spec which, by including temporarily, we can provide backwards compatibility for the v1beta1 spec as served
	// by the Java Operator. Prevents having to modify the CR outside the Operator.
	//

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
	AgentTlsSecretName          string            `json:"agent.tls.secretName,omitempty"`
	AgentTlsCertificate         string            `json:"agent.tls.certificate,omitempty"`
	AgentTlsKey                 string            `json:"agent.tls.key,omitempty"`
	OpenTelemetryEnabled        bool              `json:"opentelemetry.enabled,omitempty"`
	ClusterName                 string            `json:"cluster.name,omitempty"`
	AgentEnv                    map[string]string `json:"agent.env,omitempty"`
	// END of OLD spec
}

//+k8s:openapi-gen=true

// ResourceInfo holds Name and UID to given object
type ResourceInfo struct {
	Name string `json:"name"`
	UID  string `json:"uid"`
}

// AgentOperatorState type representing the running state of the Agent Operator itself.
type AgentOperatorState string

const (
	// OperatorStateRunning the operator is running properly and all changes applied successfully.
	OperatorStateRunning AgentOperatorState = "Running"
	// OperatorStateUpdating the operator is running properly but is currently applying CR changes and getting the Agent in the correct state.
	OperatorStateUpdating AgentOperatorState = "Updating"
	// OperatorStateFailed the operator is not running properly and likely there were issues applying the CustomResource correctly.
	OperatorStateFailed AgentOperatorState = "Failed"
)

//+k8s:openapi-gen=true

// InstanaAgentStatus defines the observed state of InstanaAgent
type InstanaAgentStatus struct {
	Status     AgentOperatorState `json:"status,omitempty"`
	Reason     string             `json:"reason,omitempty"`
	LastUpdate metav1.Time        `json:"lastUpdate,omitempty"`

	OldVersionsUpdated bool `json:"oldVersionsUpdated,omitempty"`

	ConfigMap       ResourceInfo            `json:"configmap,omitempty"`
	DaemonSet       ResourceInfo            `json:"daemonset,omitempty"`
	LeadingAgentPod map[string]ResourceInfo `json:"leadingAgentPod,omitempty"`

	// Other Status fields that need to be included for backwards compatibility (Conversion WebHook needs to result in same CR
	// when converting back and forth)

	ServiceAccount     ResourceInfo `json:"serviceaccount,omitempty"`
	ClusterRole        ResourceInfo `json:"clusterrole,omitempty"`
	ClusterRoleBinding ResourceInfo `json:"clusterrolebinding,omitempty"`
	Secret             ResourceInfo `json:"secret,omitempty"`
}

//+kubebuilder:object:root=true
//+k8s:openapi-gen=true
//+kubebuilder:subresource:status
//+kubebuilder:resource:path=agents,singular=agent,shortName=ia,scope=Namespaced,categories=monitoring;openshift-optional
//+kubebuilder:storageversion
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
