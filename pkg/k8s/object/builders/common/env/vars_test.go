package env

import (
	"strconv"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func randString() string {
	return rand.String(rand.IntnRange(1, 15))
}

type varMethodTest struct {
	name          string
	getMethod     func(builder *envBuilder) func() optional.Optional[corev1.EnvVar]
	agent         *instanav1.InstanaAgent
	helpersExpect func(hlprs *MockHelpers)
	expected      optional.Optional[corev1.EnvVar]
}

func testVarMethod(t *testing.T, tests []varMethodTest) {
	for _, test := range tests {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				hlprs := NewMockHelpers(ctrl)
				if helpersExpect := test.helpersExpect; helpersExpect != nil {
					helpersExpect(hlprs)
				}

				builder := &envBuilder{
					agent:   test.agent,
					Helpers: hlprs,
				}
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

type fromSecretTest struct {
	t                  *testing.T
	expectedSecretName string
	expected           optional.Optional[corev1.EnvVar]
	getMethod          func(builder *envBuilder) func() optional.Optional[corev1.EnvVar]
}

func testFromSecretMethod(test *fromSecretTest) {
	testVarMethod(
		test.t, []varMethodTest{
			{
				name:      "build",
				getMethod: test.getMethod,
				agent:     nil,
				helpersExpect: func(hlprs *MockHelpers) {
					hlprs.EXPECT().KeysSecretName().Return(test.expectedSecretName)
				},
				expected: test.expected,
			},
		},
	)
}

func TestEnvBuilder_agentZone(t *testing.T) {
	clusterName := randString()
	zoneName := randString()

	agentZoneMethod := func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
		return builder.agentZoneEnv
	}

	testVarMethod(
		t, []varMethodTest{
			{
				name:          "clusterName_is_set",
				getMethod:     agentZoneMethod,
				helpersExpect: func(hlprs *MockHelpers) {},
				agent: &instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						Cluster: instanav1.Name{
							Name: clusterName,
						},
						Zone: instanav1.Name{
							Name: zoneName,
						},
					},
				},
				expected: optional.Of(
					corev1.EnvVar{
						Name:  "AGENT_ZONE",
						Value: clusterName,
					},
				),
			},

			{
				name:          "clusterName_is_not_set",
				getMethod:     agentZoneMethod,
				helpersExpect: func(hlprs *MockHelpers) {},
				agent: &instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						Zone: instanav1.Name{
							Name: zoneName,
						},
					},
				},
				expected: optional.Of(
					corev1.EnvVar{
						Name:  "AGENT_ZONE",
						Value: zoneName,
					},
				),
			},
		},
	)
}

func TestEnvBuilder_backendURLEnv(t *testing.T) {
	testLiteralAlways(
		&literalAlwaysTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.backendURLEnv
			},
			expected: optional.Of(
				corev1.EnvVar{
					Name:  "BACKEND_URL",
					Value: "https://$(BACKEND)",
				},
			),
		},
	)
}

func TestEnvBuilder_backendEnv(t *testing.T) {
	k8sSensorResourceName := randString()

	testVarMethod(
		t, []varMethodTest{
			{
				name: "from_configMap",
				getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
					return builder.backendEnv
				},
				agent: &instanav1.InstanaAgent{},
				helpersExpect: func(hlprs *MockHelpers) {
					hlprs.EXPECT().K8sSensorResourcesName().Return(k8sSensorResourceName)
				},
				expected: optional.Of(
					corev1.EnvVar{
						Name: "BACKEND",
						ValueFrom: &corev1.EnvVarSource{
							ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
								LocalObjectReference: corev1.LocalObjectReference{
									Name: k8sSensorResourceName,
								},
								Key: constants.BackendKey,
							},
						},
					},
				),
			},
		},
	)
}

func TestEnvBuilder_agentKeyEnv(t *testing.T) {
	const expectedSecretName = "agent-key-secret"

	testFromSecretMethod(
		&fromSecretTest{
			t:                  t,
			expectedSecretName: expectedSecretName,
			expected: optional.Of(
				corev1.EnvVar{
					Name: "INSTANA_AGENT_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: expectedSecretName,
							},
							Key: "key",
						},
					},
				},
			),
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.agentKeyEnv
			},
		},
	)
}

