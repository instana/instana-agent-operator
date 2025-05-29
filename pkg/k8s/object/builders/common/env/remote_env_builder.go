/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package env

import (
	"errors"
	"fmt"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
)

type EnvVarRemote int

const (
	AgentModeEnvRemote EnvVarRemote = iota
	ZoneNameEnvRemote
	ClusterNameEnvRemote
	AgentEndpointEnvRemote
	AgentEndpointPortEnvRemote
	MavenRepoURLEnvRemote
	MavenRepoFeaturesPathRemote
	MavenRepoSharedPathRemote
	MirrorReleaseRepoUrlEnvRemote
	MirrorReleaseRepoUsernameEnvRemote
	MirrorReleaseRepoPasswordEnvRemote
	MirrorSharedRepoUrlEnvRemote
	MirrorSharedRepoUsernameEnvRemote
	MirrorSharedRepoPasswordEnvRemote
	ProxyHostEnvRemote
	ProxyPortEnvRemote
	ProxyProtocolEnvRemote
	ProxyUserEnvRemote
	ProxyPasswordEnvRemote
	ProxyUseDNSEnvRemote
	ListenAddressEnvRemote
	RedactK8sSecretsEnvRemote
	AgentZoneEnvRemote
	HTTPSProxyEnvRemote
	BackendURLEnvRemote
	NoProxyEnvRemote
	ConfigPathEnvRemote
	EntrypointSkipBackendTemplateGenerationRemote
	BackendEnvRemote
	InstanaAgentKeyEnvRemote
	AgentKeyEnvRemote
	DownloadKeyEnvRemote
	InstanaAgentPodNameEnvRemote
	PodNameEnvRemote
	PodIPEnvRemote
	PodUIDEnvRemote
	PodNamespaceEnvRemote
	K8sServiceDomainEnvRemote
	EnableAgentSocketEnvRemote
)

type EnvBuilderRemote interface {
	Build(envVarsRemote ...EnvVarRemote) []corev1.EnvVar
}

type envBuilderRemote struct {
	agent   *instanav1.RemoteAgent
	zone    *instanav1.Zone
	helpers helpers.RemoteHelpers
}

func NewEnvBuilderRemote(agent *instanav1.RemoteAgent, zone *instanav1.Zone) EnvBuilderRemote {
	return &envBuilderRemote{
		agent:   agent,
		zone:    zone,
		helpers: helpers.NewRemoteHelpers(agent),
	}
}

// Build fetches all existing user provided environment variables and bundles them
// together with the environment variables that are defined in the list of EnvVar
// integers.
func (e *envBuilderRemote) Build(envVars ...EnvVarRemote) []corev1.EnvVar {
	allEnvVars := e.getUserProvidedEnvs()
	for _, envVarNumber := range envVars {
		builtEnvVar := e.buildRemote(envVarNumber)
		if builtEnvVar != nil {
			allEnvVars = append(allEnvVars, *builtEnvVar)
		}
	}

	return allEnvVars
}

