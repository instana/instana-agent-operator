package ports

import (
	"errors"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

type InstanaAgentPort string

const (
	AgentAPIsPort           InstanaAgentPort = "agent-apis"
	AgentSocketPort         InstanaAgentPort = "agent-socket"
	OpenTelemetryLegacyPort InstanaAgentPort = "opentelemetry-legacy"
	OpenTelemetryGRPCPort   InstanaAgentPort = "opentelemetry-grpc"
	OpenTelemetryHTTPPort   InstanaAgentPort = "opentelemetry-http"
)

func (p InstanaAgentPort) portNumber() int32 {
	switch p {
	case AgentAPIsPort:
		return 42699
	case AgentSocketPort:
		return 42666
	case OpenTelemetryLegacyPort:
		return 55680
	case OpenTelemetryGRPCPort:
		return 4317
	case OpenTelemetryHTTPPort:
		return 4318
	default:
		panic(errors.New("unknown port requested"))
	}
}

func (p InstanaAgentPort) isEnabled(agent *instanav1.InstanaAgent) bool {
	switch p {
	case OpenTelemetryLegacyPort:
		fallthrough
	case OpenTelemetryGRPCPort:
		return agent.Spec.OpenTelemetry.GrpcIsEnabled()
	case OpenTelemetryHTTPPort:
		return agent.Spec.OpenTelemetry.HttpIsEnabled()
	default:
		return true
	}
}

type PortsBuilder interface {
	GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort
	GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort
}

type portsBuilder struct {
	*instanav1.InstanaAgent
}

func (p *portsBuilder) GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort {
	enabledPorts := list.NewListFilter[InstanaAgentPort]().Filter(
		ports, func(port InstanaAgentPort) bool {
			return port.isEnabled(p.InstanaAgent)
		},
	)

	return list.NewListMapTo[InstanaAgentPort, corev1.ServicePort]().MapTo(
		enabledPorts,
		func(port InstanaAgentPort) corev1.ServicePort {
			return corev1.ServicePort{
				Name:       string(port),
				Protocol:   corev1.ProtocolTCP,
				Port:       port.portNumber(),
				TargetPort: intstr.FromString(string(port)),
			}
		},
	)
}

func (p *portsBuilder) GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort {
	return list.NewListMapTo[InstanaAgentPort, corev1.ContainerPort]().MapTo(
		ports,
		func(port InstanaAgentPort) corev1.ContainerPort {
			return corev1.ContainerPort{
				Name:          string(port),
				ContainerPort: port.portNumber(),
				Protocol:      corev1.ProtocolTCP,
			}
		},
	)
}

func NewPortsBuilder(agent *instanav1.InstanaAgent) PortsBuilder {
	return &portsBuilder{
		InstanaAgent: agent,
	}
}
