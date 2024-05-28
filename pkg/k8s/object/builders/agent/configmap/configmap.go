package configmap

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

type configMapBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager
}

func (c *configMapBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (c *configMapBuilder) IsNamespaced() bool {
	return true
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

func (c *configMapBuilder) getData() map[string]string {
	res := make(map[string]string)

	optional.Of(c.Spec.Cluster.Name).IfPresent(
		func(clusterName string) {
			res["cluster_name"] = clusterName
		},
	)

	optional.Of(c.Spec.Agent.ConfigurationYaml).IfPresent(
		func(configYaml string) {
			res["configuration.yaml"] = configYaml
		},
	)

	if otlp := c.Spec.OpenTelemetry; otlp.IsEnabled() {
		otlpPluginSettings := map[string]instanav1.OpenTelemetry{"com.instana.plugin.opentelemetry": otlp}
		res["configuration-opentelemetry.yaml"] = yamlOrDie(&otlpPluginSettings)
	}

	if pointer.DerefOrEmpty(c.Spec.Prometheus.RemoteWrite.Enabled) {
		res["configuration-prometheus-remote-write.yaml"] = yamlOrDie(
			map[string]any{
				"com.instana.plugin.prometheus": map[string]any{
					"remote_write": map[string]any{
						"enabled": true,
					},
				},
			},
		)
	}

	// Deprecated since k8s sensor deployment will always be enabled now,
	// can remove once deprecated sensor is removed from agent

	res["configuration-disable-kubernetes-sensor.yaml"] = yamlOrDie(
		map[string]any{
			"com.instana.plugin.kubernetes": map[string]any{
				"enabled": false,
			},
		},
	)

	for i, backend := range c.Spec.Agent.AdditionalBackends {
		lines := make([]string, 0, 10)
		lines = append(
			lines,
			keyEqualsValue("host", backend.EndpointHost),
			keyEqualsValue("port", optional.Of(backend.EndpointPort).GetOrDefault("443")),
			keyEqualsValue("key", backend.Key),
			keyEqualsValue("protocol", "HTTP/2"),
		)

		optional.Of(c.Spec.Agent.ProxyHost).IfPresent(
			func(proxyHost string) {
				lines = append(
					lines,
					keyEqualsValue("proxy.type", "HTTP"),
					keyEqualsValue("proxy.host", proxyHost),
					keyEqualsValue(
						"proxy.port", optional.Of(c.Spec.Agent.ProxyPort).GetOrDefault("80"),
					),
				)
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

		res["additional-backend-"+strconv.Itoa(i+2)] = strings.Join(lines, "\n")
	}

	return res

}

func (c *configMapBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(cm client.Object) {
				c.statusManager.SetAgentConfigMap(client.ObjectKeyFromObject(cm))
			},
		)
	}()

	return optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      c.Name,
				Namespace: c.Namespace,
			},
			Data: c.getData(),
		},
	)
}

func NewConfigMapBuilder(agent *instanav1.InstanaAgent, statusManager status.AgentStatusManager) builder.ObjectBuilder {
	return &configMapBuilder{
		InstanaAgent:  agent,
		statusManager: statusManager,
	}
}
