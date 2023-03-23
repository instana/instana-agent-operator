package env

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	_map "github.com/instana/instana-agent-operator/pkg/collections/map"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
)

// Directly From CR

func AgentModeEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_MODE", agent.Spec.Agent.Mode)
}

// TODO: Leader elector port still needed?

func ZoneNameEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_ZONE", agent.Spec.Zone.Name)
}

func ClusterNameEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_KUBERNETES_CLUSTER_NAME", agent.Spec.Cluster.Name)
}

func AgentEndpointEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_ENDPOINT", agent.Spec.Agent.EndpointHost)
}

func AgentEndpointPortEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_ENDPOINT_PORT", agent.Spec.Agent.EndpointPort)
}

func MavenRepoUrlEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_MVN_REPOSITORY_URL", agent.Spec.Agent.MvnRepoUrl)
}

func ProxyHostEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_PROXY_HOST", agent.Spec.Agent.ProxyHost)
}

func ProxyPortEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_PROXY_PORT", agent.Spec.Agent.ProxyPort)
}

func ProxyProtocolEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_PROXY_PROTOCOL", agent.Spec.Agent.ProxyProtocol)
}

func ProxyUserEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_PROXY_USER", agent.Spec.Agent.ProxyUser)
}

func ProxyPasswordEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_PROXY_PASSWORD", agent.Spec.Agent.ProxyPassword)
}

func ProxyUseDNSEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_PROXY_USE_DNS", agent.Spec.Agent.ProxyUseDNS)
}

func ListenAddressEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_AGENT_HTTP_LISTEN", agent.Spec.Agent.ListenAddress)
}

func RedactK8sSecretsEnv(agent *instanav1.InstanaAgent) EnvBuilder {
	return fromCRField("INSTANA_KUBERNETES_REDACT_SECRETS", agent.Spec.Agent.RedactKubernetesSecrets)
}

// From a Secret

func AgentKeyEnv(helpers helpers.Helpers) EnvBuilder {
	return fromLiteralVal(corev1.EnvVar{
		Name: "INSTANA_AGENT_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: helpers.KeysSecretName(),
				},
				Key: "key",
			},
		},
	})
}

func DownloadKeyEnv(helpers helpers.Helpers) EnvBuilder {
	return fromLiteralVal(corev1.EnvVar{
		Name: "INSTANA_DOWNLOAD_KEY",
		ValueFrom: &corev1.EnvVarSource{
			SecretKeyRef: &corev1.SecretKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: helpers.KeysSecretName(),
				},
				Key:      "downloadKey",
				Optional: pointer.To(true),
			},
		},
	})
}

// From Pod Reference

func PodNameEnv() EnvBuilder {
	return fromLiteralVal(corev1.EnvVar{
		Name: "INSTANA_AGENT_POD_NAME",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "metadata.name",
			},
		},
	})
}

func PodIpEnv() EnvBuilder {
	return fromLiteralVal(corev1.EnvVar{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	})
}

// From user-provided in CR

func UserProvidedEnv(agent *instanav1.InstanaAgent) []EnvBuilder {
	return _map.NewMapConverter[string, string, EnvBuilder]().
		ToList(agent.Spec.Agent.Env, func(name string, value string) EnvBuilder {
			return fromLiteralVal(
				corev1.EnvVar{
					Name:  name,
					Value: value,
				},
			)
		})
}