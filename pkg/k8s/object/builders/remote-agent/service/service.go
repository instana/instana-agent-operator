package service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	componentName = constants.ComponentRemoteAgent
)

func NewServiceBuilder(agent *instanav1.RemoteAgent) builder.ObjectBuilder {
	return &serviceBuilder{
		remoteAgent: agent,

		podSelectorLabelGenerator: transformations.PodSelectorLabelsRemote(agent, componentName),
		portsBuilder:              ports.NewPortsBuilderRemote(agent),
	}
}

type serviceBuilder struct {
	remoteAgent               *instanav1.RemoteAgent
	podSelectorLabelGenerator transformations.PodSelectorLabelGenerator
	portsBuilder              ports.PortsBuilder
}

func (s *serviceBuilder) ComponentName() string {
	return componentName
}

func (s *serviceBuilder) IsNamespaced() bool {
	return true
}

func (s *serviceBuilder) build() *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.remoteAgent.Name,
			Namespace: s.remoteAgent.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: s.podSelectorLabelGenerator.GetPodSelectorLabels(),
			Ports: s.portsBuilder.GetServicePorts(
				ports.AgentAPIsPort,
				ports.OpenTelemetryLegacyPort,
				ports.OpenTelemetryGRPCPort,
				ports.OpenTelemetryHTTPPort,
			),
			InternalTrafficPolicy: pointer.To(corev1.ServiceInternalTrafficPolicyLocal),
		},
	}
}

func (s *serviceBuilder) Build() optional.Optional[client.Object] {
	switch {
	case pointer.DerefOrEmpty(s.remoteAgent.Spec.Service.Create):
		fallthrough
	default:
		return optional.Empty[client.Object]()
	}
}
