/*
(c) Copyright IBM Corp. 2024, 2025
*/

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
	componentName = constants.ComponentInstanaAgent
)

func NewServiceBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &serviceBuilder{
		instanaAgent:              agent,
		podSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		portsBuilder:              ports.NewPortsBuilder(agent.Spec.OpenTelemetry),
		openTelemetrySettings:     agent.Spec.OpenTelemetry,
	}
}

type serviceBuilder struct {
	instanaAgent              *instanav1.InstanaAgent
	podSelectorLabelGenerator transformations.PodSelectorLabelGenerator
	portsBuilder              ports.PortsBuilder
	openTelemetrySettings     instanav1.OpenTelemetry
}

func (s *serviceBuilder) ComponentName() string {
	return componentName
}

func (s *serviceBuilder) IsNamespaced() bool {
	return true
}

func (s *serviceBuilder) build() *corev1.Service {
	localPolicy := corev1.ServiceInternalTrafficPolicyLocal
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "v1",
			Kind:       "Service",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      s.instanaAgent.Name,
			Namespace: s.instanaAgent.Namespace,
		},
		Spec: corev1.ServiceSpec{
			Selector:              s.podSelectorLabelGenerator.GetPodSelectorLabels(),
			Ports:                 s.portsBuilder.GetServicePorts(),
			InternalTrafficPolicy: &localPolicy,
		},
	}
}

func (s *serviceBuilder) Build() optional.Optional[client.Object] {
	switch {
	case pointer.DerefOrEmpty(s.instanaAgent.Spec.Service.Create):
		fallthrough
	case pointer.DerefOrEmpty(s.instanaAgent.Spec.Prometheus.RemoteWrite.Enabled):
		fallthrough
	case *s.openTelemetrySettings.Enabled.Enabled:
		return optional.Of[client.Object](s.build())
	default:
		return optional.Empty[client.Object]()
	}
}