func (e *envBuilderRemote) buildRemote(envVar EnvVarRemote) *corev1.EnvVar {
	switch envVar {
	case AgentModeEnvRemote:
		return e.agentModeEnv()
	case ZoneNameEnvRemote:
		return e.zoneNameEnv()
	case ClusterNameEnvRemote:
		return stringToEnvVar("INSTANA_KUBERNETES_CLUSTER_NAME", e.agent.Spec.Cluster.Name)
	case AgentEndpointEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_ENDPOINT", e.agent.Spec.Agent.EndpointHost)
	case AgentEndpointPortEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_ENDPOINT_PORT", e.agent.Spec.Agent.EndpointPort)
	case MavenRepoURLEnvRemote:
		return stringToEnvVar("INSTANA_MVN_REPOSITORY_URL", e.agent.Spec.Agent.MvnRepoUrl)
	case MavenRepoFeaturesPathRemote:
		return stringToEnvVar("INSTANA_MVN_REPOSITORY_FEATURES_PATH", e.agent.Spec.Agent.MvnRepoFeaturesPath)
	case MavenRepoSharedPathRemote:
		return stringToEnvVar("INSTANA_MVN_REPOSITORY_SHARED_PATH", e.agent.Spec.Agent.MvnRepoSharedPath)
	case MirrorReleaseRepoUrlEnvRemote:
		return stringToEnvVar("AGENT_RELEASE_REPOSITORY_MIRROR_URL", e.agent.Spec.Agent.MirrorReleaseRepoUrl)
	case MirrorReleaseRepoUsernameEnvRemote:
		return stringToEnvVar("AGENT_RELEASE_REPOSITORY_MIRROR_USERNAME", e.agent.Spec.Agent.MirrorReleaseRepoUsername)
	case MirrorSharedRepoUrlEnvRemote:
		return stringToEnvVar("INSTANA_SHARED_REPOSITORY_MIRROR_URL", e.agent.Spec.Agent.MirrorSharedRepoUrl)
	case MirrorSharedRepoUsernameEnvRemote:
		return stringToEnvVar("INSTANA_SHARED_REPOSITORY_MIRROR_USERNAME", e.agent.Spec.Agent.MirrorSharedRepoUsername)
	case MirrorSharedRepoPasswordEnvRemote:
		return stringToEnvVar("INSTANA_SHARED_REPOSITORY_MIRROR_PASSWORD", e.agent.Spec.Agent.MirrorSharedRepoPassword)
	case MirrorReleaseRepoPasswordEnvRemote:
		return stringToEnvVar("AGENT_RELEASE_REPOSITORY_MIRROR_PASSWORD", e.agent.Spec.Agent.MirrorReleaseRepoPassword)
	case ProxyHostEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_PROXY_HOST", e.agent.Spec.Agent.ProxyHost)
	case ProxyPortEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_PROXY_PORT", e.agent.Spec.Agent.ProxyPort)
	case ProxyProtocolEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_PROXY_PROTOCOL", e.agent.Spec.Agent.ProxyProtocol)
	case ProxyUserEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_PROXY_USER", e.agent.Spec.Agent.ProxyUser)
	case ProxyPasswordEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_PROXY_PASSWORD", e.agent.Spec.Agent.ProxyPassword)
	case ProxyUseDNSEnvRemote:
		return boolToEnvVar("INSTANA_AGENT_PROXY_USE_DNS", e.agent.Spec.Agent.ProxyUseDNS)
	case ListenAddressEnvRemote:
		return stringToEnvVar("INSTANA_AGENT_HTTP_LISTEN", e.agent.Spec.Agent.ListenAddress)
	case RedactK8sSecretsEnvRemote:
		return stringToEnvVar("INSTANA_KUBERNETES_REDACT_SECRETS", e.agent.Spec.Agent.RedactKubernetesSecrets)
	case AgentZoneEnvRemote:
		return &corev1.EnvVar{Name: "AGENT_ZONE", Value: optional.Of(e.agent.Spec.Cluster.Name).GetOrDefault(e.agent.Spec.Zone.Name)}
	case HTTPSProxyEnvRemote:
		return e.httpsProxyEnv()
	case BackendURLEnvRemote:
		return &corev1.EnvVar{Name: "BACKEND_URL", Value: "https://$(BACKEND)"}
	case NoProxyEnvRemote:
		if e.agent.Spec.Agent.ProxyHost == "" {
			return nil
		}
		return &corev1.EnvVar{Name: "NO_PROXY", Value: "kubernetes.default.svc"}
	case ConfigPathEnvRemote:
		return &corev1.EnvVar{Name: "CONFIG_PATH", Value: volume.RemoteConfigDirectory}
	case EntrypointSkipBackendTemplateGenerationRemote:
		return &corev1.EnvVar{Name: "ENTRYPOINT_SKIP_BACKEND_TEMPLATE_GENERATION", Value: "true"}
	case BackendEnvRemote:
		return e.backendEnv()
	case InstanaAgentKeyEnvRemote:
		return e.agentKeyHelper("INSTANA_AGENT_KEY")
	case AgentKeyEnvRemote:
		return e.agentKeyHelper("AGENT_KEY")
	case DownloadKeyEnvRemote:
		return e.downloadKeyEnv()
	case InstanaAgentPodNameEnvRemote:
		return e.envWithObjectFieldSelector("INSTANA_AGENT_POD_NAME", "metadata.name")
	case PodNameEnvRemote:
		return e.envWithObjectFieldSelector("POD_NAME", "metadata.name")
	case PodIPEnvRemote:
		return e.envWithObjectFieldSelector("POD_IP", "status.podIP")
	case PodUIDEnvRemote:
		return e.envWithObjectFieldSelector("POD_UID", "metadata.uid")
	case PodNamespaceEnvRemote:
		return e.envWithObjectFieldSelector("POD_NAMESPACE", "metadata.namespace")
	case K8sServiceDomainEnvRemote:
		return &corev1.EnvVar{Name: "K8S_SERVICE_DOMAIN", Value: e.helpers.HeadlessServiceName() + "." + e.agent.Namespace + ".svc"}
	case EnableAgentSocketEnvRemote:
		return boolToEnvVar("ENABLE_AGENT_SOCKET", e.agent.Spec.Agent.ServiceMesh.Enabled)
	default:
		panic(errors.New("unknown environment variable requested"))
	}
}

