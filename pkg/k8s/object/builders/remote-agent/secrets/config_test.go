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

package secrets

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	backend "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/stretchr/testify/assert"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConfigBuilderComponentName(t *testing.T) {
	ctrl := gomock.NewController(t)
	statusManager := mocks.NewMockRemoteAgentStatusManager(ctrl)
	s := NewConfigBuilder(&instanav1.RemoteAgent{}, statusManager, &corev1.Secret{}, make([]backend.K8SensorBackend, 0))

	assert.True(t, s.IsNamespaced())
}

func TestConfigBuilderIsNamespaced(t *testing.T) {
	ctrl := gomock.NewController(t)
	statusManager := mocks.NewMockRemoteAgentStatusManager(ctrl)
	s := NewConfigBuilder(&instanav1.RemoteAgent{}, statusManager, &corev1.Secret{}, make([]backend.K8SensorBackend, 0))

	assert.Equal(t, "remote-instana-agent", s.ComponentName())
}

// TestAgentSecretConfigBuild is a all-in-one test to go through the agent config to extract secrets in all cases
func TestAgentSecretConfigBuild(t *testing.T) {
	objectMeta := metav1.ObjectMeta{
		Name:      "object-name-value",
		Namespace: "object-namespace-value",
	}
	objectMetaConfig := metav1.ObjectMeta{
		Name:      objectMeta.Name + "-config",
		Namespace: objectMeta.Namespace,
	}
	cluster := instanav1.Name{
		Name: objectMeta.Name,
	}
	backends := []instanav1.BackendSpec{
		{
			EndpointHost: "additional-backend-2-host",
			EndpointPort: "additional-backend-2-port",
			Key:          "additional-backend-2-key",
		},
		{
			EndpointHost: "additional-backend-3-host",
			EndpointPort: "additional-backend-3-port",
			Key:          "additional-backend-3-key",
		},
		{
			EndpointHost: "additional-backend-4-host",
			EndpointPort: "additional-backend-4-port",
			Key:          "additional-backend-4-key",
		},
	}
	secretType := metav1.TypeMeta{
		APIVersion: "v1",
		Kind:       "Secret",
	}

	for _, test := range []struct {
		name        string
		agent       instanav1.RemoteAgent
		k8sBackends []backend.K8SensorBackend
		keysSecret  *corev1.Secret
		expected    map[string][]byte
	}{
		{
			name: "Should return v1.Secret struct containing data from the InstanaAgentSpec as Backend-1.cfg with inline field, yaml and pure string fields",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Cluster: cluster,
					Agent: instanav1.BaseAgentSpec{
						EndpointHost:      "main-backend-host",
						EndpointPort:      "main-backend-port",
						Key:               "main-backend-key",
						ConfigurationYaml: "configuration-yaml-value",
						ProxyHost:         "proxy-host-value",
						ProxyPort:         "proxy-port-value",
						ProxyUser:         "proxy-user-value",
						ProxyPassword:     "proxy-password-value",
						ProxyUseDNS:       true,
					},
				},
			},
			k8sBackends: []backend.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointHost:   "main-backend-host",
					EndpointPort:   "main-backend-port",
					EndpointKey:    "main-backend-key",
				},
			},
			keysSecret: &corev1.Secret{},
			expected: map[string][]byte{
				"cluster_name":       []byte(objectMeta.Name),
				"configuration.yaml": []byte("configuration-yaml-value"),
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
				"com.instana.agent.main.sender.Backend-1.cfg":  []byte("host=main-backend-host\nport=main-backend-port\nprotocol=HTTP/2\nkey=main-backend-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
			},
		},
		{
			name: "Should return v1.Secret struct containing Backend-2+ configs as Backend-1 configuration is missing",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						ProxyHost:          "proxy-host-value",
						ProxyPort:          "proxy-port-value",
						ProxyUser:          "proxy-user-value",
						ProxyPassword:      "proxy-password-value",
						ProxyUseDNS:        true,
						AdditionalBackends: backends,
					},
				},
			},
			k8sBackends: []backend.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointHost:   "",
					EndpointPort:   "",
					EndpointKey:    "",
				},
				{
					ResourceSuffix: "-2",
					EndpointHost:   "additional-backend-2-host",
					EndpointPort:   "additional-backend-2-port",
					EndpointKey:    "additional-backend-2-key",
				},
				{
					ResourceSuffix: "-3",
					EndpointHost:   "additional-backend-3-host",
					EndpointPort:   "additional-backend-3-port",
					EndpointKey:    "additional-backend-3-key",
				},
				{
					ResourceSuffix: "-4",
					EndpointHost:   "additional-backend-4-host",
					EndpointPort:   "additional-backend-4-port",
					EndpointKey:    "additional-backend-4-key",
				},
			},
			keysSecret: &corev1.Secret{},
			expected: map[string][]byte{
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
				"com.instana.agent.main.sender.Backend-2.cfg":  []byte("host=additional-backend-2-host\nport=additional-backend-2-port\nprotocol=HTTP/2\nkey=additional-backend-2-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
				"com.instana.agent.main.sender.Backend-3.cfg":  []byte("host=additional-backend-3-host\nport=additional-backend-3-port\nprotocol=HTTP/2\nkey=additional-backend-3-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
				"com.instana.agent.main.sender.Backend-4.cfg":  []byte("host=additional-backend-4-host\nport=additional-backend-4-port\nprotocol=HTTP/2\nkey=additional-backend-4-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
			},
		},
		{
			name: "Should return corev1.Secret struct containing both Backend-1 and additional backends",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						EndpointHost:       "main-backend-host",
						EndpointPort:       "main-backend-port",
						Key:                "main-backend-key",
						ProxyHost:          "proxy-host-value",
						ProxyPort:          "proxy-port-value",
						ProxyUser:          "proxy-user-value",
						ProxyPassword:      "proxy-password-value",
						ProxyUseDNS:        true,
						AdditionalBackends: backends,
					},
				},
			},
			keysSecret: &corev1.Secret{},
			k8sBackends: []backend.K8SensorBackend{
				{
					EndpointHost: "main-backend-host",
					EndpointPort: "main-backend-port",
					EndpointKey:  "main-backend-key",
				},
				{
					ResourceSuffix: "-2",
					EndpointHost:   "additional-backend-2-host",
					EndpointPort:   "additional-backend-2-port",
					EndpointKey:    "additional-backend-2-key",
				},
				{
					ResourceSuffix: "-3",
					EndpointHost:   "additional-backend-3-host",
					EndpointPort:   "additional-backend-3-port",
					EndpointKey:    "additional-backend-3-key",
				},
				{
					ResourceSuffix: "-4",
					EndpointHost:   "additional-backend-4-host",
					EndpointPort:   "additional-backend-4-port",
					EndpointKey:    "additional-backend-4-key",
				},
			},
			expected: map[string][]byte{
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
				"com.instana.agent.main.sender.Backend-1.cfg":  []byte("host=main-backend-host\nport=main-backend-port\nprotocol=HTTP/2\nkey=main-backend-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
				"com.instana.agent.main.sender.Backend-2.cfg":  []byte("host=additional-backend-2-host\nport=additional-backend-2-port\nprotocol=HTTP/2\nkey=additional-backend-2-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
				"com.instana.agent.main.sender.Backend-3.cfg":  []byte("host=additional-backend-3-host\nport=additional-backend-3-port\nprotocol=HTTP/2\nkey=additional-backend-3-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
				"com.instana.agent.main.sender.Backend-4.cfg":  []byte("host=additional-backend-4-host\nport=additional-backend-4-port\nprotocol=HTTP/2\nkey=additional-backend-4-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
			},
		},
		{
			name: "Should create the config v1.Secret without Backend-N.cfg fields when nothing was specified",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{},
				},
			},
			keysSecret: &corev1.Secret{},
			expected: map[string][]byte{
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
			},
		},
		{
			name: "Should use the v1.Secret provided key in the Backend-1.cfg when keysSecret has been specified",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						ProxyHost:     "proxy-host-value",
						ProxyPort:     "proxy-port-value",
						ProxyUser:     "proxy-user-value",
						ProxyPassword: "proxy-password-value",
						ProxyUseDNS:   true,
						EndpointHost:  "main-backend-host",
						EndpointPort:  "main-backend-port",
					},
				},
			},
			keysSecret: &corev1.Secret{
				Type:       corev1.SecretTypeOpaque,
				TypeMeta:   secretType,
				ObjectMeta: objectMeta,
				Data: map[string][]byte{
					"key": []byte("key-from-secret"),
				},
			},
			k8sBackends: []backend.K8SensorBackend{
				{
					EndpointHost: "main-backend-host",
					EndpointPort: "main-backend-port",
				},
			},
			expected: map[string][]byte{
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
				"com.instana.agent.main.sender.Backend-1.cfg":  []byte("host=main-backend-host\nport=main-backend-port\nprotocol=HTTP/2\nkey=key-from-secret\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
			},
		},
		{
			name: "Should use the v1.Secret provided key in the Backend-1.cfg when keysSecret has been specified",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						ProxyHost:     "proxy-host-value",
						ProxyPort:     "proxy-port-value",
						ProxyUser:     "proxy-user-value",
						ProxyPassword: "proxy-password-value",
						ProxyUseDNS:   true,
						EndpointHost:  "main-backend-host",
						EndpointPort:  "main-backend-port",
						AdditionalBackends: []instanav1.BackendSpec{
							{
								EndpointPort: "additional-backend-2-port",
							},
							{
								EndpointHost: "additional-backend-3-host",
								EndpointPort: "additional-backend-3-port",
								Key:          "additional-backend-3-key",
							},
						},
					},
				},
			},
			keysSecret: &corev1.Secret{
				Type:       corev1.SecretTypeOpaque,
				TypeMeta:   secretType,
				ObjectMeta: objectMeta,
				Data: map[string][]byte{
					"key": []byte("key-from-secret"),
				},
			},
			k8sBackends: []backend.K8SensorBackend{
				{
					ResourceSuffix: "",
					EndpointHost:   "main-backend-host",
					EndpointPort:   "main-backend-port",
				},
				{
					ResourceSuffix: "-2",
					EndpointPort:   "additional-backend-2-port",
				},
				{
					ResourceSuffix: "-3",
					EndpointHost:   "additional-backend-3-host",
					EndpointPort:   "additional-backend-3-port",
					EndpointKey:    "additional-backend-3-key",
				},
			},
			expected: map[string][]byte{
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
				"com.instana.agent.main.sender.Backend-1.cfg":  []byte("host=main-backend-host\nport=main-backend-port\nprotocol=HTTP/2\nkey=key-from-secret\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
				"com.instana.agent.main.sender.Backend-3.cfg":  []byte("host=additional-backend-3-host\nport=additional-backend-3-port\nprotocol=HTTP/2\nkey=additional-backend-3-key\nproxy.type=HTTP\nproxy.host=proxy-host-value\nproxy.port=proxy-port-value\nproxy.dns=true\nproxy.user=proxy-user-value\nproxy.password=proxy-password-value\n"),
			},
		},
		{
			name: "Should not add any backends when keys dont exist",
			agent: instanav1.RemoteAgent{
				ObjectMeta: objectMeta,
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						ProxyHost:     "proxy-host-value",
						ProxyPort:     "proxy-port-value",
						ProxyUser:     "proxy-user-value",
						ProxyPassword: "proxy-password-value",
						ProxyUseDNS:   true,
						EndpointHost:  "main-backend-host",
						EndpointPort:  "main-backend-port",
						AdditionalBackends: []instanav1.BackendSpec{
							{
								EndpointPort: "additional-backend-2-port",
							},
						},
					},
				},
			},
			keysSecret: &corev1.Secret{},
			k8sBackends: []backend.K8SensorBackend{
				{
					EndpointHost: "main-backend-host",
					EndpointPort: "main-backend-port",
				},
				{
					ResourceSuffix: "-2",
					EndpointPort:   "additional-backend-2-port",
				},
			},
			expected: map[string][]byte{
				"configuration-disable-kubernetes-sensor.yaml": []byte("com.instana.plugin.kubernetes:\n    enabled: false\n"),
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				ctrl := gomock.NewController(t)

				statusManager := mocks.NewMockRemoteAgentStatusManager(ctrl)
				statusManager.EXPECT().SetAgentSecretConfig(gomock.Any()).AnyTimes()

				builder := NewConfigBuilder(&test.agent, statusManager, test.keysSecret, test.k8sBackends)

				actual := builder.Build().Get()

				expected := &corev1.Secret{
					Type:       corev1.SecretTypeOpaque,
					TypeMeta:   secretType,
					ObjectMeta: objectMetaConfig,
					Data:       test.expected,
				}

				assert.Equal(t, expected, actual)
			},
		)
	}
}
