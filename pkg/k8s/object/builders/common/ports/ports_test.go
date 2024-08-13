/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package ports_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
)

func TestInstanaAgentPortMappings(t *testing.T) {
	enabled := true
	disabled := false

	for _, test := range []struct {
		name                  string
		port                  ports.InstanaAgentPort
		openTelemetrySettings instanav1.OpenTelemetry
		expectedPortNumber    int32
		expectEnabled         bool
		expectPanic           bool
	}{
		{
			name:                  string(ports.AgentAPIsPort),
			port:                  ports.AgentAPIsPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &enabled}},
			expectedPortNumber:    ports.AgentAPIsPortNumber,
			expectEnabled:         true,
		},

		{
			name:                  string(ports.OpenTelemetryLegacyPort) + "_not_enabled",
			port:                  ports.OpenTelemetryLegacyPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &disabled}},
			expectedPortNumber:    ports.OpenTelemetryLegacyPortNumber,
			expectEnabled:         false,
		},
		{
			name:                  string(ports.OpenTelemetryLegacyPort) + "_enabled",
			port:                  ports.OpenTelemetryLegacyPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &enabled}},
			expectedPortNumber:    ports.OpenTelemetryLegacyPortNumber,
			expectEnabled:         true,
		},

		{
			name:                  string(ports.OpenTelemetryGRPCPort) + "_not_enabled",
			port:                  ports.OpenTelemetryGRPCPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &disabled}},
			expectedPortNumber:    ports.OpenTelemetryGRPCPortNumber,
			expectEnabled:         false,
		},
		{
			name:                  string(ports.OpenTelemetryGRPCPort) + "_enabled",
			port:                  ports.OpenTelemetryGRPCPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &enabled}},
			expectedPortNumber:    ports.OpenTelemetryGRPCPortNumber,
			expectEnabled:         true,
		},

		{
			name:                  string(ports.OpenTelemetryHTTPPort) + "_not_enabled",
			port:                  ports.OpenTelemetryHTTPPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, HTTP: &instanav1.Enabled{Enabled: &disabled}},
			expectedPortNumber:    ports.OpenTelemetryHTTPPortNumber,
			expectEnabled:         false,
		},
		{
			name:                  string(ports.OpenTelemetryHTTPPort) + "_enabled",
			port:                  ports.OpenTelemetryHTTPPort,
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, HTTP: &instanav1.Enabled{Enabled: &enabled}},
			expectedPortNumber:    ports.OpenTelemetryHTTPPortNumber,
			expectEnabled:         true,
		},
		{
			name:                  "unknown_port",
			port:                  ports.InstanaAgentPort("unknown"),
			openTelemetrySettings: instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}},
			expectEnabled:         true,
			expectPanic:           true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				assertions.Equal(string(test.port), test.port.String())
				assertions.Equal(test.expectEnabled, test.port.IsEnabled(test.openTelemetrySettings))

				if test.expectPanic {
					assertions.PanicsWithError(
						"unknown port requested", func() {
							test.port.PortNumber()
						},
					)
				} else {
					assertions.Equal(test.expectedPortNumber, test.port.PortNumber())
				}
			},
		)
	}
}