func (e *envBuilderRemote) agentModeEnv() *corev1.EnvVar {
	const envVarName = "INSTANA_AGENT_MODE"
	if e.zone != nil {
		return &corev1.EnvVar{
			Name:  envVarName,
			Value: string(e.zone.Mode),
		}
	}
	return stringToEnvVar(envVarName, string(e.agent.Spec.Agent.Mode))
}

func (e *envBuilderRemote) zoneNameEnv() *corev1.EnvVar {
	const envVarName = "INSTANA_ZONE"
	if e.zone != nil {
		return &corev1.EnvVar{Name: envVarName, Value: e.zone.Name.Name}
	}
	return stringToEnvVar(envVarName, e.agent.Spec.Zone.Name)
}

func (e *envBuilderRemote) httpsProxyEnv() *corev1.EnvVar {
	if e.agent.Spec.Agent.ProxyHost == "" {
		return nil
	}

	if e.agent.Spec.Agent.ProxyUser == "" || e.agent.Spec.Agent.ProxyPassword == "" {
		return &corev1.EnvVar{
			Name: "HTTPS_PROXY",
			Value: fmt.Sprintf(
				"%s://%s:%s",
				optional.Of(e.agent.Spec.Agent.ProxyProtocol).GetOrDefault("http"),
				e.agent.Spec.Agent.ProxyHost,
				optional.Of(e.agent.Spec.Agent.ProxyPort).GetOrDefault("80"),
			),
		}
	}

	return &corev1.EnvVar{
		Name: "HTTPS_PROXY",
		Value: fmt.Sprintf(
			"%s://%s%s:%s",
			optional.Of(e.agent.Spec.Agent.ProxyProtocol).GetOrDefault("http"),
			e.agent.Spec.Agent.ProxyUser+":"+e.agent.Spec.Agent.ProxyPassword+"@",
			e.agent.Spec.Agent.ProxyHost,
			optional.Of(e.agent.Spec.Agent.ProxyPort).GetOrDefault("80"),
		),
	}
}

func (e *envBuilderRemote) backendEnv() *corev1.EnvVar {
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

func (e *envBuilderRemote) agentKeyHelper(name string) *corev1.EnvVar {
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

func (e *envBuilderRemote) downloadKeyEnv() *corev1.EnvVar {
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

func (e *envBuilderRemote) envWithObjectFieldSelector(name string, fieldPath string) *corev1.EnvVar {
	return &corev1.EnvVar{
		Name: name,
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: fieldPath,
			},
		},
	}
}

func (e *envBuilderRemote) getUserProvidedEnvs() []corev1.EnvVar {
	envVars := []corev1.EnvVar{}
	for name, value := range e.agent.Spec.Agent.Env {
		envVars = append(envVars, corev1.EnvVar{
			Name:  name,
			Value: value,
		})
	}
	return envVars
}
