/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc. 2024
*/

package instana_agent_config

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/or_die"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

type secretBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager
	keysSecret    *corev1.Secret
}

func NewSecretBuilder(agent *instanav1.InstanaAgent, statusManager status.AgentStatusManager, keysSecret *corev1.Secret) builder.ObjectBuilder {
	return &secretBuilder{
		InstanaAgent:  agent,
		statusManager: statusManager,
		keysSecret:    keysSecret,
	}
}

func (c *secretBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (c *secretBuilder) IsNamespaced() bool {
	return true
}

func (c *secretBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(cm client.Object) {
				c.statusManager.SetAgentConfigSecret(client.ObjectKeyFromObject(cm))
			},
		)
	}()

	return optional.Of[client.Object](
		&corev1.Secret{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Secret",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Name + "-config",
				Namespace: c.Namespace,
			},
			Data: c.getData(),
			Type: corev1.SecretTypeOpaque,
		},
	)
}

func (c *secretBuilder) getData() map[string][]byte {
	res := make(map[string][]byte, 2)

	optional.Of(c.Spec.Cluster.Name).IfPresent(
		func(clusterName string) {
			res["cluster_name"] = []byte(clusterName)
		},
	)

	optional.Of(c.Spec.Agent.ConfigurationYaml).IfPresent(
		func(configYaml string) {
			res["configuration.yaml"] = []byte(configYaml)
		},
	)

	if otlp := c.Spec.OpenTelemetry; otlp.IsEnabled() {
		otlpPluginSettings := map[string]instanav1.OpenTelemetry{"com.instana.plugin.opentelemetry": otlp}
		res["configuration-opentelemetry.yaml"] = []byte(yamlOrDie(&otlpPluginSettings))
	}

	if pointer.DerefOrEmpty(c.Spec.Prometheus.RemoteWrite.Enabled) {
		res["configuration-prometheus-remote-write.yaml"] = []byte(
			yamlOrDie(
				map[string]any{
					"com.instana.plugin.prometheus": map[string]any{
						"remote_write": map[string]any{
							"enabled": true,
						},
					},
				},
			),
		)

	}

	// Deprecated since k8s sensor deployment will always be enabled now,
	// can remove once deprecated sensor is removed from agent

	res["configuration-disable-kubernetes-sensor.yaml"] = []byte(
		yamlOrDie(
			map[string]any{
				"com.instana.plugin.kubernetes": map[string]any{
					"enabled": false,
				},
			},
		),
	)

	backendLines := make([]string, 0, 10)
	backendLines = append(
		backendLines,
		keyEqualsValue("host", c.Spec.Agent.EndpointHost),
		keyEqualsValue("port", optional.Of(c.Spec.Agent.EndpointPort).GetOrDefault("443")),
		keyEqualsValue("protocol", "HTTP/2"),
	)

	if c.Spec.Agent.Key == "" {
		// Agent key was retrieved from external secret
		agentKey := c.keysSecret.Data["key"]
		backendLines = append(
			backendLines,
			keyEqualsValue("key", string(agentKey)),
		)
	} else {
		// Agent key was directly defined on the CRD
		backendLines = append(
			backendLines,
			keyEqualsValue("key", c.Spec.Agent.Key),
		)
	}

	optional.Of(c.Spec.Agent.ProxyHost).IfPresent(
		func(proxyHost string) {
			backendLines = append(
				backendLines,
				keyEqualsValue("proxy.type", optional.Of(c.Spec.Agent.ProxyProtocol).GetOrDefault("HTTP")),
				keyEqualsValue("proxy.host", proxyHost),
				keyEqualsValue(
					"proxy.port", optional.Of(c.Spec.Agent.ProxyPort).GetOrDefault("80"),
				),
			)
		},
	)

	optional.Of(c.Spec.Agent.ProxyUseDNS).IfPresent(
		func(useDns bool) {
			backendLines = append(backendLines, keyEqualsValue("proxy.dns", "true"))
		},
	)

	optional.Of(c.Spec.Agent.ProxyUser).IfPresent(
		func(proxyUser string) {
			backendLines = append(backendLines, keyEqualsValue("proxy.user", proxyUser))
		},
	)

	optional.Of(c.Spec.Agent.ProxyPassword).IfPresent(
		func(proxyPassword string) {
			backendLines = append(backendLines, keyEqualsValue("proxy.password", proxyPassword))
		},
	)

	if c.Spec.Agent.ProxyUseDNS {
		backendLines = append(backendLines, keyEqualsValue("proxyUseDNS", "true"))
	}

	res["com.instana.agent.main.sender.Backend-1.cfg"] = []byte(strings.Join(backendLines, "\n"))

	for i, backend := range c.Spec.Agent.AdditionalBackends {
		lines := make([]string, 0, 10)
		lines = append(
			lines,
			keyEqualsValue("host", backend.EndpointHost),
			keyEqualsValue("port", optional.Of(backend.EndpointPort).GetOrDefault("443")),
			keyEqualsValue("protocol", "HTTP/2"),
			keyEqualsValue("key", backend.Key),
		)

		optional.Of(c.Spec.Agent.ProxyHost).IfPresent(
			func(proxyHost string) {
				lines = append(
					lines,
					keyEqualsValue("proxy.type", optional.Of(c.Spec.Agent.ProxyProtocol).GetOrDefault("HTTP")),
					keyEqualsValue("proxy.host", proxyHost),
					keyEqualsValue(
						"proxy.port", optional.Of(c.Spec.Agent.ProxyPort).GetOrDefault("80"),
					),
				)
			},
		)

		optional.Of(c.Spec.Agent.ProxyUseDNS).IfPresent(
			func(useDns bool) {
				lines = append(lines, keyEqualsValue("proxy.dns", "true"))
			},
		)

		optional.Of(c.Spec.Agent.ProxyUser).IfPresent(
			func(proxyUser string) {
				lines = append(lines, keyEqualsValue("proxy.user", proxyUser))
			},
		)

		optional.Of(c.Spec.Agent.ProxyPassword).IfPresent(
			func(proxyPassword string) {
				lines = append(lines, keyEqualsValue("proxy.password", proxyPassword))
			},
		)

		if c.Spec.Agent.ProxyUseDNS {
			lines = append(lines, keyEqualsValue("proxyUseDNS", "true"))
		}

		res["com.instana.agent.main.sender.Backend-"+strconv.Itoa(i+2)+".cfg"] = []byte(strings.Join(lines, "\n"))
	}

	return res

}

func yamlOrDie(obj any) string {
	return string(
		or_die.New[[]byte]().
			ResultOrDie(
				func() ([]byte, error) {
					return yaml.Marshal(obj)
				},
			),
	)
}

func keyEqualsValue(key string, value string) string {
	return key + "=" + value
}
