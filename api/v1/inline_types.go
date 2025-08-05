/*
 (c) Copyright IBM Corp. 2021, 2025
*/

package v1

import (
	"fmt"
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"

	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
)

type AgentMode string

const (
	APM            AgentMode = "APM"
	INFRASTRUCTURE AgentMode = "INFRASTRUCTURE"
	AWS            AgentMode = "AWS"
	KUBERNETES     AgentMode = "KUBERNETES"
)

type Name struct {
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`
}

type Create struct {
	// +kubebuilder:validation:Optional
	Create *bool `json:"create,omitempty"`
}

type Enabled struct {
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
}

func (e Enabled) String() string {
	if e.Enabled == nil {
		return "nil"
	} else {
		return strconv.FormatBool(*e.Enabled)
	}
}

// BaseAgentSpec defines the desired state info related to the running Agent
// +k8s:openapi-gen=true
type BaseAgentSpec struct {
	// Set agent mode, possible options are APM, INFRASTRUCTURE or AWS. KUBERNETES should not be used but instead enabled via
	// `kubernetes.deployment.enabled: true`.
	// +kubebuilder:validation:Optional
	Mode AgentMode `json:"mode,omitempty"`

	// Key is the secret token which your agent uses to authenticate to Instana's servers.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Agent Key",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Optional
	Key string `json:"key,omitempty"`

	// The DownloadKey, sometimes known as "sales key", that allows you to download software from Instana. It might be needed to
	// specify this in addition to the Key.
	// +kubebuilder:validation:Optional
	DownloadKey string `json:"downloadKey,omitempty"`

	// Rather than specifying the Key and optionally the DownloadKey, you can "bring your
	// own secret" creating it in the namespace in which you install the `instana-agent` and
	// specify its name in the `KeysSecret` field. The secret you create must contain a field called `key` and optionally one
	// called `downloadKey`, which contain, respectively, the values you'd otherwise set in `.agent.key` and `agent.downloadKey`.
	// +kubebuilder:validation:Optional
	KeysSecret string `json:"keysSecret,omitempty"`

	// ListenAddress is the IP addresses the Agent HTTP server will listen on. Normally this will just be localhost (`127.0.0.1`),
	// the pod public IP and any container runtime bridge interfaces. Set `listenAddress: *` for making the Agent listen on all
	// network interfaces.
	// +kubebuilder:validation:Optional
	ListenAddress string `json:"listenAddress,omitempty"`

	// EndpointHost is the hostname of the Instana server your agents will connect to.
	// +kubebuilder:validation:Required
	EndpointHost string `json:"endpointHost"`

	// EndpointPort is the port number (as a String) of the Instana server your agents will connect to.
	// +kubebuilder:validation:Required
	EndpointPort string `json:"endpointPort"`

	// The minimum number of seconds for which a newly created Pod should be ready without any of its containers crashing, for it to be considered available
	// +kubebuilder:validation:Optional
	MinReadySeconds int `json:"minReadySeconds,omitempty"`

	// These are additional backends the Instana agent will report to besides
	// the one configured via the `agent.endpointHost`, `agent.endpointPort` and `agent.key` setting.
	// +kubebuilder:validation:Optional
	AdditionalBackends []BackendSpec `json:"additionalBackends,omitempty"`

	// TLS for end-to-end encryption between the Instana Agent and clients accessing the Agent.
	// The Instana Agent does not yet allow enforcing TLS encryption, enabling makes it possible for clients to 'opt-in'.
	// So TLS is only enabled on a connection when requested by the client.
	// +kubebuilder:validation:Optional
	TlsSpec `json:"tls,omitempty"`

	// Override the container image used for the Instana Agent pods.
	// +kubebuilder:validation:Optional
	ExtendedImageSpec `json:"image,omitempty"`

	// Control how to update the Agent DaemonSet
	// +kubebuilder:validation:Optional
	UpdateStrategy appsv1.DaemonSetUpdateStrategy `json:"updateStrategy,omitempty"`

	// Override Agent Pod specific settings such as annotations, labels and resources.
	// +kubebuilder:validation:Optional
	Pod AgentPodSpec `json:"pod,omitempty"`

	// proxyHost sets the INSTANA_AGENT_PROXY_HOST environment variable.
	// +kubebuilder:validation:Optional
	ProxyHost string `json:"proxyHost,omitempty"`
	// proxyPort sets the INSTANA_AGENT_PROXY_PORT environment variable.
	// +kubebuilder:validation:Optional
	ProxyPort string `json:"proxyPort,omitempty"`
	// proxyProtocol sets the INSTANA_AGENT_PROXY_PROTOCOL environment variable.
	// +kubebuilder:validation:Optional
	ProxyProtocol string `json:"proxyProtocol,omitempty"`
	// proxyUser sets the INSTANA_AGENT_PROXY_USER environment variable.
	// +kubebuilder:validation:Optional
	ProxyUser string `json:"proxyUser,omitempty"`
	// proxyPassword sets the INSTANA_AGENT_PROXY_PASSWORD environment variable.
	// +kubebuilder:validation:Optional
	ProxyPassword string `json:"proxyPassword,omitempty"`
	// proxyUseDNS sets the INSTANA_AGENT_PROXY_USE_DNS environment variable.
	// +kubebuilder:validation:Optional
	ProxyUseDNS bool `json:"proxyUseDNS,omitempty"`

	// Use the `env` field to set additional environment variables for the Instana Agent, for example:
	// env:
	//   INSTANA_AGENT_TAGS: dev
	// +kubebuilder:validation:Optional
	Env map[string]string `json:"env,omitempty"`

	// Supply Agent configuration e.g. for configuring certain Sensors.
	// +kubebuilder:validation:Optional
	ConfigurationYaml string `json:"configuration_yaml,omitempty"`

	// RedactKubernetesSecrets sets the INSTANA_KUBERNETES_REDACT_SECRETS environment variable.
	// +kubebuilder:validation:Optional
	RedactKubernetesSecrets string `json:"redactKubernetesSecrets,omitempty"`

	// Host sets a host path to be mounted as the Agent Maven repository (mainly for debugging or development purposes)
	// +kubebuilder:validation:Optional
	Host HostSpec `json:"host,omitempty"`

	// ServiceMesh sets the ENABLE_AGENT_SOCKET environment variable.
	// +kubebuilder:validation:Optional
	ServiceMesh ServiceMeshSpec `json:"serviceMesh,omitempty"`

	// Override for the Maven repository URL when the Agent needs to connect to a locally provided Maven repository 'proxy'
	// Alternative to `Host` for referencing a different Maven repo.
	// +kubebuilder:validation:Optional
	MvnRepoUrl string `json:"instanaMvnRepoUrl,omitempty"`
	// Sets the INSTANA_MVN_REPOSITORY_FEATURES_PATH environment variable
	// +kubebuilder:validation:Optional
	MvnRepoFeaturesPath string `json:"instanaMvnRepoFeaturesPath,omitempty"`
	// Sets the INSTANA_MVN_REPOSITORY_SHARED_PATH environment variable
	// +kubebuilder:validation:Optional
	MvnRepoSharedPath string `json:"instanaMvnRepoSharedPath,omitempty"`
	// URLs to dependency files that will be fetched via an init container and shared with the agent pod
	// +kubebuilder:validation:Optional
	DependencyURLs []string `json:"dependencyURLs,omitempty"`
	// Sets the AGENT_RELEASE_REPOSITORY_MIRROR_URL environment variable
	// +kubebuilder:validation:Optional
	MirrorReleaseRepoUrl string `json:"agentReleaseRepoMirrorUrl,omitempty"`
	// Sets the AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME environment variable
	// +kubebuilder:validation:Optional
	MirrorReleaseRepoUsername string `json:"agentReleaseRepoMirrorUsername,omitempty"`
	// Sets the AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD environment variable
	// +kubebuilder:validation:Optional
	MirrorReleaseRepoPassword string `json:"agentReleaseRepoMirrorPassword,omitempty"`
	// Sets the INSTANA_SHARED_REPOSITORY_MIRROR_URL environment variable
	// +kubebuilder:validation:Optional
	MirrorSharedRepoUrl string `json:"instanaSharedRepoMirrorUrl,omitempty"`
	// Sets the INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME environment variable
	// +kubebuilder:validation:Optional
	MirrorSharedRepoUsername string `json:"instanaSharedRepoMirrorUsername,omitempty"`
	// Sets the INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD environment variable
	// +kubebuilder:validation:Optional
	MirrorSharedRepoPassword string `json:"instanaSharedRepoMirrorPassword,omitempty"`
}

type ResourceRequirements corev1.ResourceRequirements

func (r ResourceRequirements) GetOrDefault() corev1.ResourceRequirements {
	requestsDefaulter := map_defaulter.NewMapDefaulter((*map[corev1.ResourceName]resource.Quantity)(&r.Requests))
	requestsDefaulter.SetIfEmpty(corev1.ResourceMemory, resource.MustParse("768Mi"))
	requestsDefaulter.SetIfEmpty(corev1.ResourceCPU, resource.MustParse("0.5"))

	limitsDefaulter := map_defaulter.NewMapDefaulter((*map[corev1.ResourceName]resource.Quantity)(&r.Limits))
	limitsDefaulter.SetIfEmpty(corev1.ResourceMemory, resource.MustParse("768Mi"))
	limitsDefaulter.SetIfEmpty(corev1.ResourceCPU, resource.MustParse("1.5"))

	return corev1.ResourceRequirements(r)
}

type AgentPodSpec struct {
	// agent.pod.annotations are additional annotations to be added to the agent pods.
	// +kubebuilder:validation:Optional
	Annotations map[string]string `json:"annotations,omitempty"`

	// agent.pod.labels are additional labels to be added to the agent pods.
	// +kubebuilder:validation:Optional
	Labels map[string]string `json:"labels,omitempty"`

	// agent.pod.tolerations are tolerations to influence agent pod assignment.
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// agent.pod.affinity are affinities to influence agent pod assignment.
	// https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
	// +kubebuilder:validation:Optional
	Affinity corev1.Affinity `json:"affinity,omitempty"`

	// agent.pod.priorityClassName is the name of an existing PriorityClass that should be set on the agent pods
	// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// Override Agent resource requirements to e.g. give the Agent container more memory.
	ResourceRequirements `json:",inline"`

	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Set additional volumes for the agent pod.
	// +kubebuilder:validation:Optional
	Volumes []corev1.Volume `json:"volumes,omitempty"`

	// Set additional volume mounts for the agent pod.
	// +kubebuilder:validation:Optional
	VolumeMounts []corev1.VolumeMount `json:"volumeMounts,omitempty"`

	// Set additional environment variables for the agent pod.
	// +kubebuilder:validation:Optional
	Env []corev1.EnvVar `json:"env,omitempty"`
}

type TlsSpec struct {
	// secretName is the name of the secret that has the relevant files.
	// +kubebuilder:validation:Optional
	SecretName string `json:"secretName,omitempty"`
	// certificate (together with key) is the alternative to an existing Secret. Must be base64 encoded.
	// +kubebuilder:validation:Optional
	Certificate []byte `json:"certificate,omitempty"`
	// key (together with certificate) is the alternative to an existing Secret. Must be base64 encoded.
	// +kubebuilder:validation:Optional
	Key []byte `json:"key,omitempty"`
}

type ImageSpec struct {
	// Name is the name of the container image of the Instana agent.
	// +kubebuilder:validation:Optional
	Name string `json:"name,omitempty"`

	// Digest (a.k.a. Image ID) of the agent container image. If specified, it has priority over `agent.image.tag`,
	// which will then be ignored.
	// +kubebuilder:validation:Optional
	Digest string `json:"digest,omitempty"`

	// Tag is the name of the agent container image; if `agent.image.digest` is specified, this property is ignored.
	// +kubebuilder:validation:Optional
	Tag string `json:"tag,omitempty"`

	// PullPolicy specifies when to pull the image container.
	// +kubebuilder:validation:Optional
	PullPolicy corev1.PullPolicy `json:"pullPolicy,omitempty"`
}

type ExtendedImageSpec struct {
	// +kubebuilder:validation:Optional
	ImageSpec `json:",inline"`

	// PullSecrets allows you to override the default pull secret that is created when `agent.image.name` starts with
	// "containers.instana.io". Setting `agent.image.pullSecrets` prevents the creation of the default "containers-instana-io" secret.
	// +kubebuilder:validation:Optional
	PullSecrets []corev1.LocalObjectReference `json:"pullSecrets,omitempty"`
}

func (i ImageSpec) Image() string {
	switch {
	case i.Digest != "":
		return fmt.Sprintf("%s@%s", i.Name, i.Digest)
	case i.Tag != "":
		return fmt.Sprintf("%s:%s", i.Name, i.Tag)
	default:
		return i.Name
	}
}

type HostSpec struct {
	// +kubebuilder:validation:Optional
	Repository string `json:"repository,omitempty"`
}

type ServiceMeshSpec struct {
	// +kubebuilder:validation:Optional
	Enabled bool `json:"enabled,omitempty"`
}

type Prometheus struct {
	// +kubebuilder:validation:Optional
	RemoteWrite Enabled `json:"remoteWrite,omitempty"`
}

type BackendSpec struct {
	// +kubebuilder:validation:Required
	EndpointHost string `json:"endpointHost"`
	// +kubebuilder:validation:Required
	EndpointPort string `json:"endpointPort"`
	// +kubebuilder:validation:Optional
	Key string `json:"key"`
}

type ServiceAccountSpec struct {
	// Specifies whether a ServiceAccount should be created.
	Create `json:",inline"`

	// Name of the ServiceAccount. If not set and `create` is true, a name is generated using the fullname template.
	Name `json:",inline"`

	Annotations map[string]string `json:"annotations,omitempty"`
}

type PodSecurityPolicySpec struct {
	// Specifies whether a PodSecurityPolicy should be authorized for the Instana Agent pods.
	// Requires `rbac.create` to be `true` as well.
	Enabled `json:",inline"`
	// The name of an existing PodSecurityPolicy you would like to authorize for the Instana Agent pods.
	// If not set and `enable` is true, a PodSecurityPolicy will be created with a name generated using the fullname template.
	Name `json:",inline"`
}

type KubernetesSpec struct {
	// +kubebuilder:validation:Optional
	DeploymentSpec KubernetesDeploymentSpec `json:"deployment,omitempty"`
}

type K8sSpec struct {
	// +kubebuilder:validation:Optional
	DeploymentSpec KubernetesDeploymentSpec `json:"deployment,omitempty"`
	// +kubebuilder:validation:Optional
	ImageSpec ImageSpec `json:"image,omitempty"`
	// Toggles the PDB for the K8s Sensor
	// +kubebuilder:validation:Optional
	PodDisruptionBudget Enabled `json:"podDisruptionBudget,omitempty"`
}

type KubernetesPodSpec struct {
	ResourceRequirements `json:",inline"`

	// +kubebuilder:validation:Optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// +kubebuilder:validation:Optional
	PriorityClassName string `json:"priorityClassName,omitempty"`

	// agent.pod.tolerations are tolerations to influence agent pod assignment.
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// agent.pod.affinity are affinities to influence agent pod assignment.
	// https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
	// +kubebuilder:validation:Optional
	Affinity corev1.Affinity `json:"affinity,omitempty"`
}

type KubernetesDeploymentSpec struct {
	// Specify if separate deployment of the Kubernetes Sensor should be enabled.
	Enabled `json:",inline"`

	// The minimum number of seconds for which a newly created Pod should be ready without any of its containers crashing, for it to be considered available
	// +kubebuilder:validation:Optional
	MinReadySeconds int `json:"minReadySeconds,omitempty"`

	// Specify the number of replicas for the Kubernetes Sensor.
	// +kubebuilder:validation:Optional
	Replicas int `json:"replicas,omitempty"`

	// Override pod resource requirements for the Kubernetes Sensor pods.
	// +kubebuilder:validation:Optional
	Pod KubernetesPodSpec `json:"pod,omitempty"`
}

type OpenTelemetry struct {
	// Deprecated setting for backwards compatibility. Specify whether Open telemetry is enabled (default is true).
	// +kubebuilder:validation:Optional
	Enabled `json:",inline" yaml:",inline"`

	// Specify whether GRPC is enabled (default is true).
	// +kubebuilder:validation:Optional
	GRPC OpenTelemetryPortConfig `json:"grpc,omitempty" yaml:"grpc,omitempty"`

	// Specify whether HTTP is enabled (default is true).
	// +kubebuilder:validation:Optional
	HTTP OpenTelemetryPortConfig `json:"http,omitempty" yaml:"http,omitempty"`
}

type OpenTelemetryPortConfig struct {
	// Explicitly enable
	// +kubebuilder:validation:Optional
	Enabled *bool `json:"enabled,omitempty" yaml:"enabled,omitempty"`
	// Specify the port
	// +kubebuilder:validation:Optional
	Port *int32 `json:"port,omitempty" yaml:"port,omitempty"`
}

type Zone struct {
	// +kubebuilder:validation:Optional
	Name `json:",inline"`
	// +kubebuilder:validation:Optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`
	// +kubebuilder:validation:Optional
	Affinity corev1.Affinity `json:"affinity,omitempty"`
	// +kubebuilder:validation:Optional
	Mode AgentMode `json:"mode,omitempty"`
}
