/*
(c) Copyright IBM Corp. 2024, 2025
*/

package headless_service

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

func TestHeadlessServiceBuilder_IsNamespaced(t *testing.T) {
	assertions := require.New(t)
	assertions.True(NewHeadlessServiceBuilder(&instanav1.InstanaAgent{}).IsNamespaced())
}

func TestHeadlessServiceBuilder_ComponentName(t *testing.T) {
	assertions := require.New(t)
	assertions.Equal(constants.ComponentInstanaAgent, NewHeadlessServiceBuilder(&instanav1.InstanaAgent{}).ComponentName())
}

func TestHeadlessServiceBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "agent-namespace",
		},
	}

	hlprs := mocks.NewMockHelpers(ctrl)
	hlprs.EXPECT().HeadlessServiceName().Return("headless-service-name")

	podSelectorLabelGenerator := mocks.NewMockPodSelectorLabelGenerator(ctrl)
	podSelectorLabelGenerator.EXPECT().
		GetPodSelectorLabels().
		Return(map[string]string{"foo": "bar", "hello": "world"})

	portsBuilder := mocks.NewMockPortsBuilder(ctrl)
	portsBuilder.EXPECT().GetServicePorts().Return([]corev1.ServicePort{{Name: "headless-service-port"}})

	expected := optional.Of[client.Object](
		&corev1.Service{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "v1",
				Kind:       "Service",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "headless-service-name",
				Namespace: "agent-namespace",
			},
			Spec: corev1.ServiceSpec{
				ClusterIP: corev1.ClusterIPNone,
				Selector:  map[string]string{"foo": "bar", "hello": "world"},
				Ports:     []corev1.ServicePort{{Name: "headless-service-port"}},
			},
		},
	)

	actual := (&headlessServiceBuilder{
		InstanaAgent:              agent,
		Helpers:                   hlprs,
		PodSelectorLabelGenerator: podSelectorLabelGenerator,
		PortsBuilder:              portsBuilder,
	}).Build()

	assertions.Equal(expected, actual)
}

func TestNewHeadlessServiceBuilder(t *testing.T) {
	assertions := require.New(t)

	agent := &instanav1.InstanaAgent{ObjectMeta: metav1.ObjectMeta{Name: "some-agent"}}

	expected := &headlessServiceBuilder{
		InstanaAgent:              agent,
		Helpers:                   helpers.NewHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, constants.ComponentInstanaAgent),
		PortsBuilder:              ports.NewPortsBuilder(agent.Spec.OpenTelemetry),
	}

	actual := NewHeadlessServiceBuilder(agent)

	assertions.Equal(expected, actual)
}
