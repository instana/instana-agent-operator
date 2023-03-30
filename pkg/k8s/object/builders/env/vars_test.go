package env

import (
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"testing"

	"github.com/instana/instana-agent-operator/pkg/collections/list"

	"github.com/golang/mock/gomock"
	"github.com/instana/instana-agent-operator/pkg/pointer"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/stretchr/testify/require"
)

func testCRFieldEnvVar(
	t *testing.T,
	f func(agent *instanav1.InstanaAgent) optional.Optional[corev1.EnvVar],
	agent *instanav1.InstanaAgent,
	expectedName string,
	expectedValue string,
) {
	t.Run("when_empty", func(t *testing.T) {
		assertions := require.New(t)
		actual := f(&instanav1.InstanaAgent{})

		assertions.Empty(actual)
	})
	t.Run("with_value", func(t *testing.T) {
		assertions := require.New(t)
		actual := f(agent)

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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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
	testCRFieldEnvVar(
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

func testKeysSecretEnvVar(
	t *testing.T,
	f func(helpers helpers.Helpers) optional.Optional[corev1.EnvVar],
	expectedName string,
	expectedKey string,
	expectedOptional *bool,
) {
	ctrl := gomock.NewController(t)
	assertions := require.New(t)

	hlprs := NewMockHelpers(ctrl)
	hlprs.EXPECT().KeysSecretName().Return("weoijsdfjjsf")

	assertions.Equal(
		optional.Of(corev1.EnvVar{
			Name: expectedName,
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: "weoijsdfjjsf",
					},
					Key:      expectedKey,
					Optional: expectedOptional,
				},
			},
		}),
		f(hlprs),
	)
}

func TestAgentKeyEnv(t *testing.T) {
	testKeysSecretEnvVar(t, AgentKeyEnv, "INSTANA_AGENT_KEY", "key", nil)
}

func TestDownloadKeyEnv(t *testing.T) {
	testKeysSecretEnvVar(t, DownloadKeyEnv, "INSTANA_DOWNLOAD_KEY", "downloadKey", pointer.To(true))
}

func testFromPodField(t *testing.T, f func() optional.Optional[corev1.EnvVar], literal corev1.EnvVar) {
	assertions := require.New(t)

	assertions.Equal(optional.Of(literal), f())
}

func TestPodNameEnv(t *testing.T) {
	testFromPodField(
		t,
		PodNameEnv,
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

func TestPodIpEnv(t *testing.T) {
	testFromPodField(t, PodIpEnv, corev1.EnvVar{
		Name: "POD_IP",
		ValueFrom: &corev1.EnvVarSource{
			FieldRef: &corev1.ObjectFieldSelector{
				FieldPath: "status.podIP",
			},
		},
	})
}

func TestUserProvidedEnv(t *testing.T) {
	assertions := require.New(t)

	opts := UserProvidedEnv(&instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Env: map[string]string{
					"foo":      "bar",
					"hello":    "world",
					"oijrgoij": "45ioiojdij",
				},
			},
		},
	})
	actual := list.NewListMapTo[optional.Optional[corev1.EnvVar], corev1.EnvVar]().MapTo(opts, func(builder optional.Optional[corev1.EnvVar]) corev1.EnvVar {
		return builder.Get()
	})

	assertions.ElementsMatch(
		[]corev1.EnvVar{
			{
				Name:  "foo",
				Value: "bar",
			},
			{
				Name:  "hello",
				Value: "world",
			},
			{
				Name:  "oijrgoij",
				Value: "45ioiojdij",
			},
		},
		actual,
	)
}
