package agent_configmap

import (
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/or_die"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

type agentConfigMapBuilder struct {
	*instanav1.InstanaAgent
}

func (a *agentConfigMapBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (a *agentConfigMapBuilder) IsNamespaced() bool {
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

func (a *agentConfigMapBuilder) getData() map[string]string {
	res := make(map[string]string)

	optional.Of(a.Spec.Cluster.Name).IfPresent(
		func(clusterName string) {
			res["cluster_name"] = clusterName
		},
	)

	optional.Of(a.Spec.Agent.ConfigurationYaml).IfPresent(
		func(configYaml string) {
			res["configuration.yaml"] = configYaml
		},
	)

	if otlp := a.Spec.OpenTelemetry; otlp.IsEnabled() {
		res["configuration-opentelemetry.yaml"] = yamlOrDie(&otlp)
	}

	if pointer.DerefOrEmpty(a.Spec.Prometheus.RemoteWrite.Enabled) {
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

	for i, backend := range a.Spec.Agent.AdditionalBackends {
		lines := make([]string, 0, 10)
		lines = append(
			lines,
			keyEqualsValue("host", backend.EndpointHost),
			keyEqualsValue("port", optional.Of(backend.EndpointPort).GetOrDefault("443")),
			keyEqualsValue("key", backend.Key),
			keyEqualsValue("protocol", "HTTP/2"),
		)

		optional.Of(a.Spec.Agent.ProxyHost).IfPresent(
			func(proxyHost string) {
				lines = append(
					lines,
					keyEqualsValue("proxy.type", "HTTP"),
					keyEqualsValue("proxy.host", proxyHost),
					keyEqualsValue(
						"proxy.port", optional.Of(a.Spec.Agent.ProxyPort).GetOrDefault("80"),
					),
				)
			},
		)

		optional.Of(a.Spec.Agent.ProxyUser).IfPresent(
			func(proxyUser string) {
				lines = append(lines, keyEqualsValue("proxy.user", proxyUser))
			},
		)

		optional.Of(a.Spec.Agent.ProxyPassword).IfPresent(
			func(proxyPassword string) {
				lines = append(lines, keyEqualsValue("proxy.password", proxyPassword))
			},
		)

		if a.Spec.Agent.ProxyUseDNS {
			lines = append(lines, keyEqualsValue("proxyUseDNS", "true"))
		}

		res["additional-backend-"+strconv.Itoa(i+2)] = strings.Join(lines, "\n")
	}

	return res

}

func (a *agentConfigMapBuilder) Build() optional.Optional[client.Object] {
	return optional.Of[client.Object](
		&corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "ConfigMap",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      a.Name,
				Namespace: a.Namespace,
			},
			Data: a.getData(),
		},
	)
}

// TODO: standardize constructors for cleaner initialization

func NewAgentConfigMapBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &agentConfigMapBuilder{
		InstanaAgent: agent,
	}
}
