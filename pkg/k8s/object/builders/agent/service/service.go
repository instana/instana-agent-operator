package service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	componentName = constants.ComponentInstanaAgent
)

type serviceBuilder struct {
	*instanav1.InstanaAgent

	transformations.PodSelectorLabelGenerator
	ports.PortsBuilder
	helpers.OpenTelemetrySettings
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
			Name:      s.Name,
			Namespace: s.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector: s.GetPodSelectorLabels(),
			Ports: s.GetServicePorts(
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
	case pointer.DerefOrEmpty(s.Spec.Service.Create):
		fallthrough
	case pointer.DerefOrEmpty(s.Spec.Prometheus.RemoteWrite.Enabled):
		fallthrough
	case s.OpenTelemetrySettings.IsEnabled():
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}

func NewServiceBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &serviceBuilder{
		InstanaAgent: agent,

		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		PortsBuilder:              ports.NewPortsBuilder(agent),
		OpenTelemetrySettings:     agent.Spec.OpenTelemetry,
	}
}
