package env

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func testEnvFromField(
	name string,
	expectedValue string,
	agent *instanav1.InstanaAgent,
	f func(agent *instanav1.InstanaAgent) EnvBuilder,
) (string, func(t *testing.T)) {
	return name, func(t *testing.T) {
		t.Run("when_empty", func(t *testing.T) {
			assertions := require.New(t)

			actual := f(&instanav1.InstanaAgent{}).Build()

			assertions.Equal(optional.Empty[corev1.EnvVar](), actual)
		})
		t.Run("when_filled", func(t *testing.T) {
			assertions := require.New(t)

			actual := f(agent).Build()

			assertions.Equal(
				optional.Of(corev1.EnvVar{
					Name:  name,
					Value: expectedValue,
				}),
				actual,
			)
		})
	}
}

func TestEnvBuilders(t *testing.T) {
	t.Run(testEnvFromField(
		"INSTANA_AGENT_MODE",
		string(instanav1.APM),
		&instanav1.InstanaAgent{
			Spec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Mode: instanav1.APM,
				},
			},
		},
		AgentModeEnv,
	))
}
