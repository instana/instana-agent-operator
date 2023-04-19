package ports

import (
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Test + add to services and daemonset
// TODO: Possibly refactor EnvVars (and volumes?) to be more like this?

type InstanaAgentPort string

const (
	AgentAPIsPort           InstanaAgentPort = "agent-apis"
	AgentSocketPort         InstanaAgentPort = "agent-socket"
	OpenTelemetryLegacyPort InstanaAgentPort = "opentelemetry-legacy"
	OpenTelemetryGRPCPort   InstanaAgentPort = "opentelemetry-grpc"
	OpenTelemetryHTTPPort   InstanaAgentPort = "opentelemetry-http"
)

type portBuilder func(agent *instanav1.InstanaAgent) optional.Optional[int32]

var (
	portMappings = map[InstanaAgentPort]portBuilder{
		AgentAPIsPort: func(_ *instanav1.InstanaAgent) optional.Optional[int32] {
			return optional.Of[int32](42699)
		},
		AgentSocketPort: func(_ *instanav1.InstanaAgent) optional.Optional[int32] {
			return optional.Of[int32](42666)
		},
		OpenTelemetryLegacyPort: func(agent *instanav1.InstanaAgent) optional.Optional[int32] {
			switch agent.Spec.OpenTelemetry.GrpcIsEnabled() {
			case true:
				return optional.Of[int32](55680)
			default:
				return optional.Empty[int32]()
			}
		},
		OpenTelemetryGRPCPort: func(agent *instanav1.InstanaAgent) optional.Optional[int32] {
			switch agent.Spec.OpenTelemetry.GrpcIsEnabled() {
			case true:
				return optional.Of[int32](4317)
			default:
				return optional.Empty[int32]()
			}
		},
		OpenTelemetryHTTPPort: func(agent *instanav1.InstanaAgent) optional.Optional[int32] {
			switch agent.Spec.OpenTelemetry.HttpIsEnabled() {
			case true:
				return optional.Of[int32](4318)
			default:
				return optional.Empty[int32]()
			}
		},
	}
)

type portHolder struct {
	name InstanaAgentPort
	port int32
}

type PortsBuilder interface {
	GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort
	GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort
}

type portsBuilder struct {
	*instanav1.InstanaAgent
}

func (p *portsBuilder) getDesiredPorts(ports ...InstanaAgentPort) []portHolder {
	res := make([]portHolder, 0, len(ports))

	for _, name := range ports {
		portMappings[name](p.InstanaAgent).IfPresent(
			func(port int32) {
				res = append(
					res, portHolder{
						name: name,
						port: port,
					},
				)
			},
		)
	}

	return res
}

func (p *portsBuilder) GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort {
	return list.NewListMapTo[portHolder, corev1.ServicePort]().MapTo(
		p.getDesiredPorts(ports...), func(port portHolder) corev1.ServicePort {
			return corev1.ServicePort{
				Name:       string(port.name),
				Protocol:   corev1.ProtocolTCP,
				Port:       port.port,
				TargetPort: intstr.FromString(string(port.name)),
			}
		},
	)
}

func (p *portsBuilder) GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort {
	return list.NewListMapTo[portHolder, corev1.ContainerPort]().MapTo(
		p.getDesiredPorts(ports...),
		func(port portHolder) corev1.ContainerPort {
			return corev1.ContainerPort{
				Name:          string(port.name),
				ContainerPort: port.port,
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
