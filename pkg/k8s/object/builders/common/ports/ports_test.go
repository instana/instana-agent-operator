package ports

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestPortMappings(t *testing.T) {
	for _, test := range []struct {
		name                   string
		port                   InstanaAgentPort
		otlpSettingsConditions func(openTelemetrySettings *MockOpenTelemetrySettings)
		expectedPortNumber     int32
		expectEnabled          bool
		expectPanic            bool
	}{
		{
			name:                   string(AgentAPIsPort),
			port:                   AgentAPIsPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {},
			expectedPortNumber:     42699,
			expectEnabled:          true,
		},

		{
			name:                   string(AgentSocketPort),
			port:                   AgentSocketPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {},
			expectedPortNumber:     42666,
			expectEnabled:          true,
		},

		{
			name: string(OpenTelemetryLegacyPort) + "_not_enabled",
			port: OpenTelemetryLegacyPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {
				openTelemetrySettings.EXPECT().GrpcIsEnabled().Return(false)
			},
			expectedPortNumber: 55680,
			expectEnabled:      false,
		},
		{
			name: string(OpenTelemetryLegacyPort) + "_enabled",
			port: OpenTelemetryLegacyPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {
				openTelemetrySettings.EXPECT().GrpcIsEnabled().Return(true)
			},
			expectedPortNumber: 55680,
			expectEnabled:      true,
		},

		{
			name: string(OpenTelemetryGRPCPort) + "_not_enabled",
			port: OpenTelemetryGRPCPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {
				openTelemetrySettings.EXPECT().GrpcIsEnabled().Return(false)
			},
			expectedPortNumber: 4317,
			expectEnabled:      false,
		},
		{
			name: string(OpenTelemetryGRPCPort) + "_enabled",
			port: OpenTelemetryGRPCPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {
				openTelemetrySettings.EXPECT().GrpcIsEnabled().Return(true)
			},
			expectedPortNumber: 4317,
			expectEnabled:      true,
		},

		{
			name: string(OpenTelemetryHTTPPort) + "_not_enabled",
			port: OpenTelemetryHTTPPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {
				openTelemetrySettings.EXPECT().HttpIsEnabled().Return(false)
			}, expectedPortNumber: 4318,
			expectEnabled: false,
		},
		{
			name: string(OpenTelemetryHTTPPort) + "_enabled",
			port: OpenTelemetryHTTPPort,
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {
				openTelemetrySettings.EXPECT().HttpIsEnabled().Return(true)
			},
			expectedPortNumber: 4318,
			expectEnabled:      true,
		},
		{
			name:                   "unknown_port",
			port:                   InstanaAgentPort("unknown"),
			otlpSettingsConditions: func(openTelemetrySettings *MockOpenTelemetrySettings) {},
			expectEnabled:          true,
			expectPanic:            true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				openTelemetrySettings := NewMockOpenTelemetrySettings(ctrl)
				test.otlpSettingsConditions(openTelemetrySettings)

				assertions.Equal(string(test.port), test.port.String())

				assertions.Equal(test.expectEnabled, test.port.isEnabled(openTelemetrySettings))

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

func Test_toServicePort(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	p := NewMockPort(ctrl)
	p.EXPECT().String().Return("roijdoijsglkjsdf").Times(2)
	p.EXPECT().portNumber().Return(int32(98798))

	expected := corev1.ServicePort{
		Name:       "roijdoijsglkjsdf",
		Protocol:   corev1.ProtocolTCP,
		Port:       98798,
		TargetPort: intstr.FromString("roijdoijsglkjsdf"),
	}

	actual := toServicePort(p)

	assertions.Equal(expected, actual)
}

func Test_toContainerPort(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	p := NewMockPort(ctrl)
	p.EXPECT().String().Return("roijdoijsglkjsdf")
	p.EXPECT().portNumber().Return(int32(98798))

	expected := corev1.ContainerPort{
		Name:          "roijdoijsglkjsdf",
		ContainerPort: 98798,
		Protocol:      corev1.ProtocolTCP,
	}

	actual := toContainerPort(p)

	assertions.Equal(expected, actual)
}

func TestPortsBuilder_GetServicePorts(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	otlp := instanav1.OpenTelemetry{}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "eoidoijdsg",
		},
		Spec: instanav1.InstanaAgentSpec{
			OpenTelemetry: otlp,
		},
	}

	p1 := NewMockPort(ctrl)
	p1.EXPECT().isEnabled(gomock.Eq(otlp)).Return(true)
	p1.EXPECT().String().Return("p1").Times(2)
	p1.EXPECT().portNumber().Return(int32(1))

	p2 := NewMockPort(ctrl)
	p2.EXPECT().isEnabled(gomock.Eq(otlp)).Return(false)

	p3 := NewMockPort(ctrl)
	p3.EXPECT().isEnabled(gomock.Eq(otlp)).Return(true)
	p3.EXPECT().String().Return("p3").Times(2)
	p3.EXPECT().portNumber().Return(int32(3))

	expected := []corev1.ServicePort{
		{
			Name:       "p1",
			Port:       1,
			TargetPort: intstr.FromString("p1"),
			Protocol:   corev1.ProtocolTCP,
		},
		{
			Name:       "p3",
			Port:       3,
			TargetPort: intstr.FromString("p3"),
			Protocol:   corev1.ProtocolTCP,
		},
	}

	pb := NewPortsBuilder(agent)
	actual := pb.GetServicePorts(p1, p2, p3)

	assertions.Equal(expected, actual)
}

func TestPortsBuilder_GetContainerPorts(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	p1 := NewMockPort(ctrl)
	p1.EXPECT().String().Return("p1")
	p1.EXPECT().portNumber().Return(int32(1))

	p2 := NewMockPort(ctrl)
	p2.EXPECT().String().Return("p2")
	p2.EXPECT().portNumber().Return(int32(2))

	p3 := NewMockPort(ctrl)
	p3.EXPECT().String().Return("p3")
	p3.EXPECT().portNumber().Return(int32(3))

	expected := []corev1.ContainerPort{
		{
			Name:          "p1",
			ContainerPort: 1,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          "p2",
			ContainerPort: 2,
			Protocol:      corev1.ProtocolTCP,
		},
		{
			Name:          "p3",
			ContainerPort: 3,
			Protocol:      corev1.ProtocolTCP,
		},
	}

	actual := NewPortsBuilder(&instanav1.InstanaAgent{}).GetContainerPorts(p1, p2, p3)

	assertions.Equal(expected, actual)
}

func TestNewPortsBuilder(t *testing.T) {
	assertions := require.New(t)

	agent := &instanav1.InstanaAgent{}

	pb := NewPortsBuilder(agent).(*portsBuilder)

	assertions.Same(agent, pb.InstanaAgent)
}
