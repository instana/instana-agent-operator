package ports

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

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

func TestPortsBuilder_GetServicePorts_GetContainerPorts(t *testing.T) {
	for _, test := range []struct {
		name                   string
		agentSpec              instanav1.InstanaAgentSpec
		requested              []InstanaAgentPort
		expectedServicePorts   []corev1.ServicePort
		expectedContainerPorts []corev1.ContainerPort
	}{
		{
			name:                   "none",
			agentSpec:              instanav1.InstanaAgentSpec{},
			requested:              []InstanaAgentPort{},
			expectedServicePorts:   []corev1.ServicePort{},
			expectedContainerPorts: []corev1.ContainerPort{},
		},
		{
			name:      "some",
			agentSpec: instanav1.InstanaAgentSpec{OpenTelemetry: instanav1.OpenTelemetry{GRPC: &instanav1.Enabled{}}},
			requested: []InstanaAgentPort{AgentAPIsPort, OpenTelemetryGRPCPort, OpenTelemetryHTTPPort},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       string(AgentAPIsPort),
					TargetPort: intstr.FromString(string(AgentAPIsPort)),
					Port:       42699,
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       string(OpenTelemetryGRPCPort),
					TargetPort: intstr.FromString(string(OpenTelemetryGRPCPort)),
					Port:       4317,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          string(AgentAPIsPort),
					ContainerPort: 42699,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          string(OpenTelemetryGRPCPort),
					ContainerPort: 4317,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
		{
			name: "all",
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					GRPC: &instanav1.Enabled{},
					HTTP: &instanav1.Enabled{},
				},
			},
			requested: []InstanaAgentPort{
				AgentAPIsPort,
				AgentSocketPort,
				OpenTelemetryLegacyPort,
				OpenTelemetryGRPCPort,
				OpenTelemetryHTTPPort,
			},
			expectedServicePorts: []corev1.ServicePort{
				{
					Name:       string(AgentAPIsPort),
					TargetPort: intstr.FromString(string(AgentAPIsPort)),
					Port:       42699,
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       string(AgentSocketPort),
					TargetPort: intstr.FromString(string(AgentSocketPort)),
					Port:       42666,
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       string(OpenTelemetryLegacyPort),
					TargetPort: intstr.FromString(string(OpenTelemetryLegacyPort)),
					Port:       55680,
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       string(OpenTelemetryGRPCPort),
					TargetPort: intstr.FromString(string(OpenTelemetryGRPCPort)),
					Port:       4317,
					Protocol:   corev1.ProtocolTCP,
				},
				{
					Name:       string(OpenTelemetryHTTPPort),
					TargetPort: intstr.FromString(string(OpenTelemetryHTTPPort)),
					Port:       4318,
					Protocol:   corev1.ProtocolTCP,
				},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				{
					Name:          string(AgentAPIsPort),
					ContainerPort: 42699,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          string(AgentSocketPort),
					ContainerPort: 42666,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          string(OpenTelemetryLegacyPort),
					ContainerPort: 55680,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          string(OpenTelemetryGRPCPort),
					ContainerPort: 4317,
					Protocol:      corev1.ProtocolTCP,
				},
				{
					Name:          string(OpenTelemetryHTTPPort),
					ContainerPort: 4318,
					Protocol:      corev1.ProtocolTCP,
				},
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				pb := NewPortsBuilder(&instanav1.InstanaAgent{Spec: test.agentSpec})

				assertions.Equal(test.expectedServicePorts, pb.GetServicePorts(test.requested...))
				assertions.Equal(test.expectedContainerPorts, pb.GetContainerPorts(test.requested...))
			},
		)
	}
}

func TestNewPortsBuilder(t *testing.T) {
	assertions := require.New(t)

	agent := &instanav1.InstanaAgent{}

	pb := NewPortsBuilder(agent).(*portsBuilder)

	assertions.Same(agent, pb.InstanaAgent)
}
