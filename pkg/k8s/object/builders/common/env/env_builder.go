// /*
// (c) Copyright IBM Corp. 2024
// (c) Copyright Instana Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
// */
//

package env

import (
	"errors"

	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
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
	agent *instanav1.InstanaAgent
	zone  *instanav1.Zone
	helpers.Helpers
}

// Mapping between EnvVar constants and the functions that build them must be included here
func (e *envBuilder) getBuilder(envVar EnvVar) func() optional.Optional[corev1.EnvVar] {
	switch envVar {
	case AgentModeEnv:
		return e.agentModeEnv
	case ZoneNameEnv:
		return e.zoneNameEnv
	case ClusterNameEnv:
		return e.clusterNameEnv
	case AgentEndpointEnv:
		return e.agentEndpointEnv
	case AgentEndpointPortEnv:
		return e.agentEndpointPortEnv
	case MavenRepoURLEnv:
		return e.mavenRepoURLEnv
	case MavenRepoFeaturesPath:
		return e.mavenRepoFeaturesPath
	case MavenRepoSharedPath:
		return e.mavenRepoSharedPath
	case ProxyHostEnv:
		return e.proxyHostEnv
	case ProxyPortEnv:
		return e.proxyPortEnv
	case ProxyProtocolEnv:
		return e.proxyProtocolEnv
	case ProxyUserEnv:
		return e.proxyUserEnv
	case ProxyPasswordEnv:
		return e.proxyPasswordEnv
	case ProxyUseDNSEnv:
		return e.proxyUseDNSEnv
	case ListenAddressEnv:
		return e.listenAddressEnv
	case RedactK8sSecretsEnv:
		return e.redactK8sSecretsEnv
	case AgentZoneEnv:
		return e.agentZoneEnv
	case HTTPSProxyEnv:
		return e.httpsProxyEnv
	case BackendURLEnv:
		return e.backendURLEnv
	case NoProxyEnv:
		return e.noProxyEnv
	case ConfigPathEnv:
		return e.configPathEnv
	case BackendEnv:
		return e.backendEnv
	case InstanaAgentKeyEnv:
		return e.instanaAgentKeyEnv
	case AgentKeyEnv:
		return e.agentKeyEnv
	case DownloadKeyEnv:
		return e.downloadKeyEnv
	case InstanaAgentPodNameEnv:
		return e.instanaAgentPodNameEnv
	case PodNameEnv:
		return e.podNameEnv
	case PodIPEnv:
		return e.podIPEnv
	case PodUIDEnv:
		return e.podUIDEnv
	case PodNamespaceEnv:
		return e.podNamespaceEnv
	case K8sServiceDomainEnv:
		return e.k8sServiceDomainEnv
	case EnableAgentSocketEnv:
		return e.enableAgentSocketEnv
	default:
		panic(errors.New("unknown environment variable requested"))
	}
}

func (e *envBuilder) Build(envVars ...EnvVar) []corev1.EnvVar {
	userProvided := e.userProvidedEnv()

	builtFromSpec := list.NewListMapTo[EnvVar, optional.Optional[corev1.EnvVar]]().MapTo(
		envVars,
		func(envVar EnvVar) optional.Optional[corev1.EnvVar] {
			return e.getBuilder(envVar)()
		},
	)

	return optional.NewNonEmptyOptionalMapper[corev1.EnvVar]().AllNonEmpty(append(userProvided, builtFromSpec...))
}

func NewEnvBuilder(agent *instanav1.InstanaAgent) EnvBuilder {
	return NewEnvBuilderWithZoneInfo(agent, nil)
}

func NewEnvBuilderWithZoneInfo(agent *instanav1.InstanaAgent, zone *instanav1.Zone) EnvBuilder {
	return &envBuilder{
		agent:   agent,
		zone:    zone,
		Helpers: helpers.NewHelpers(agent),
	}
}
