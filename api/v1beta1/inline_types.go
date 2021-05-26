/*
 * (c) Copyright IBM Corp. 2021
 * (c) Copyright Instana Inc. 2021
 */
package v1beta1

import (
	"fmt"
	"strconv"

	appV1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
)

type AgentMode string

const (
	APM            AgentMode = "APM"
	INFRASTRUCTURE AgentMode = "INFRASTRUCTURE"
	AWS            AgentMode = "AWS"
	KUBERNETES     AgentMode = "KUBERNETES"
)

type Name struct {
	Name string `json:"name,omitempty"`
}

type Create struct {
	Create bool `json:"create,omitempty"`
}

type Enabled struct {
	Enabled bool `json:"enabled,omitempty"`
}

// BaseAgentSpec defines the desired state info related to the running Agent
// +k8s:openapi-gen=true
type BaseAgentSpec struct {
	//agent.mode is used to set agent mode and it can be APM, INFRASTRUCTURE or AWS
	Mode AgentMode `json:"mode,omitempty"`

	// agent.key is the secret token which your agent uses to authenticate to Instana's servers.
	// +operator-sdk:csv:customresourcedefinitions:type=spec,displayName="Agent Key",xDescriptors={"urn:alm:descriptor:io.kubernetes:Secret"}
	// +kubebuilder:validation:Required
	Key string `json:"key,omitempty"`

	// agent.downloadKey is key, sometimes known as "sales key", that allows you to download,
	// software from Instana.
	DownloadKey string `json:"downloadKey,omitempty"`

	// Rather than specifying the agent key and optionally the download key, you can "bring your
	// own secret" creating it in the namespace in which you install the `instana-agent` and
	// specify its name in the `keysSecret` field. The secret you create must contains
	// a field called `key` and optionally one called `downloadKey`, which contain, respectively,
	// the values you'd otherwise set in `.agent.key` and `agent.downloadKey`.
	KeysSecret string `json:"keysSecret,omitempty"`

	// agent.listenAddress is the IP address the agent HTTP server will listen to.
	ListenAddress string `json:"listenAddress,omitempty"`

	// agent.endpointHost is the hostname of the Instana server your agents will connect to.
	// +kubebuilder:validation:Required
	EndpointHost string `json:"endpointHost,omitempty"`

	// agent.endpointPort is the port number (as a String) of the Instana server your agents will connect to.
	// +kubebuilder:validation:Required
	EndpointPort string `json:"endpointPort,omitempty"`

	// These are additional backends the Instana agent will report to besides
	// the one configured via the `agent.endpointHost`, `agent.endpointPort` and `agent.key` setting
	AdditionalBackends []BackendSpec `json:"additionalBackends,omitempty"`

	Image *ImageSpec `json:"image,omitempty"`

	UpdateStrategy *appV1.DaemonSetUpdateStrategy `json:"updateStrategy,omitempty"`

	Pod *AgentPodSpec `json:"pod,omitempty"`

	// agent.proxyHost sets the INSTANA_AGENT_PROXY_HOST environment variable.
	ProxyHost string `json:"proxyHost,omitempty"`
	// agent.proxyPort sets the INSTANA_AGENT_PROXY_PORT environment variable.
	ProxyPort string `json:"proxyPort,omitempty"`
	// agent.proxyProtocol sets the INSTANA_AGENT_PROXY_PROTOCOL environment variable.
	ProxyProtocol string `json:"proxyProtocol,omitempty"`
	// agent.proxyUser sets the INSTANA_AGENT_PROXY_USER environment variable.
	ProxyUser string `json:"proxyUser,omitempty"`
	// agent.proxyPassword sets the INSTANA_AGENT_PROXY_PASSWORD environment variable.
	ProxyPassword string `json:"proxyPassword,omitempty"`
	// agent.proxyUseDNS sets the INSTANA_AGENT_PROXY_USE_DNS environment variable.
	ProxyUseDNS bool `json:"proxyUseDNS,omitempty"`

	// use this to set additional environment variables for the instana agent
	// for example:
	//  env:
	//   INSTANA_AGENT_TAGS: dev
	Env map[string]string `json:"env,omitEmpty"`

	Configuration      *ConfigurationSpec `json:"configuration,omitempty"`
	Configuration_yaml string             `json:"configuration_yaml,omitempty"`

	// agent.redactKubernetesSecrets sets the INSTANA_KUBERNETES_REDACT_SECRETS environment variable.
	RedactKubernetesSecrets string `json:"redactKubernetesSecrets,omitempty"`

	// agent.host.repository sets a host path to be mounted as the agent maven repository (for debugging or development purposes)
	Host *HostSpec `json:"host,omitempty"`
}

func (spec *BaseAgentSpec) AdditionalBackendsValues() []string {
	values := []string{}
	if len(spec.AdditionalBackends) != 0 {
		for index, backend := range spec.AdditionalBackends {
			if len(backend.EndpointHost) != 0 {
				values = append(values, fmt.Sprintf("agent.additionalBackends[%d].endpointHost=%s", index, backend.EndpointHost))
			}
			if len(backend.EndpointPort) != 0 {
				values = append(values, fmt.Sprintf("agent.additionalBackends[%d].endpointPort=%s", index, backend.EndpointPort))
			}
			if len(backend.Key) != 0 {
				values = append(values, fmt.Sprintf("agent.additionalBackends[%d].key=%s", index, backend.Key))
			}
		}
	}

	return values
}

