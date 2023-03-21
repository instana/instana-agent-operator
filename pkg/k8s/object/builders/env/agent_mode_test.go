package env

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/stretchr/testify/require"
)

func TestAgentModeEnv(t *testing.T) {
	assertions := require.New(t)

	actual := AgentModeEnv(&instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Mode: instanav1.KUBERNETES,
			},
		},
	})

	assertions.Equal(
		&fromFieldIfSet[instanav1.AgentMode]{
			name:          "INSTANA_AGENT_MODE",
			providedValue: optional.Of(instanav1.KUBERNETES),
		},
		actual,
	)
}
