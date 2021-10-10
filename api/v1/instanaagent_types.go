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
	Agent BaseAgentSpec `json:"agent,omitempty"`

	// cluster.name represents the name that will be assigned to this cluster in Instana
	Cluster Name `json:"cluster,omitempty"`

	OpenShift bool `json:"openshift,omitempty"`

	// Specifies whether RBAC resources should be created
	Rbac Create `json:"rbac,omitempty"`
	// Specifies whether to create the instana-agent service to expose within the cluster the Prometheus remote-write, OpenTelemetry GRCP endpoint and other APIs
	// Note: Requires Kubernetes 1.17+, as it uses topologyKeys
	Service Create `json:"service,omitempty"`
	// If true, it will also apply `service.create=true`
	OpenTelemetry Enabled `json:"opentelemetry,omitempty"`

	Prometheus `json:"prometheus,omitempty"`
	// Specifies whether a ServiceAccount should be created
	// The name of the ServiceAccount to use.
	// If not set and `create` is true, a name is generated using the fullname template
	// name: instana-agent
	ServiceAccount Create `json:"serviceAccount,omitempty"`

	PodSecurityPolicySpec `json:"podSecurityPolicy,omitempty"`

	Zone Name `json:"zone,omitempty"`

	KubernetesSpec `json:"kubernetes,omitempty"`

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

//+k8s:openapi-gen=true

// InstanaAgentStatus defines the observed state of InstanaAgent
type InstanaAgentStatus struct {
	ConfigMap       ResourceInfo `json:"configmap,omitempty"`
	DaemonSet       ResourceInfo `json:"daemonset,omitempty"`
	LeadingAgentPod ResourceInfo `json:"leadingAgentPod,omitempty"`

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
