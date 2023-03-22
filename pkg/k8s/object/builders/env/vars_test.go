package env

import (
	"testing"

	"github.com/instana/instana-agent-operator/pkg/pointer"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/stretchr/testify/require"
)

func testOptionalEnv(
	t *testing.T,
	f func(agent *instanav1.InstanaAgent) EnvBuilder,
	agent *instanav1.InstanaAgent,
	expectedName string,
	expectedValue string,
) {
	t.Run("when_empty", func(t *testing.T) {
		assertions := require.New(t)
		actual := f(&instanav1.InstanaAgent{}).Build()

		assertions.Empty(actual)
	})
	t.Run("with_value", func(t *testing.T) {
		assertions := require.New(t)
		actual := f(agent).Build()

		assertions.Equal(
			optional.Of(corev1.EnvVar{
				Name:  expectedName,
				Value: expectedValue,
			}),
			actual,
		)
	})
}

func TestAgentModeEnv(t *testing.T) {
	testOptionalEnv(
		t,
		AgentModeEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Mode: instanav1.KUBERNETES,
				},
			},
		},
		"INSTANA_AGENT_MODE",
		string(instanav1.KUBERNETES),
	)
}

func TestZoneNameEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ZoneNameEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Zone: instanav1.Name{
					Name: "oiweoiohewf",
				},
			},
		},
		"INSTANA_ZONE",
		"oiweoiohewf",
	)
}

func TestClusterNameEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ClusterNameEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Cluster: instanav1.Name{
					Name: "oiweoiohewf",
				},
			},
		},
		"INSTANA_KUBERNETES_CLUSTER_NAME",
		"oiweoiohewf",
	)
}

func TestAgentEndpointEnv(t *testing.T) {
	testOptionalEnv(
		t,
		AgentEndpointEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					EndpointHost: "kljdskoije",
				},
			},
		},
		"INSTANA_AGENT_ENDPOINT",
		"kljdskoije",
	)
}

func TestAgentEndpointPortEnv(t *testing.T) {
	testOptionalEnv(
		t,
		AgentEndpointPortEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					EndpointPort: "480230932",
				},
			},
		},
		"INSTANA_AGENT_ENDPOINT_PORT",
		"480230932",
	)
}

func TestMavenRepoUrlEnv(t *testing.T) {
	testOptionalEnv(
		t,
		MavenRepoUrlEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					MvnRepoUrl: "tgiojreoihsef",
				},
			},
		},
		"INSTANA_MVN_REPOSITORY_URL",
		"tgiojreoihsef",
	)
}

func TestProxyHostEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ProxyHostEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyHost: "giroijwsoidoisd",
				},
			},
		},
		"INSTANA_AGENT_PROXY_HOST",
		"giroijwsoidoisd",
	)
}

func TestProxyPortEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ProxyPortEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyPort: "boieoijspojs",
				},
			},
		},
		"INSTANA_AGENT_PROXY_PORT",
		"boieoijspojs",
	)
}

func TestProxyProtocolEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ProxyProtocolEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyProtocol: "eoidoijsoihe",
				},
			},
		},
		"INSTANA_AGENT_PROXY_PROTOCOL",
		"eoidoijsoihe",
	)
}

func TestProxyUserEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ProxyUserEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyUser: "hoieoijsdoifjsd",
				},
			},
		},
		"INSTANA_AGENT_PROXY_USER",
		"hoieoijsdoifjsd",
	)
}

func TestProxyPasswordEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ProxyPasswordEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyPassword: "ruiohdoigjseijsdf",
				},
			},
		},
		"INSTANA_AGENT_PROXY_PASSWORD",
		"ruiohdoigjseijsdf",
	)
}

func TestProxyUseDNSEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ProxyUseDNSEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyUseDNS: true,
				},
			},
		},
		"INSTANA_AGENT_PROXY_USE_DNS",
		"true",
	)
}

func TestListenAddressEnv(t *testing.T) {
	testOptionalEnv(
		t,
		ListenAddressEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ListenAddress: "iroihlksdifho",
				},
			},
		},
		"INSTANA_AGENT_HTTP_LISTEN",
		"iroihlksdifho",
	)
}

func TestRedactK8sSecretsEnv(t *testing.T) {
	testOptionalEnv(
		t,
		RedactK8sSecretsEnv,
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					RedactKubernetesSecrets: "w894309u2oijsgf",
				},
			},
		},
		"INSTANA_KUBERNETES_REDACT_SECRETS",
		"w894309u2oijsgf",
	)
}

func TestAgentKeyEnv(t *testing.T) {
	t.Run("no_user_provided_secret", func(t *testing.T) {
		t.Run("keys_secret_not_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			h := AgentKeyEnv(&instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "riuoidfoisd",
				},
			})
			actual := h.Build()

			assertions.Equal(
				optional.Of(corev1.EnvVar{
					Name: "INSTANA_AGENT_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "riuoidfoisd",
							},
							Key: "key",
						},
					},
				}),
				actual,
			)
		})
		t.Run("keys_secret_is_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			h := AgentKeyEnv(&instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oiew9oisdoijdsf",
				},
				Spec: instanav1.InstanaAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						KeysSecret: "riuoidfoisd",
					},
				},
			})
			actual := h.Build()

			assertions.Equal(
				optional.Of(corev1.EnvVar{
					Name: "INSTANA_AGENT_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "riuoidfoisd",
							},
							Key: "key",
						},
					},
				}),
				actual,
			)
		})
	})
}

func TestDownloadKeyEnv(t *testing.T) {
	t.Run("no_user_provided_secret", func(t *testing.T) {
		t.Run("keys_secret_not_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			h := DownloadKeyEnv(&instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "riuoidfoisd",
				},
			})
			actual := h.Build()

			assertions.Equal(
				optional.Of(corev1.EnvVar{
					Name: "INSTANA_DOWNLOAD_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "riuoidfoisd",
							},
							Key:      "downloadKey",
							Optional: pointer.To(true),
						},
					},
				}),
				actual,
			)
		})
		t.Run("keys_secret_is_provided_by_user", func(t *testing.T) {
			assertions := require.New(t)

			h := DownloadKeyEnv(&instanav1.InstanaAgent{
				ObjectMeta: metav1.ObjectMeta{
					Name: "oiew9oisdoijdsf",
				},
				Spec: instanav1.InstanaAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						KeysSecret: "riuoidfoisd",
					},
				},
			})
			actual := h.Build()

			assertions.Equal(
				optional.Of(corev1.EnvVar{
					Name: "INSTANA_DOWNLOAD_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: "riuoidfoisd",
							},
							Key:      "downloadKey",
							Optional: pointer.To(true),
						},
					},
				}),
				actual,
			)
		})
	})
}
