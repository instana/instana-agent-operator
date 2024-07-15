/*
(c) Copyright IBM Corp. 2024
*/

package instana_agent_config

import (
	"errors"
	"fmt"
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
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

var log = logf.Log.WithName("instana-agent-config-secret-builder")

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

	if c.Spec.Cluster.Name != "" {
		res["cluster_name"] = []byte(c.Spec.Cluster.Name)
	}

	if c.Spec.Agent.ConfigurationYaml != "" {
		res["configuration.yaml"] = []byte(c.Spec.Agent.ConfigurationYaml)
	}

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

	// appending the backend config
	backendConfig, _ := getBackendConfig(c)

	for k, v := range backendConfig {
		res[k] = v
	}

	return res
}

func getBackendConfig(c *secretBuilder) (map[string][]byte, error) {
	res := make(map[string][]byte, 2)

	// render additional backends configuration
	for i, backend := range c.Spec.Agent.AdditionalBackends {
		lines := make([]string, 0, 10)
		if backend.Key == "" || backend.EndpointHost == "" {
			// skip backend as it would be broken anyways, should be caught by the schema validator anyways, but better be safe than sorry
			log.Error(fmt.Errorf("key or endpointHost undefined"), "skipping additional backend due to missing values")
			continue
		}
		lines = append(
			lines,
			keyEqualsValue("host", backend.EndpointHost),
			keyEqualsValueWithDefault("port", backend.EndpointPort, "443"),
			keyEqualsValue("protocol", "HTTP/2"),
			keyEqualsValue("key", backend.Key),
		)

		if c.Spec.Agent.ProxyHost != "" {
			lines = append(
				lines,
				keyEqualsValueWithDefault("proxy.type", c.Spec.Agent.ProxyProtocol, "HTTP"),
				keyEqualsValue("proxy.host", c.Spec.Agent.ProxyHost),
				keyEqualsValueWithDefault("proxy.port", c.Spec.Agent.ProxyPort, "80"),
			)
		}

		if c.Spec.Agent.ProxyUseDNS {
			lines = append(lines, keyEqualsValue("proxy.dns", "true"))
		}

		if c.Spec.Agent.ProxyUser != "" && c.Spec.Agent.ProxyPassword != "" {
			lines = append(
				lines,
				keyEqualsValue("proxy.user", c.Spec.Agent.ProxyUser),
				keyEqualsValue("proxy.password", c.Spec.Agent.ProxyPassword),
			)
		}

		res["com.instana.agent.main.sender.Backend-"+strconv.Itoa(i+2)+".cfg"] = []byte(strings.Join(lines, "\n") + "\n")
	}

	if c.Spec.Agent.EndpointHost == "" {
		// We don't have sufficient information to produce this backend, an endpointHost is required, skip rendering and return
		return res, errors.New("spec.Agent.EndpointHost is not defined and required to render the given backend config")
	}

	var agentKey string
	if keyValueFromSecret, ok := c.keysSecret.Data["key"]; ok {
		agentKey = string(keyValueFromSecret)
		if c.Spec.Agent.Key != "" {
			log.V(1).Info("keysSecret and spec.agent.key are both defined, preferring keysSecret")
		}
	} else {
		if c.Spec.Agent.Key != "" {
			agentKey = string(c.Spec.Agent.Key)
		} else {
			err := errors.New("keysSecret does not contain key attribute and spec.Agent.Key is not defined either")
			log.Error(err, "Missing agent key, skipping to render main backend")
			return res, err
		}
	}

	backendLines := make([]string, 0, 10)
	backendLines = append(
		backendLines,
		keyEqualsValue("host", c.Spec.Agent.EndpointHost),
		keyEqualsValueWithDefault("port", c.Spec.Agent.EndpointPort, "443"),
		keyEqualsValue("protocol", "HTTP/2"),
	)

	backendLines = append(
		backendLines,
		keyEqualsValue("key", string(agentKey)),
	)

	if c.Spec.Agent.ProxyHost != "" {
		backendLines = append(
			backendLines,
			keyEqualsValueWithDefault("proxy.type", c.Spec.Agent.ProxyProtocol, "HTTP"),
			keyEqualsValue("proxy.host", c.Spec.Agent.ProxyHost),
			keyEqualsValueWithDefault("proxy.port", c.Spec.Agent.ProxyPort, "80"),
		)
	}

	if c.Spec.Agent.ProxyUseDNS {
		backendLines = append(backendLines, keyEqualsValue("proxy.dns", "true"))
	}

	if c.Spec.Agent.ProxyUser != "" {
		backendLines = append(backendLines, keyEqualsValue("proxy.user", c.Spec.Agent.ProxyUser))
	}

	if c.Spec.Agent.ProxyPassword != "" {
		backendLines = append(backendLines, keyEqualsValue("proxy.password", c.Spec.Agent.ProxyPassword))
	}

	res["com.instana.agent.main.sender.Backend-1.cfg"] = []byte(strings.Join(backendLines, "\n") + "\n")
	return res, nil
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

func keyEqualsValueWithDefault(key string, value string, defaultValue string) string {
	if value != "" {
		return keyEqualsValue(key, value)
	}
	return keyEqualsValue(key, defaultValue)
}

func keyEqualsValue(key string, value string) string {
	return key + "=" + value
}
