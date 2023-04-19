package ports

import (
	"testing"

	"github.com/stretchr/testify/require"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestPortMappings(t *testing.T) {
	for _, test := range []struct {
		name      string
		portName  InstanaAgentPort
		agentSpec instanav1.InstanaAgentSpec
		expected  optional.Optional[int32]
	}{
		{
			name:      string(AgentAPIsPort),
			portName:  AgentAPIsPort,
			agentSpec: instanav1.InstanaAgentSpec{},
			expected:  optional.Of[int32](42699),
		},

		{
			name:      string(AgentSocketPort),
			portName:  AgentSocketPort,
			agentSpec: instanav1.InstanaAgentSpec{},
			expected:  optional.Of[int32](42666),
		},

		{
			name:      string(OpenTelemetryLegacyPort) + "_not_enabled",
			portName:  OpenTelemetryLegacyPort,
			agentSpec: instanav1.InstanaAgentSpec{},
			expected:  optional.Empty[int32](),
		},
		{
			name:     string(OpenTelemetryLegacyPort) + "_enabled",
			portName: OpenTelemetryLegacyPort,
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					GRPC: &instanav1.Enabled{},
				},
			},
			expected: optional.Of[int32](55680),
		},

		{
			name:      string(OpenTelemetryGRPCPort) + "_not_enabled",
			portName:  OpenTelemetryGRPCPort,
			agentSpec: instanav1.InstanaAgentSpec{},
			expected:  optional.Empty[int32](),
		},
		{
			name:     string(OpenTelemetryGRPCPort) + "_enabled",
			portName: OpenTelemetryGRPCPort,
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					GRPC: &instanav1.Enabled{},
				},
			},
			expected: optional.Of[int32](4317),
		},

		{
			name:      string(OpenTelemetryHTTPPort) + "_not_enabled",
			portName:  OpenTelemetryHTTPPort,
			agentSpec: instanav1.InstanaAgentSpec{},
			expected:  optional.Empty[int32](),
		},
		{
			name:     string(OpenTelemetryHTTPPort) + "_enabled",
			portName: OpenTelemetryHTTPPort,
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					HTTP: &instanav1.Enabled{},
				},
			},
			expected: optional.Of[int32](4318),
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				agent := &instanav1.InstanaAgent{
					Spec: test.agentSpec,
				}

				actual := portMappings[test.portName](agent)

				assertions.Equal(test.expected, actual)
			},
		)
	}
}