type AgentPodSpec struct {
	// agent.pod.annotations are additional annotations to be added to the agent pods.
	Annotations map[string]string `json:"annotations,omitempty"`

	// agent.pod.tolerations are tolerations to influence agent pod assignment.
	Tollerations []coreV1.Toleration `json:"tollerations,omitempty"`

	// agent.pod.affinity are affinities to influence agent pod assignment.
	// https://kubernetes.io/docs/concepts/configuration/taint-and-toleration/
	Affinity *coreV1.Affinity `json:"affinity,omitempty"`

	// agent.pod.priorityClassName is the name of an existing PriorityClass that should be set on the agent pods
	// https://kubernetes.io/docs/concepts/configuration/pod-priority-preemption/
	PriorityClassName string `json:"priorityClassName,omitempty"`

	coreV1.ResourceRequirements `json:",inline"`
}

type ImageSpec struct {
	// agent.image.name is the name of the container image of the Instana agent.
	Name string `json:"name,omitempty"`
	// agent.image.digest is the digest (a.k.a. Image ID) of the agent container image; if specified, it has priority over agent.image.tag, which will be ignored.
	Digest string `json:"digest,omitempty"`
	// agent.image.tag is the tag name of the agent container image; if agent.image.digest is specified, this property is ignored.
	Tag string `json:"tag,omitempty"`
	// agent.image.pullPolicy specifies when to pull the image container.
	PullPolicy string `json:"pullPolicy,omitempty"`
	// agent.image.pullSecrets allows you to override the default pull secret that is created when agent.image.name starts with "containers.instana.io"
	// Setting agent.image.pullSecrets prevents the creation of the default "containers-instana-io" secret.
	PullSecrets []PullSecretSpec `json:"pullSecrets,omitempty"`
}

func (spec *ImageSpec) Values() []string {
	values := []string{}
	if len(spec.Name) != 0 {
		values = append(values, "agent.image.name="+spec.Name)
	}
	if len(spec.Digest) != 0 {
		values = append(values, "agent.image.digest="+spec.Digest)
	}
	if len(spec.Tag) != 0 {
		values = append(values, "agent.image.tag="+spec.Tag)
	}
	if len(spec.PullPolicy) != 0 {
		values = append(values, "agent.image.pullPolicy="+spec.PullPolicy)
	}
	if len(spec.PullSecrets) != 0 {
		for index, secret := range spec.PullSecrets {

			values = append(values, fmt.Sprintf("agent.image.pullSecrets[%d].name=%s", index, secret.Name.Name))
		}
	}

	return values
}

type PullSecretSpec struct {
	Name `json:",inline"`
}

type HostSpec struct {
	Repository string `json:"repository,omitempty"`
}

type ConfigurationSpec struct {
	// When setting this to true, the Helm chart will automatically look up the entries
	// of the default instana-agent ConfigMap, and mount as agent configuration files
	// under /opt/instana/agent/etc/instana all entries with keys that match the
	// 'configuration-*.yaml' scheme
	AutoMountConfigEntries bool `json:"autoMountConfigEntries,omitempty"`
}

type Prometheus struct {
	RemoteWrite *Enabled `json:"remoteWrite,omitempty"`
}

type BackendSpec struct {
	EndpointHost string `json:"endpointHost,omitempty"`
	EndpointPort string `json:"endpointPort,omitempty"`
	Key          string `json:"key,omitempty"`
}

func (backendSpec *BackendSpec) toString() {

}

type PodSecurityPolicySpec struct {
	// Specifies whether a PodSecurityPolicy should be authorized for the Instana Agent pods.
	// Requires `rbac.create` to be `true` as well.
	Enabled `json:",inline"`
	// The name of an existing PodSecurityPolicy you would like to authorize for the Instana Agent pods.
	// If not set and `enable` is true, a PodSecurityPolicy will be created with a name generated using the fullname template.
	Name `json:",inline"`
}

type K8sSpec struct {
	Deployment *K8sDeploymentSpec `json:"deployment,omitempty"`
}

type K8sDeploymentSpec struct {
	Enabled  `json:",inline"`
	Replicas int                          `json:"replicas,omitempty"`
	Pod      *coreV1.ResourceRequirements `json:"pod,omitempty"`
}

func (spec *K8sSpec) Values() []string {
	values := []string{}
	if spec.Deployment != nil {
		values = append(values, "kubernetes.deployment.enabled="+strconv.FormatBool(spec.Deployment.Enabled.Enabled))
		values = append(values, "kubernetes.deployment.replicas="+strconv.Itoa(spec.Deployment.Replicas))
		if spec.Deployment.Pod != nil {
			if spec.Deployment.Pod.Requests != nil {
				requestList := &spec.Deployment.Pod.Requests
				if val, ok := (*requestList)["memory"]; ok {
					values = append(values, "kubernetes.deployment.pod.requests.memory="+val.String())
				}
				if val, ok := (*requestList)["cpu"]; ok {
					values = append(values, "kubernetes.deployment.pod.requests.cpu="+val.String())
				}
			}
			if spec.Deployment.Pod.Limits != nil {
				limitsList := &spec.Deployment.Pod.Limits
				if val, ok := (*limitsList)["memory"]; ok {
					values = append(values, "kubernetes.deployment.pod.limits.memory="+val.String())
				}
				if val, ok := (*limitsList)["cpu"]; ok {
					values = append(values, "kubernetes.deployment.pod.limits.cpu="+val.String())
				}
			}
		}
	}
	return values
}
