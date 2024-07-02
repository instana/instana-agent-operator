/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc.

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

package ports

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	corev1 "k8s.io/api/core/v1"
)

type PortsBuilder interface {
	GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort
	GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort
}

type portsBuilder struct {
	InstanaAgent *instanav1.InstanaAgent
}

func NewPortsBuilder(agent *instanav1.InstanaAgent) PortsBuilder {
	return &portsBuilder{
		InstanaAgent: agent,
	}
}

func (p *portsBuilder) GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort {
	enabledPorts := list.NewListFilter[InstanaAgentPort]().Filter(
		ports, func(port InstanaAgentPort) bool {
			return port.IsEnabled(p.InstanaAgent.Spec.OpenTelemetry)
		},
	)

	return list.NewListMapTo[InstanaAgentPort, corev1.ServicePort]().MapTo(enabledPorts, toServicePort)
}

func (p *portsBuilder) GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort {
	return list.NewListMapTo[InstanaAgentPort, corev1.ContainerPort]().MapTo(ports, toContainerPort)
}
