package headless_service

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
)

const (
	componentName = constants.ComponentRemoteAgent
)

type remoteHeadlessServiceBuilder struct {
	*instanav1.RemoteAgent

	helpers.RemoteHelpers
	transformations.PodSelectorLabelGenerator
	ports.PortsBuilder
}

func (h *remoteHeadlessServiceBuilder) Build() builder.OptionalObject {
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
				Ports: h.GetServicePorts(
					ports.AgentAPIsPort,
					ports.OpenTelemetryLegacyPort,
					ports.OpenTelemetryGRPCPort,
					ports.OpenTelemetryHTTPPort,
				),
			},
		},
	)
}

func (h *remoteHeadlessServiceBuilder) ComponentName() string {
	return componentName
}

func (h *remoteHeadlessServiceBuilder) IsNamespaced() bool {
	return true
}

func NewHeadlessServiceBuilder(agent *instanav1.RemoteAgent) builder.ObjectBuilder {
	return &remoteHeadlessServiceBuilder{
		RemoteAgent: agent,

		RemoteHelpers:             helpers.NewRemoteHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabelsRemote(agent, componentName),
		PortsBuilder:              ports.NewPortsBuilderRemote(agent),
	}
}
