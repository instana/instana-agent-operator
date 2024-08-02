/*
(c) Copyright IBM Corp. 2024
*/

package env

import (
	"errors"
	"fmt"
	"strconv"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
)

type EnvVar int

const (
	AgentModeEnv EnvVar = iota
	ZoneNameEnv
	ClusterNameEnv
	AgentEndpointEnv
	AgentEndpointPortEnv
	MavenRepoURLEnv
	MavenRepoFeaturesPath
	MavenRepoSharedPath
	ProxyHostEnv
	ProxyPortEnv
	ProxyProtocolEnv
	ProxyUserEnv
	ProxyPasswordEnv
	ProxyUseDNSEnv
	ListenAddressEnv
	RedactK8sSecretsEnv
	AgentZoneEnv
	HTTPSProxyEnv
	BackendURLEnv
	NoProxyEnv
	ConfigPathEnv
	EntrypointSkipBackendTemplateGeneration
	BackendEnv
	InstanaAgentKeyEnv
	AgentKeyEnv
	DownloadKeyEnv
	InstanaAgentPodNameEnv
	PodNameEnv
	PodIPEnv
	PodUIDEnv
	PodNamespaceEnv
	K8sServiceDomainEnv
	EnableAgentSocketEnv
)

type EnvBuilder interface {
	Build(envVars ...EnvVar) []corev1.EnvVar
}

type envBuilder struct {
	agent   *instanav1.InstanaAgent
	zone    *instanav1.Zone
	helpers helpers.Helpers
}

func NewEnvBuilder(agent *instanav1.InstanaAgent, zone *instanav1.Zone) EnvBuilder {
	return &envBuilder{
		agent:   agent,
		zone:    zone,
		helpers: helpers.NewHelpers(agent),
	}
}

// Build fetches all existing user provided environment variables and bundles them
// together with the environment variables that are defined in the list of EnvVar
// integers.
func (e *envBuilder) Build(envVars ...EnvVar) []corev1.EnvVar {
	allEnvVars := e.getUserProvidedEnvs()
	for _, envVarNumber := range envVars {
		builtEnvVar := e.build(envVarNumber)
		if builtEnvVar != nil {
			allEnvVars = append(allEnvVars, *builtEnvVar)
		}
	}

	return allEnvVars
}

func (e *envBuilder) build(envVar EnvVar) *corev1.EnvVar {
	switch envVar {
	case AgentModeEnv:
		return e.agentModeEnv()
	case ZoneNameEnv:
		return e.zoneNameEnv()
	case ClusterNameEnv:
		return stringToEnvVar("INSTANA_KUBERNETES_CLUSTER_NAME", e.agent.Spec.Cluster.Name)
	case AgentEndpointEnv:
		return stringToEnvVar("INSTANA_AGENT_ENDPOINT", e.agent.Spec.Agent.EndpointHost)
	case AgentEndpointPortEnv:
		return stringToEnvVar("INSTANA_AGENT_ENDPOINT_PORT", e.agent.Spec.Agent.EndpointPort)
	case MavenRepoURLEnv:
		return stringToEnvVar("INSTANA_MVN_REPOSITORY_URL", e.agent.Spec.Agent.MvnRepoUrl)
	case MavenRepoFeaturesPath:
		return stringToEnvVar("INSTANA_MVN_REPOSITORY_FEATURES_PATH", e.agent.Spec.Agent.MvnRepoFeaturesPath)
	case MavenRepoSharedPath:
		return stringToEnvVar("INSTANA_MVN_REPOSITORY_SHARED_PATH", e.agent.Spec.Agent.MvnRepoSharedPath)
	case ProxyHostEnv:
		return stringToEnvVar("INSTANA_AGENT_PROXY_HOST", e.agent.Spec.Agent.ProxyHost)
	case ProxyPortEnv:
		return stringToEnvVar("INSTANA_AGENT_PROXY_PORT", e.agent.Spec.Agent.ProxyPort)
	case ProxyProtocolEnv:
		return stringToEnvVar("INSTANA_AGENT_PROXY_PROTOCOL", e.agent.Spec.Agent.ProxyProtocol)
	case ProxyUserEnv:
		return stringToEnvVar("INSTANA_AGENT_PROXY_USER", e.agent.Spec.Agent.ProxyUser)
	case ProxyPasswordEnv:
		return stringToEnvVar("INSTANA_AGENT_PROXY_PASSWORD", e.agent.Spec.Agent.ProxyPassword)
	case ProxyUseDNSEnv:
		return boolToEnvVar("INSTANA_AGENT_PROXY_USE_DNS", e.agent.Spec.Agent.ProxyUseDNS)
	case ListenAddressEnv:
		return stringToEnvVar("INSTANA_AGENT_HTTP_LISTEN", e.agent.Spec.Agent.ListenAddress)
	case RedactK8sSecretsEnv:
		return stringToEnvVar("INSTANA_KUBERNETES_REDACT_SECRETS", e.agent.Spec.Agent.RedactKubernetesSecrets)
	case AgentZoneEnv:
		return &corev1.EnvVar{Name: "AGENT_ZONE", Value: optional.Of(e.agent.Spec.Cluster.Name).GetOrDefault(e.agent.Spec.Zone.Name)}
	case HTTPSProxyEnv:
		return e.httpsProxyEnv()
	case BackendURLEnv:
		return &corev1.EnvVar{Name: "BACKEND_URL", Value: "https://$(BACKEND)"}
	case NoProxyEnv:
		if e.agent.Spec.Agent.ProxyHost == "" {
			return nil
		}
		return &corev1.EnvVar{Name: "NO_PROXY", Value: "kubernetes.default.svc"}
	case ConfigPathEnv:
		return &corev1.EnvVar{Name: "CONFIG_PATH", Value: volume.InstanaConfigDirectory}
	case EntrypointSkipBackendTemplateGeneration:
		return &corev1.EnvVar{Name: "ENTRYPOINT_SKIP_BACKEND_TEMPLATE_GENERATION", Value: "true"}
	case BackendEnv:
		return e.backendEnv()
	case InstanaAgentKeyEnv:
		return e.agentKeyHelper("INSTANA_AGENT_KEY")
	case AgentKeyEnv:
		return e.agentKeyHelper("AGENT_KEY")
	case DownloadKeyEnv:
		return e.downloadKeyEnv()
	case InstanaAgentPodNameEnv:
		return e.envWithObjectFieldSelector("INSTANA_AGENT_POD_NAME", "metadata.name")
	case PodNameEnv:
		return e.envWithObjectFieldSelector("POD_NAME", "metadata.name")
	case PodIPEnv:
		return e.envWithObjectFieldSelector("POD_IP", "status.podIP")
	case PodUIDEnv:
		return e.envWithObjectFieldSelector("POD_UID", "metadata.uid")
	case PodNamespaceEnv:
		return e.envWithObjectFieldSelector("POD_NAMESPACE", "metadata.namespace")
	case K8sServiceDomainEnv:
		return &corev1.EnvVar{Name: "K8S_SERVICE_DOMAIN", Value: e.helpers.HeadlessServiceName() + "." + e.agent.Namespace + ".svc"}
	case EnableAgentSocketEnv:
		return boolToEnvVar("ENABLE_AGENT_SOCKET", e.agent.Spec.Agent.ServiceMesh.Enabled)
	default:
		panic(errors.New("unknown environment variable requested"))
	}
}

