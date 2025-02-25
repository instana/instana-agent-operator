/*
(c) Copyright IBM Corp. 2024,2025
*/

package ports

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	corev1 "k8s.io/api/core/v1"
)

type PortsBuilder interface {
	GetServicePorts() []corev1.ServicePort
	GetContainerPorts() []corev1.ContainerPort
}

type portsBuilder struct {
	otel instanav1.OpenTelemetry
}

func NewPortsBuilder(otel instanav1.OpenTelemetry) PortsBuilder {
	return &portsBuilder{otel}
}

// GetServicePorts is responsible of creating a list of service ports based on the InstanaAgent configuration
func (p *portsBuilder) GetServicePorts() []corev1.ServicePort {
	servicePorts := []corev1.ServicePort{InstanaAgentAPIPortConfig.AsServicePort()}
	if *p.otel.Enabled.Enabled != false { //nolint:gosimple
		if *p.otel.GRPC.Enabled {
			GRPCPortConfig := DefaultOpenTelemetryGRPCPortConfig
			// Override default value if the config contains a new port number
			if p.otel.GRPC.Port != nil {
				GRPCPortConfig.Port = *p.otel.GRPC.Port
			}
			servicePorts = append(servicePorts, GRPCPortConfig.AsServicePort(), OpenTelemetryLegacyPortConfig.AsServicePort())
		}
		if *p.otel.HTTP.Enabled {
			HTTPPortConfig := DefaultOpenTelemetryHTTPPortConfig
			// Override default value if the config contains a new port number
			if p.otel.HTTP.Port != nil {
				HTTPPortConfig.Port = *p.otel.HTTP.Port
			}
			servicePorts = append(servicePorts, HTTPPortConfig.AsServicePort())
		}
	}

	return servicePorts
}

// GetContainerPorts is responsible of creating a list of container ports based on the InstanaAgent configuration
func (p *portsBuilder) GetContainerPorts() []corev1.ContainerPort {
	containerPorts := []corev1.ContainerPort{InstanaAgentAPIPortConfig.AsContainerPort()}
	if *p.otel.Enabled.Enabled != false { //nolint:gosimple
		if *p.otel.GRPC.Enabled {
			GRPCPortConfig := DefaultOpenTelemetryGRPCPortConfig
			// Override default value if the config contains a new port number
			if p.otel.GRPC.Port != nil {
				GRPCPortConfig.Port = *p.otel.GRPC.Port
			}
			containerPorts = append(containerPorts, GRPCPortConfig.AsContainerPort(), OpenTelemetryLegacyPortConfig.AsContainerPort())
		}
		if *p.otel.HTTP.Enabled {
			HTTPPortConfig := DefaultOpenTelemetryHTTPPortConfig
			// Override default value if the config contains a new port number
			if p.otel.HTTP.Port != nil {
				HTTPPortConfig.Port = *p.otel.HTTP.Port
			}
			containerPorts = append(containerPorts, HTTPPortConfig.AsContainerPort())
		}
	}

	return containerPorts
}
