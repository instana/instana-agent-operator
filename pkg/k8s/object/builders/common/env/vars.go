package env

import (
	"fmt"
	"strings"

	corev1 "k8s.io/api/core/v1"

	_map "github.com/instana/instana-agent-operator/pkg/collections/map"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

// Directly From CR

func (e *envBuilder) agentModeEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_MODE", e.agent.Spec.Agent.Mode)
}

func (e *envBuilder) zoneNameEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_ZONE", e.agent.Spec.Zone.Name)
}

func (e *envBuilder) clusterNameEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_KUBERNETES_CLUSTER_NAME", e.agent.Spec.Cluster.Name)
}

func (e *envBuilder) agentEndpointEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_ENDPOINT", e.agent.Spec.Agent.EndpointHost)
}

func (e *envBuilder) agentEndpointPortEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_ENDPOINT_PORT", e.agent.Spec.Agent.EndpointPort)
}

func (e *envBuilder) mavenRepoURLEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_MVN_REPOSITORY_URL", e.agent.Spec.Agent.MvnRepoUrl)
}

// TODO: Two new ones added here recently (INSTANA_MVN_REPOSITORY_FEATURES_PATH and INSTANA_MVN_REPOSITORY_SHARED_PATH)

func (e *envBuilder) proxyHostEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_PROXY_HOST", e.agent.Spec.Agent.ProxyHost)
}

func (e *envBuilder) proxyPortEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_PROXY_PORT", e.agent.Spec.Agent.ProxyPort)
}

func (e *envBuilder) proxyProtocolEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_PROXY_PROTOCOL", e.agent.Spec.Agent.ProxyProtocol)
}

func (e *envBuilder) proxyUserEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_PROXY_USER", e.agent.Spec.Agent.ProxyUser)
}

func (e *envBuilder) proxyPasswordEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_PROXY_PASSWORD", e.agent.Spec.Agent.ProxyPassword)
}

func (e *envBuilder) proxyUseDNSEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_PROXY_USE_DNS", e.agent.Spec.Agent.ProxyUseDNS)
}

func (e *envBuilder) listenAddressEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_AGENT_HTTP_LISTEN", e.agent.Spec.Agent.ListenAddress)
}

func (e *envBuilder) redactK8sSecretsEnv() optional.Optional[corev1.EnvVar] {
	return fromCRField("INSTANA_KUBERNETES_REDACT_SECRETS", e.agent.Spec.Agent.RedactKubernetesSecrets)
}

func (e *envBuilder) agentZoneEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name:  "AGENT_ZONE",
			Value: optional.Of(e.agent.Spec.Cluster.Name).GetOrDefault(e.agent.Spec.Zone.Name),
		},
	)
}

func (e *envBuilder) proxyUserPass() string {
	return optional.Map[string, string](
		optional.Of(e.agent.Spec.Agent.ProxyUser),
		func(proxyUser string) string {
			return optional.Map[string, string](
				optional.Of(e.agent.Spec.Agent.ProxyPassword),
				func(proxyPassword string) string {
					return fmt.Sprintf("%s:%s@", proxyUser, proxyPassword)
				},
			).Get()
		},
	).Get()
}

func (e *envBuilder) httpsProxyEnv() optional.Optional[corev1.EnvVar] {
	return optional.Map[string, corev1.EnvVar](
		optional.Of(e.agent.Spec.Agent.ProxyHost),
		func(proxyHost string) corev1.EnvVar {
			return corev1.EnvVar{
				Name: "HTTPS_PROXY",
				Value: fmt.Sprintf(
					"http://%s%s:%s",
					e.proxyUserPass(),
					proxyHost,
					optional.Of(e.agent.Spec.Agent.ProxyPort).GetOrDefault("80"),
				),
			}
		},
	)
}

// Static

func (e *envBuilder) backendURLEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name:  "BACKEND_URL",
			Value: "https://$(BACKEND)",
		},
	)
}

func (e *envBuilder) noProxyEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name:  "NO_PROXY",
			Value: "kubernetes.default.svc",
		},
	)
}

// From a ConfigMap

func (e *envBuilder) backendEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "BACKEND",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: e.K8sSensorResourcesName(),
					},
					Key: constants.BackendKey,
				},
			},
		},
	)
}

// From a Secret

func (e *envBuilder) agentKeyEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "INSTANA_AGENT_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: e.KeysSecretName(),
					},
					Key: constants.AgentKey,
				},
			},
		},
	)
}

func (e *envBuilder) downloadKeyEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "INSTANA_DOWNLOAD_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: e.KeysSecretName(),
					},
					Key:      constants.DownloadKey,
					Optional: pointer.To(true),
				},
			},
		},
	)
}

// From Pod Reference

func (e *envBuilder) podNameEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "INSTANA_AGENT_POD_NAME",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	)
}

func (e *envBuilder) podIPEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "POD_IP",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "status.podIP",
				},
			},
		},
	)
}

func (e *envBuilder) podUIDEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "POD_UID",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.uid",
				},
			},
		},
	)
}

func (e *envBuilder) podNamespaceEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name: "POD_NAMESPACE",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.namespace",
				},
			},
		},
	)
}

// Referencing Another Object Created by the Operator

func (e *envBuilder) k8sServiceDomainEnv() optional.Optional[corev1.EnvVar] {
	return optional.Of(
		corev1.EnvVar{
			Name:  "K8S_SERVICE_DOMAIN",
			Value: strings.Join([]string{e.HeadlessServiceName(), e.agent.Namespace, "svc"}, "."),
		},
	)
}

// From user-provided in CR

func (e *envBuilder) userProvidedEnv() []optional.Optional[corev1.EnvVar] {
	return _map.NewMapConverter[string, string, optional.Optional[corev1.EnvVar]]().
		ToList(
			e.agent.Spec.Agent.Env, func(name string, value string) optional.Optional[corev1.EnvVar] {
				return optional.Of(
					corev1.EnvVar{
						Name:  name,
						Value: value,
					},
				)
			},
		)
}
