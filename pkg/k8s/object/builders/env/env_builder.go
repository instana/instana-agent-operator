package env

import (
	"errors"

	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Test

type EnvVar int

const (
	AgentModeEnv EnvVar = iota
	ZoneNameEnv
	ClusterNameEnv
	AgentEndpointEnv
	AgentEndpointPortEnv
	MavenRepoURLEnv
	ProxyHostEnv
	ProxyPortEnv
	ProxyProtocolEnv
	ProxyUserEnv
	ProxyPasswordEnv
	ProxyUseDNSEnv
	ListenAddressEnv
	RedactK8sSecretsEnv
	AgentKeyEnv
	DownloadKeyEnv
	PodNameEnv
	PodIPEnv
	K8sServiceDomainEnv
)

type EnvBuilder interface {
	Build(envVars ...EnvVar) []corev1.EnvVar
}

type envBuilder struct {
	agent *instanav1.InstanaAgent
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
	case AgentKeyEnv:
		return e.agentKeyEnv
	case DownloadKeyEnv:
		return e.downloadKeyEnv
	case PodNameEnv:
		return e.podNameEnv
	case PodIPEnv:
		return e.podIPEnv
	case K8sServiceDomainEnv:
		return e.k8sServiceDomainEnv
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
	return &envBuilder{
		agent:   agent,
		Helpers: helpers.NewHelpers(agent),
	}
}