func TestEnvBuilder_downloadKeyEnv(t *testing.T) {
	const expectedSecretName = "download-key-secret"

	testFromSecretMethod(
		&fromSecretTest{
			t:                  t,
			expectedSecretName: expectedSecretName,
			expected: optional.Of(
				corev1.EnvVar{
					Name: "INSTANA_DOWNLOAD_KEY",
					ValueFrom: &corev1.EnvVarSource{
						SecretKeyRef: &corev1.SecretKeySelector{
							LocalObjectReference: corev1.LocalObjectReference{
								Name: expectedSecretName,
							},
							Key:      "downloadKey",
							Optional: pointer.To(true),
						},
					},
				},
			),
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.downloadKeyEnv
			},
		},
	)
}

type literalAlwaysTest struct {
	t         *testing.T
	getMethod func(builder *envBuilder) func() optional.Optional[corev1.EnvVar]
	expected  optional.Optional[corev1.EnvVar]
}

func testLiteralAlways(test *literalAlwaysTest) {
	testVarMethod(
		test.t, []varMethodTest{
			{
				name:      "build",
				getMethod: test.getMethod,
				expected:  test.expected,
			},
		},
	)
}

func TestEnvBuilder_podNameEnv(t *testing.T) {
	testLiteralAlways(
		&literalAlwaysTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.podNameEnv
			},
			expected: optional.Of(
				corev1.EnvVar{
					Name: "INSTANA_AGENT_POD_NAME",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.name",
						},
					},
				},
			),
		},
	)
}

func TestEnvBuilder_podIPEnv(t *testing.T) {
	testLiteralAlways(
		&literalAlwaysTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.podIPEnv
			},
			expected: optional.Of(
				corev1.EnvVar{
					Name: "POD_IP",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "status.podIP",
						},
					},
				},
			),
		},
	)
}

func TestEnvBuilder_podUIDEnv(t *testing.T) {
	testLiteralAlways(
		&literalAlwaysTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.podUIDEnv
			},
			expected: optional.Of(
				corev1.EnvVar{
					Name: "POD_UID",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.uid",
						},
					},
				},
			),
		},
	)
}

func TestEnvBuilder_podNamespaceEnv(t *testing.T) {
	testLiteralAlways(
		&literalAlwaysTest{
			t: t,
			getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
				return builder.podNamespaceEnv
			},
			expected: optional.Of(
				corev1.EnvVar{
					Name: "POD_NAMESPACE",
					ValueFrom: &corev1.EnvVarSource{
						FieldRef: &corev1.ObjectFieldSelector{
							FieldPath: "metadata.namespace",
						},
					},
				},
			),
		},
	)
}

func TestK8sServiceDomainEnv(t *testing.T) {
	testVarMethod(
		t, []varMethodTest{
			{
				name: "build",
				getMethod: func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
					return builder.k8sServiceDomainEnv
				},
				agent: &instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "goijesdlkvlk",
					},
				},
				helpersExpect: func(hlprs *MockHelpers) {
					hlprs.EXPECT().HeadlessServiceName().Return("roidilmsdgo")
				},
				expected: optional.Of(
					corev1.EnvVar{
						Name:  "K8S_SERVICE_DOMAIN",
						Value: "roidilmsdgo.goijesdlkvlk.svc",
					},
				),
			},
		},
	)
}

func TestUserProvidedEnv(t *testing.T) {
	assertions := require.New(t)

	builder := &envBuilder{
		agent: &instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Env: map[string]string{
						"foo":      "bar",
						"hello":    "world",
						"oijrgoij": "45ioiojdij",
					},
				},
			},
		},
	}

	expected := []corev1.EnvVar{
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
	}

	opts := builder.userProvidedEnv()
	actual := list.NewListMapTo[optional.Optional[corev1.EnvVar], corev1.EnvVar]().MapTo(
		opts, func(builder optional.Optional[corev1.EnvVar]) corev1.EnvVar {
			return builder.Get()
		},
	)

	assertions.ElementsMatch(expected, actual)
}
