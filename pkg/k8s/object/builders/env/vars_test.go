package env

import (
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type varMethodTest struct {
	name      string
	getMethod func(builder *envBuilder) func() optional.Optional[corev1.EnvVar]
	agent     *instanav1.InstanaAgent
	expected  optional.Optional[corev1.EnvVar]
}

func testVarMethod(t *testing.T, tests []varMethodTest) {
	for _, test := range tests {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				builder := NewEnvBuilder(test.agent).(*envBuilder)
				method := test.getMethod(builder)
				actual := method()

				assertions.Equal(test.expected, actual)
			},
		)
	}
}

type crFieldVarTest struct {
	t             *testing.T
	getMethod     func(builder *envBuilder) func() optional.Optional[corev1.EnvVar]
	expectedName  string
	expectedValue string
	agentSpec     instanav1.InstanaAgentSpec
}

func testFromCRField(test *crFieldVarTest) {
	testVarMethod(
		test.t, []varMethodTest{
			{
				name:      "not_provided",
				getMethod: test.getMethod,
				agent:     &instanav1.InstanaAgent{},
				expected:  optional.Empty[corev1.EnvVar](),
			},
			{
				name:      "provided",
				getMethod: test.getMethod,
				agent:     &instanav1.InstanaAgent{Spec: test.agentSpec},
				expected: optional.Of(
					corev1.EnvVar{
						Name:  test.expectedName,
						Value: test.expectedValue,
					},
				),
			},
		},
	)
}

func TestEnvBuilder_agentModeEnv(t *testing.T) {
	const expectedValue = instanav1.KUBERNETES

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.agentModeEnv
			},
			expectedName:  "INSTANA_AGENT_MODE",
			expectedValue: string(expectedValue),
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Mode: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_zoneNameEnv(t *testing.T) {
	const expectedValue = "some-zone"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.zoneNameEnv
			},
			expectedName:  "INSTANA_ZONE",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Zone: instanav1.Name{
					Name: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_clusterNameEnv(t *testing.T) {
	const expectedValue = "some-cluster"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.clusterNameEnv
			},
			expectedName:  "INSTANA_KUBERNETES_CLUSTER_NAME",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Cluster: instanav1.Name{
					Name: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_agentEndpointEnv(t *testing.T) {
	const expectedValue = "some-agent-endpoint"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.agentEndpointEnv
			},
			expectedName:  "INSTANA_AGENT_ENDPOINT",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					EndpointHost: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_agentEndpointPortEnv(t *testing.T) {
	const expectedValue = "12345"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.agentEndpointPortEnv
			},
			expectedName:  "INSTANA_AGENT_ENDPOINT_PORT",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					EndpointPort: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_mavenRepoURLEnv(t *testing.T) {
	const expectedValue = "https://repo.maven.apache.org/maven2"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.mavenRepoURLEnv
			},
			expectedName:  "INSTANA_MVN_REPOSITORY_URL",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					MvnRepoUrl: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_proxyHostEnv(t *testing.T) {
	const expectedValue = "some-proxy-host"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.proxyHostEnv
			},
			expectedName:  "INSTANA_AGENT_PROXY_HOST",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyHost: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_proxyPortEnv(t *testing.T) {
	const expectedValue = "8888"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.proxyPortEnv
			},
			expectedName:  "INSTANA_AGENT_PROXY_PORT",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyPort: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_proxyProtocolEnv(t *testing.T) {
	const expectedValue = "http"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.proxyProtocolEnv
			},
			expectedName:  "INSTANA_AGENT_PROXY_PROTOCOL",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyProtocol: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_proxyUserEnv(t *testing.T) {
	const expectedValue = "some-proxy-user"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.proxyUserEnv
			},
			expectedName:  "INSTANA_AGENT_PROXY_USER",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyUser: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_proxyPasswordEnv(t *testing.T) {
	const expectedValue = "some-proxy-password"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.proxyPasswordEnv
			},
			expectedName:  "INSTANA_AGENT_PROXY_PASSWORD",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyPassword: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_proxyUseDNSEnv(t *testing.T) {
	const expectedValue = true

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.proxyUseDNSEnv
			},
			expectedName:  "INSTANA_AGENT_PROXY_USE_DNS",
			expectedValue: strconv.FormatBool(expectedValue),
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ProxyUseDNS: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_listenAddressEnv(t *testing.T) {
	const expectedValue = "0.0.0.0:42699"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.listenAddressEnv
			},
			expectedName:  "INSTANA_AGENT_HTTP_LISTEN",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ListenAddress: expectedValue,
				},
			},
		},
	)
}

func TestEnvBuilder_redactK8sSecretsEnv(t *testing.T) {
	const expectedValue = "true"

	testFromCRField(
		&crFieldVarTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.redactK8sSecretsEnv
			},
			expectedName:  "INSTANA_KUBERNETES_REDACT_SECRETS",
			expectedValue: expectedValue,
			agentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					RedactKubernetesSecrets: expectedValue,
				},
			},
		},
	)
}
