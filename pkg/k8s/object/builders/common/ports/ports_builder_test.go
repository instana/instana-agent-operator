/*
(c) Copyright IBM Corp. 2024,2025
*/

package ports_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestPortsBuilderGetPorts(t *testing.T) {
	for _, test := range []struct {
		name                   string
		openTelemetrySettings  instanav1.OpenTelemetry
		expectedContainerPorts []corev1.ContainerPort
		expectedServicePorts   []corev1.ServicePort
	}{
		{
			name: "Enabled all with opt-in",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: pointer.To(true)},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(true)},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(true)},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
				ports.DefaultOpenTelemetryGRPCPortConfig.AsContainerPort(),
				ports.OpenTelemetryLegacyPortConfig.AsContainerPort(),
				ports.DefaultOpenTelemetryHTTPPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
				ports.DefaultOpenTelemetryGRPCPortConfig.AsServicePort(),
				ports.OpenTelemetryLegacyPortConfig.AsServicePort(),
				ports.DefaultOpenTelemetryHTTPPortConfig.AsServicePort(),
			},
		},
		{
			name: "Enabled only GPRC",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: pointer.To(true)},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(true)},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(false)},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
				ports.DefaultOpenTelemetryGRPCPortConfig.AsContainerPort(),
				ports.OpenTelemetryLegacyPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
				ports.DefaultOpenTelemetryGRPCPortConfig.AsServicePort(),
				ports.OpenTelemetryLegacyPortConfig.AsServicePort(),
			},
		},
		{
			name: "Enabled only HTTP",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: pointer.To(true)},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(false)},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(true)},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
				ports.DefaultOpenTelemetryHTTPPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
				ports.DefaultOpenTelemetryHTTPPortConfig.AsServicePort(),
			},
		},
		{
			name: "Disable all OLTP ports without legacy setting",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(false)},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(false)},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
			},
		},
		{
			name: "Disable all OLTP ports via legacy setting",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
			},
		},
		{
			name: "Conflicting supported and legacy settings",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(true)},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(true)},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
			},
		},
	} {
		assertions := require.New(t)
		portsBuilder := ports.NewPortsBuilder(test.openTelemetrySettings)
		assertions.Equal(test.expectedContainerPorts, portsBuilder.GetContainerPorts(), "Occurred in: "+test.name)
		assertions.Equal(test.expectedServicePorts, portsBuilder.GetServicePorts(), "Occurred in: "+test.name)
	}
}

func TestPortsBuilderOverridePorts(t *testing.T) {
	enabled := true
	GRPCPort := int32(1234)
	GRPCPortConfig := ports.DefaultOpenTelemetryGRPCPortConfig
	GRPCPortConfig.Port = GRPCPort

	HTTPPort := int32(4567)
	HTTPPortConfig := ports.DefaultOpenTelemetryHTTPPortConfig
	HTTPPortConfig.Port = HTTPPort

	for _, test := range []struct {
		name                   string
		openTelemetrySettings  instanav1.OpenTelemetry
		expectedContainerPorts []corev1.ContainerPort
		expectedServicePorts   []corev1.ServicePort
	}{
		{
			name: "Both GRPC and HTTP ports are overwritten with another port number specified in OpenTelemetry",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: &enabled},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: &enabled, Port: &GRPCPort},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: &enabled, Port: &HTTPPort},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
				GRPCPortConfig.AsContainerPort(),
				ports.OpenTelemetryLegacyPortConfig.AsContainerPort(),
				HTTPPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
				GRPCPortConfig.AsServicePort(),
				ports.OpenTelemetryLegacyPortConfig.AsServicePort(),
				HTTPPortConfig.AsServicePort(),
			},
		},
		{
			name: "Only HTTP port is overwritten with another port number specified in OpenTelemetry",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: &enabled},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: &enabled},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: &enabled, Port: &HTTPPort},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
				ports.DefaultOpenTelemetryGRPCPortConfig.AsContainerPort(),
				ports.OpenTelemetryLegacyPortConfig.AsContainerPort(),
				HTTPPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
				ports.DefaultOpenTelemetryGRPCPortConfig.AsServicePort(),
				ports.OpenTelemetryLegacyPortConfig.AsServicePort(),
				HTTPPortConfig.AsServicePort(),
			},
		},
		{
			name: "Only GRPC port is overwritten with another port number specified in OpenTelemetry",
			openTelemetrySettings: instanav1.OpenTelemetry{
				Enabled: instanav1.Enabled{Enabled: &enabled},
				GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: &enabled, Port: &GRPCPort},
				HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: &enabled},
			},
			expectedContainerPorts: []corev1.ContainerPort{
				ports.InstanaAgentAPIPortConfig.AsContainerPort(),
				GRPCPortConfig.AsContainerPort(),
				ports.OpenTelemetryLegacyPortConfig.AsContainerPort(),
				ports.DefaultOpenTelemetryHTTPPortConfig.AsContainerPort(),
			},
			expectedServicePorts: []corev1.ServicePort{
				ports.InstanaAgentAPIPortConfig.AsServicePort(),
				GRPCPortConfig.AsServicePort(),
				ports.OpenTelemetryLegacyPortConfig.AsServicePort(),
				ports.DefaultOpenTelemetryHTTPPortConfig.AsServicePort(),
			},
		},
	} {
		assertions := require.New(t)
		portsBuilder := ports.NewPortsBuilder(test.openTelemetrySettings)
		assertions.Equal(test.expectedContainerPorts, portsBuilder.GetContainerPorts(), "Occurred in: "+test.name)
		assertions.Equal(test.expectedServicePorts, portsBuilder.GetServicePorts(), "Occurred in: "+test.name)
	}
}