func TestPortsBuilderGetPorts(t *testing.T) {
	enabled := true
	disabled := false

	containerPortAgentAPI := corev1.ContainerPort{
		Name:          string(ports.AgentAPIsPort),
		ContainerPort: ports.AgentAPIsPortNumber,
		Protocol:      corev1.ProtocolTCP,
	}

	containerPortOtelGPRC := corev1.ContainerPort{
		Name:          string(ports.OpenTelemetryGRPCPort),
		ContainerPort: ports.OpenTelemetryGRPCPortNumber,
		Protocol:      corev1.ProtocolTCP,
	}

	containerPortOtelHTTP := corev1.ContainerPort{
		Name:          string(ports.OpenTelemetryHTTPPort),
		ContainerPort: ports.OpenTelemetryHTTPPortNumber,
		Protocol:      corev1.ProtocolTCP,
	}

	containerPortOtelLegacy := corev1.ContainerPort{
		Name:          string(ports.OpenTelemetryLegacyPort),
		ContainerPort: ports.OpenTelemetryLegacyPortNumber,
		Protocol:      corev1.ProtocolTCP,
	}

	servicePortAgentAPI := corev1.ServicePort{
		Name:       string(ports.AgentAPIsPort),
		Port:       ports.AgentAPIsPortNumber,
		TargetPort: intstr.FromString(string(ports.AgentAPIsPort)),
		Protocol:   corev1.ProtocolTCP,
	}

	servicePortOtelGRPC := corev1.ServicePort{
		Name:       string(ports.OpenTelemetryGRPCPort),
		Port:       ports.OpenTelemetryGRPCPortNumber,
		TargetPort: intstr.FromString(string(ports.OpenTelemetryGRPCPort)),
		Protocol:   corev1.ProtocolTCP,
	}

	servicePortOtelLegacy := corev1.ServicePort{
		Name:       string(ports.OpenTelemetryLegacyPort),
		Port:       ports.OpenTelemetryLegacyPortNumber,
		TargetPort: intstr.FromString(string(ports.OpenTelemetryLegacyPort)),
		Protocol:   corev1.ProtocolTCP,
	}

	servicePortOtelHTTP := corev1.ServicePort{
		Name:       string(ports.OpenTelemetryHTTPPort),
		Port:       ports.OpenTelemetryHTTPPortNumber,
		TargetPort: intstr.FromString(string(ports.OpenTelemetryHTTPPort)),
		Protocol:   corev1.ProtocolTCP,
	}

	for _, test := range []struct {
		name                   string
		openTelemetrySettings  instanav1.OpenTelemetry
		expectedContainerPorts []corev1.ContainerPort
		expectedServicePorts   []corev1.ServicePort
	}{
		{
			name:                   "Enabled all by default",
			openTelemetrySettings:  instanav1.OpenTelemetry{},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI, containerPortOtelGPRC, containerPortOtelHTTP, containerPortOtelLegacy},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI, servicePortOtelGRPC, servicePortOtelHTTP, servicePortOtelLegacy},
		},
		{
			name:                   "Enabled all with opt-in",
			openTelemetrySettings:  instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &enabled}, HTTP: &instanav1.Enabled{Enabled: &enabled}},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI, containerPortOtelGPRC, containerPortOtelHTTP, containerPortOtelLegacy},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI, servicePortOtelGRPC, servicePortOtelHTTP, servicePortOtelLegacy},
		},
		{
			name:                   "Enabled only GPRC",
			openTelemetrySettings:  instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &enabled}, HTTP: &instanav1.Enabled{Enabled: &disabled}},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI, containerPortOtelGPRC, containerPortOtelLegacy},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI, servicePortOtelGRPC, servicePortOtelLegacy},
		},
		{
			name:                   "Enabled only HTTP",
			openTelemetrySettings:  instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &enabled}, GRPC: &instanav1.Enabled{Enabled: &disabled}, HTTP: &instanav1.Enabled{Enabled: &enabled}},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI, containerPortOtelHTTP},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI, servicePortOtelHTTP},
		},
		{
			name:                   "Disable all OLTP ports without legacy setting",
			openTelemetrySettings:  instanav1.OpenTelemetry{GRPC: &instanav1.Enabled{Enabled: &disabled}, HTTP: &instanav1.Enabled{Enabled: &disabled}},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI},
		},
		{
			name:                   "Disable all OLTP ports via legacy setting",
			openTelemetrySettings:  instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &disabled}},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI},
		},
		{
			name:                   "Conflicting supported and legacy settings",
			openTelemetrySettings:  instanav1.OpenTelemetry{Enabled: instanav1.Enabled{Enabled: &disabled}, GRPC: &instanav1.Enabled{Enabled: &enabled}, HTTP: &instanav1.Enabled{Enabled: &enabled}},
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI, containerPortOtelGPRC, containerPortOtelHTTP, containerPortOtelLegacy},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI, servicePortOtelGRPC, servicePortOtelHTTP, servicePortOtelLegacy},
		},
	} {
		assertions := require.New(t)
		actualContainerPorts := ports.
			NewPortsBuilder(&instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					OpenTelemetry: test.openTelemetrySettings,
				},
			}).
			GetContainerPorts(
				ports.AgentAPIsPort,
				ports.OpenTelemetryGRPCPort,
				ports.OpenTelemetryHTTPPort,
				ports.OpenTelemetryLegacyPort,
			)

		actualServicePorts := ports.
			NewPortsBuilder(&instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					OpenTelemetry: test.openTelemetrySettings,
				},
			}).
			GetServicePorts(
				ports.AgentAPIsPort,
				ports.OpenTelemetryGRPCPort,
				ports.OpenTelemetryHTTPPort,
				ports.OpenTelemetryLegacyPort,
			)

		assertions.Equal(test.expectedContainerPorts, actualContainerPorts, test.openTelemetrySettings)
		assertions.Equal(test.expectedServicePorts, actualServicePorts)
	}
}
