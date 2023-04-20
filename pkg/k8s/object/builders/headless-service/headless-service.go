package headless_service

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

const (
	componentName = constants.ComponentInstanaAgent
)

type headlessServiceBuilder struct {
	*instanav1.InstanaAgent

	helpers.Helpers
	transformations.PodSelectorLabelGenerator
	ports.PortsBuilder
}

func (h *headlessServiceBuilder) Build() builder.OptionalObject {
	return optional.Of[client.Object](
		&corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      h.HeadlessServiceName(),
				Namespace: h.Namespace,
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: corev1.ClusterIPNone,
				Selector:  h.GetPodSelectorLabels(),
				Ports:     h.GetServicePorts(ports.AgentAPIsPort, ports.AgentSocketPort),
			},
		},
	)
}

func (h *headlessServiceBuilder) ComponentName() string {
	return componentName
}

func (h *headlessServiceBuilder) IsNamespaced() bool {
	return true
}

func NewHeadlessServiceBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &headlessServiceBuilder{
		InstanaAgent: agent,

		Helpers:                   helpers.NewHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		PortsBuilder:              ports.NewPortsBuilder(agent),
	}
}
