package env

import (
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

func TestEnvBuilder_agentModeEnv(t *testing.T) {
	getMethod := func(builder *envBuilder) func() optional.Optional[corev1.EnvVar] {
		return builder.agentModeEnv
	}

	testVarMethod(
		t, []varMethodTest{
			{
				name:      "not_provided",
				getMethod: getMethod,
				agent:     &instanav1.InstanaAgent{},
				expected:  optional.Empty[corev1.EnvVar](),
			},
			{
				name:      "provided",
				getMethod: getMethod,
				agent: &instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Mode: instanav1.KUBERNETES,
						},
					},
				},
				expected: optional.Of(
					corev1.EnvVar{
						Name:  "AGENT_MODE_ENV",
						Value: string(instanav1.KUBERNETES),
					},
				),
			},
		},
	)
}
