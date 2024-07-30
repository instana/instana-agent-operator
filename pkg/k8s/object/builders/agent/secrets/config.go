/*
(c) Copyright IBM Corp. 2024
*/

package secrets

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-logr/logr"
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	commonbuilder "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"gopkg.in/yaml.v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	logf "sigs.k8s.io/controller-runtime/pkg/log"
)

type configBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager
	keysSecret    *corev1.Secret
	logger        logr.Logger
}

func NewConfigBuilder(
	agent *instanav1.InstanaAgent,
	statusManager status.AgentStatusManager,
	keysSecret *corev1.Secret) commonbuilder.ObjectBuilder {
	return &configBuilder{
		InstanaAgent:  agent,
		statusManager: statusManager,
		keysSecret:    keysSecret,
		logger:        logf.Log.WithName("instana-agent-config-secret-builder"),
	}
}

func (c *configBuilder) ComponentName() string {
	return constants.ComponentInstanaAgent
}

func (c *configBuilder) IsNamespaced() bool {
	return true
}

func (c *configBuilder) Build() optional.Optional[client.Object] {
	data, errs := c.data()
	if errs != nil {
		c.logger.Error(errs, "errors occurred while attempting to generate v1.Secret data-field")
	}

	secret := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Secret",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      c.Name + "-config",
			Namespace: c.Namespace,
		},
		Data: data,
		Type: corev1.SecretTypeOpaque,
	}

	c.statusManager.SetAgentSecretConfig(client.ObjectKeyFromObject(secret))

	return optional.Of[client.Object](secret)
}

func (c *configBuilder) data() (map[string][]byte, error) {
	data := map[string][]byte{}

	if c.Spec.Cluster.Name != "" {
		data["cluster_name"] = []byte(c.Spec.Cluster.Name)
	}
	if c.Spec.Agent.ConfigurationYaml != "" {
		data["configuration.yaml"] = []byte(c.Spec.Agent.ConfigurationYaml)
	}
	if otlp := c.Spec.OpenTelemetry; otlp.IsEnabled() {
		mrshl, _ := yaml.Marshal(map[string]instanav1.OpenTelemetry{"com.instana.plugin.opentelemetry": otlp})
		data["configuration-opentelemetry.yaml"] = mrshl
	}
	if pointer.DerefOrEmpty(c.Spec.Prometheus.RemoteWrite.Enabled) {
		mrshl, _ := yaml.Marshal(
			map[string]any{
				"com.instana.plugin.prometheus": map[string]any{
					"remote_write": map[string]any{
						"enabled": true,
					},
				},
			},
		)
		data["configuration-prometheus-remote-write.yaml"] = mrshl
	}

	// Deprecated since k8s sensor deployment will always be enabled now,
	// can remove once deprecated sensor is removed from agent
	mrshl, _ := yaml.Marshal(
		map[string]any{
			"com.instana.plugin.kubernetes": map[string]any{
				"enabled": false,
			},
		},
	)
	data["configuration-disable-kubernetes-sensor.yaml"] = mrshl

	backendConfig, err := c.backendConfig()

	return mergeMaps(data, backendConfig), err
}

func (c *configBuilder) backendConfig() (map[string][]byte, error) {
	config := map[string][]byte{}

	// render additional backends configuration
	for i, backend := range c.Spec.Agent.AdditionalBackends {
		if backend.Key == "" || backend.EndpointHost == "" {
			// skip backend as it would be broken anyways, should be caught by the schema validator anyways, but better be safe than sorry
			c.logger.Error(fmt.Errorf("key or endpointHost undefined"), "skipping additional backend due to missing values")
			continue
		}

		lines := []string{
			toInlineVariable("host", backend.EndpointHost),
			toInlineVariable("port", backend.EndpointPort, "443"),
			toInlineVariable("protocol", "HTTP/2"),
			toInlineVariable("key", backend.Key),
		}
		if c.Spec.Agent.ProxyHost != "" {
			lines = append(
				lines,
				toInlineVariable("proxy.type", c.Spec.Agent.ProxyProtocol, "HTTP"),
				toInlineVariable("proxy.host", c.Spec.Agent.ProxyHost),
				toInlineVariable("proxy.port", c.Spec.Agent.ProxyPort, "80"),
			)
		}
		if c.Spec.Agent.ProxyUseDNS {
			lines = append(lines, toInlineVariable("proxy.dns", strconv.FormatBool(c.Spec.Agent.ProxyUseDNS)))
		}
		if c.Spec.Agent.ProxyUser != "" && c.Spec.Agent.ProxyPassword != "" {
			lines = append(
				lines,
				toInlineVariable("proxy.user", c.Spec.Agent.ProxyUser),
				toInlineVariable("proxy.password", c.Spec.Agent.ProxyPassword),
			)
		}

		config["com.instana.agent.main.sender.Backend-"+strconv.Itoa(i+2)+".cfg"] = []byte(strings.Join(lines, "\n") + "\n")
	}

	if c.Spec.Agent.EndpointHost == "" {
		return config, errors.New("agent endpoint host has not been set")
	}

	var agentKey string
	if keyValueFromSecret, ok := c.keysSecret.Data["key"]; ok {
		agentKey = string(keyValueFromSecret)
	} else if c.Spec.Agent.Key != "" {
		agentKey = string(c.Spec.Agent.Key)
	} else {
		return config, errors.New("agent key has not been set")
	}

	backendLines := []string{
		toInlineVariable("host", c.Spec.Agent.EndpointHost),
		toInlineVariable("port", c.Spec.Agent.EndpointPort, "443"),
		toInlineVariable("protocol", "HTTP/2"),
		toInlineVariable("key", agentKey),
	}
	if c.Spec.Agent.ProxyHost != "" {
		backendLines = append(
			backendLines,
			toInlineVariable("proxy.type", c.Spec.Agent.ProxyProtocol, "HTTP"),
			toInlineVariable("proxy.host", c.Spec.Agent.ProxyHost),
			toInlineVariable("proxy.port", c.Spec.Agent.ProxyPort, "80"),
		)
	}
	if c.Spec.Agent.ProxyUseDNS {
		backendLines = append(backendLines, toInlineVariable("proxy.dns", "true"))
	}
	if c.Spec.Agent.ProxyUser != "" {
		backendLines = append(backendLines, toInlineVariable("proxy.user", c.Spec.Agent.ProxyUser))
	}
	if c.Spec.Agent.ProxyPassword != "" {
		backendLines = append(backendLines, toInlineVariable("proxy.password", c.Spec.Agent.ProxyPassword))
	}

	config["com.instana.agent.main.sender.Backend-1.cfg"] = []byte(strings.Join(backendLines, "\n") + "\n")

	return config, nil
}

// toInlineVariable stringifies in "key=value" format with a fallback value if value ends up being empty
func toInlineVariable(key string, value string, fallback ...string) string {
	if len(fallback) > 0 && value == "" {
		return key + "=" + fallback[0]
	}
	return key + "=" + value
}

func mergeMaps(map1, map2 map[string][]byte) map[string][]byte {
	for key, value := range map2 {
		map1[key] = value
	}
	return map1
}
