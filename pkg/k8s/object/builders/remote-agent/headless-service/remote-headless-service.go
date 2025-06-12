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
	transformations.PodSelectorLabelGeneratorRemote
	ports.PortsBuilderRemote
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
		RemoteAgent:                     agent,
		RemoteHelpers:                   helpers.NewRemoteHelpers(agent),
		PodSelectorLabelGeneratorRemote: transformations.PodSelectorLabelsRemote(agent, componentName),
		PortsBuilderRemote:              ports.NewPortsBuilderRemote(agent),
	}
}
