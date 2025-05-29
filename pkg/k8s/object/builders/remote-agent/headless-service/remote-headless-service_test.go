/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

	http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
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

func TestRemoteHeadlessServiceBuilder_IsNamespaced(t *testing.T) {
	assertions := require.New(t)
	assertions.True(NewHeadlessServiceBuilder(nil).IsNamespaced())
}

func TestRemoteHeadlessServiceBuilder_ComponentName(t *testing.T) {
	assertions := require.New(t)
	assertions.Equal(constants.ComponentRemoteAgent, NewHeadlessServiceBuilder(nil).ComponentName())
}

func TestRemoteHeadlessServiceBuilder_Build(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	agent := &instanav1.RemoteAgent{
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
	portsBuilder.EXPECT().GetServicePorts(
		ports.AgentAPIsPort,
	).
		Return(
			[]corev1.ServicePort{
				{
					Name: "headless-service-port",
				},
			},
		)

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

	actual := (&remoteHeadlessServiceBuilder{
		RemoteAgent:                     agent,
		RemoteHelpers:                   hlprs,
		PodSelectorLabelGeneratorRemote: podSelectorLabelGenerator,
		PortsBuilderRemote:              portsBuilder,
	}).Build()

	assertions.Equal(expected, actual)
}

func TestRemoteNewHeadlessServiceBuilder(t *testing.T) {
	assertions := require.New(t)

	agent := &instanav1.RemoteAgent{ObjectMeta: metav1.ObjectMeta{Name: "some-agent"}}

	expected := &remoteHeadlessServiceBuilder{
		RemoteAgent:                     agent,
		RemoteHelpers:                   helpers.NewRemoteHelpers(agent),
		PodSelectorLabelGeneratorRemote: transformations.PodSelectorLabelsRemote(agent, constants.ComponentRemoteAgent),
		PortsBuilderRemote:              ports.NewPortsBuilderRemote(agent),
	}

	actual := NewHeadlessServiceBuilder(agent)

	assertions.Equal(expected, actual)
}
