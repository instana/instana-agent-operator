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

package ports

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	corev1 "k8s.io/api/core/v1"
)

type RemoteAgentPort string
type PortsBuilderRemote interface {
	GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort
	GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort
}

type portsBuilderRemote struct {
	RemoteAgent *instanav1.RemoteAgent
}

func NewPortsBuilderRemote(agent *instanav1.RemoteAgent) PortsBuilderRemote {
	return &portsBuilderRemote{
		RemoteAgent: agent,
	}
}

func (p *portsBuilderRemote) GetServicePorts(ports ...InstanaAgentPort) []corev1.ServicePort {
	return list.NewListMapTo[InstanaAgentPort, corev1.ServicePort]().MapTo(ports, toServicePort)
}

func (p *portsBuilderRemote) GetContainerPorts(ports ...InstanaAgentPort) []corev1.ContainerPort {
	return list.NewListMapTo[InstanaAgentPort, corev1.ContainerPort]().MapTo(ports, toContainerPort)
}
