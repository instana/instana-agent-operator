package ports

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestPortMappings(t *testing.T) {
	for _, test := range []struct {
		name               string
		port               InstanaAgentPort
		agentSpec          instanav1.InstanaAgentSpec
		expectedPortNumber int32
		expectEnabled      bool
		expectPanic        bool
	}{
		{
			name:               string(AgentAPIsPort),
			port:               AgentAPIsPort,
			agentSpec:          instanav1.InstanaAgentSpec{},
			expectedPortNumber: 42699,
			expectEnabled:      true,
		},

		{
			name:               string(AgentSocketPort),
			port:               AgentSocketPort,
			agentSpec:          instanav1.InstanaAgentSpec{},
			expectedPortNumber: 42666,
			expectEnabled:      true,
		},

		{
			name:               string(OpenTelemetryLegacyPort) + "_not_enabled",
			port:               OpenTelemetryLegacyPort,
			agentSpec:          instanav1.InstanaAgentSpec{},
			expectedPortNumber: 55680,
			expectEnabled:      false,
		},
		{
			name: string(OpenTelemetryLegacyPort) + "_enabled",
			port: OpenTelemetryLegacyPort,
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					GRPC: &instanav1.Enabled{},
				},
			},
			expectedPortNumber: 55680,
			expectEnabled:      true,
		},

		{
			name:               string(OpenTelemetryGRPCPort) + "_not_enabled",
			port:               OpenTelemetryGRPCPort,
			agentSpec:          instanav1.InstanaAgentSpec{},
			expectedPortNumber: 4317,
			expectEnabled:      false,
		},
		{
			name: string(OpenTelemetryGRPCPort) + "_enabled",
			port: OpenTelemetryGRPCPort,
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					GRPC: &instanav1.Enabled{},
				},
			},
			expectedPortNumber: 4317,
			expectEnabled:      true,
		},

		{
			name:               string(OpenTelemetryHTTPPort) + "_not_enabled",
			port:               OpenTelemetryHTTPPort,
			agentSpec:          instanav1.InstanaAgentSpec{},
			expectedPortNumber: 4318,
			expectEnabled:      false,
		},
		{
			name: string(OpenTelemetryHTTPPort) + "_enabled",
			port: OpenTelemetryHTTPPort,
			agentSpec: instanav1.InstanaAgentSpec{
				OpenTelemetry: instanav1.OpenTelemetry{
					HTTP: &instanav1.Enabled{},
				},
			},
			expectedPortNumber: 4318,
			expectEnabled:      true,
		},
		{
			name:          "unknown_port",
			port:          InstanaAgentPort("unknown"),
			agentSpec:     instanav1.InstanaAgentSpec{},
			expectEnabled: true,
			expectPanic:   true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				agent := &instanav1.InstanaAgent{
					Spec: test.agentSpec,
				}

				assertions.Equal(test.expectEnabled, test.port.isEnabled(agent))

				if test.expectPanic {
					assertions.PanicsWithError(
						"unknown port requested", func() {
							test.port.portNumber()
						},
					)
				} else {
					assertions.Equal(test.expectedPortNumber, test.port.portNumber())
				}
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
				{
					Name:          string(OpenTelemetryHTTPPort),
					ContainerPort: 4318,
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
		{
			name:      "all_optionals_disabled",
			agentSpec: instanav1.InstanaAgentSpec{},
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