func (e *envBuilder) agentModeEnv() *corev1.EnvVar {
	const envVarName = "INSTANA_AGENT_MODE"
	if e.zone != nil {
		return &corev1.EnvVar{
			Name:  envVarName,
			Value: string(e.zone.Mode),
		}
	}
	return stringToEnvVar(envVarName, string(e.agent.Spec.Agent.Mode))
}

func (e *envBuilder) zoneNameEnv() *corev1.EnvVar {
	const envVarName = "INSTANA_ZONE"
	if e.zone != nil {
		return &corev1.EnvVar{Name: envVarName, Value: e.zone.Name.Name}
	}
	return stringToEnvVar(envVarName, e.agent.Spec.Zone.Name)
}

func (e *envBuilder) httpsProxyEnv() *corev1.EnvVar {
	if e.agent.Spec.Agent.ProxyHost == "" {
		return nil
	}

	return &corev1.EnvVar{
		Name: "HTTPS_PROXY",
		Value: fmt.Sprintf(
			"http://%s%s:%s",
			e.agent.Spec.Agent.ProxyUser+":"+e.agent.Spec.Agent.ProxyPassword+"@",
			e.agent.Spec.Agent.ProxyHost,
			optional.Of(e.agent.Spec.Agent.ProxyPort).GetOrDefault("80"),
		),
	}
}

func (e *envBuilder) backendEnv() *corev1.EnvVar {
	return &corev1.EnvVar{
		Name: "BACKEND",
		ValueFrom: &corev1.EnvVarSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: e.helpers.K8sSensorResourcesName(),
				},
				Key: constants.BackendKey,
			},
		},
	}
}

func (e *envBuilder) agentKeyHelper(name string) *corev1.EnvVar {
	return &corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: optional.Of(e.agent.Spec.Agent.KeysSecret).GetOrDefault(e.agent.Name),
				},
				Key: constants.AgentKey,
			},
		},
	}
}

func (e *envBuilder) downloadKeyEnv() *corev1.EnvVar {
	return &corev1.EnvVar{
		Name: "INSTANA_DOWNLOAD_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: optional.Of(e.agent.Spec.Agent.KeysSecret).GetOrDefault(e.agent.Name),
				},
				Key:      constants.DownloadKey,
				Optional: pointer.To(true),
			},
		},
	}
}

func (e *envBuilder) envWithObjectFieldSelector(name string, fieldPath string) *corev1.EnvVar {
	return &corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

func (e *envBuilder) getUserProvidedEnvs() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	for name, value := range e.agent.Spec.Agent.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
	return envVars
}

func stringToEnvVar(name string, val string) *corev1.EnvVar {
	if val == "" {
		return nil
	}
	return &corev1.EnvVar{Name: name, Value: val}
}

func boolToEnvVar(name string, val bool) *corev1.EnvVar {
	if !val {
		return nil
	}
	return &corev1.EnvVar{Name: name, Value: strconv.FormatBool(val)}
}
