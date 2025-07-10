/*
(c) Copyright IBM Corp. 2024, 2025
*/

package ports

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var InstanaAgentAPIPortConfig = InstanaAgentPortConfig{
	Name: "agent-apis",
	Port: 42699,
}

var OpenTelemetryLegacyPortConfig = InstanaAgentPortConfig{
	Name: "otlp-legacy",
	Port: 55680,
}

var DefaultOpenTelemetryGRPCPortConfig = InstanaAgentPortConfig{
	Name: "otlp-grpc",
	Port: 4317,
}

var DefaultOpenTelemetryHTTPPortConfig = InstanaAgentPortConfig{
	Name: "otlp-http",
	Port: 4318,
}

// InstanaAgentPortConfig is a config that represents the values that are used to create corev1.ServicePorts and corev1.ContainerPorts
type InstanaAgentPortConfig struct {
	Name string
	Port int32
}

// AsServicePort is a simple conversion function from our InstananAgentPortConfig to corev1.ServicePort
func (i *InstanaAgentPortConfig) AsServicePort() corev1.ServicePort {
	return corev1.ServicePort{
		Name:       i.Name,
		Protocol:   corev1.ProtocolTCP,
		Port:       i.Port,
		TargetPort: intstr.FromInt32(i.Port),
	}
}

// AsContainerPort is a simple conversion function from our InstanaAgentPortConfig to corev1.ContainerPort
func (i *InstanaAgentPortConfig) AsContainerPort() corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          i.Name,
		ContainerPort: i.Port,
		Protocol:      corev1.ProtocolTCP,
	}
}
