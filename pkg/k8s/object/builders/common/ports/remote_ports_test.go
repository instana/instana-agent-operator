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

package ports_test

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
)

func TestRemoteAgentPortMappings(t *testing.T) {

	for _, test := range []struct {
		name               string
		port               ports.InstanaAgentPort
		expectedPortNumber int32
		expectEnabled      bool
		expectPanic        bool
	}{
		{
			name:               string(ports.AgentAPIsPort),
			port:               ports.AgentAPIsPort,
			expectedPortNumber: ports.AgentAPIsPortNumber,
			expectEnabled:      true,
		},

		{
			name:          "unknown_port",
			port:          ports.InstanaAgentPort("unknown"),
			expectEnabled: true,
			expectPanic:   true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				assertions.Equal(string(test.port), test.port.String())

				if test.expectPanic {
					assertions.PanicsWithError(
						"unknown port requested", func() {
							test.port.PortNumber()
						},
					)
				} else {
					assertions.Equal(test.expectedPortNumber, test.port.PortNumber())
				}
			},
		)
	}
}

func TestPortsBuildeRemoteGetPorts(t *testing.T) {
	containerPortAgentAPI := corev1.ContainerPort{
		Name:          string(ports.AgentAPIsPort),
		ContainerPort: ports.AgentAPIsPortNumber,
		Protocol:      corev1.ProtocolTCP,
	}

	servicePortAgentAPI := corev1.ServicePort{
		Name:       string(ports.AgentAPIsPort),
		Port:       ports.AgentAPIsPortNumber,
		TargetPort: intstr.FromString(string(ports.AgentAPIsPort)),
		Protocol:   corev1.ProtocolTCP,
	}

	for _, test := range []struct {
		name                   string
		expectedContainerPorts []corev1.ContainerPort
		expectedServicePorts   []corev1.ServicePort
	}{
		{
			name:                   "Enabled all by default",
			expectedContainerPorts: []corev1.ContainerPort{containerPortAgentAPI},
			expectedServicePorts:   []corev1.ServicePort{servicePortAgentAPI},
		},
	} {
		assertions := require.New(t)
		actualContainerPorts := ports.
			NewPortsBuilderRemote(&instanav1.RemoteAgent{}).
			GetContainerPorts(
				ports.AgentAPIsPort,
			)

		actualServicePorts := ports.
			NewPortsBuilderRemote(&instanav1.RemoteAgent{}).
			GetServicePorts(
				ports.AgentAPIsPort,
			)

		assertions.Equal(test.expectedContainerPorts, actualContainerPorts)
		assertions.Equal(test.expectedServicePorts, actualServicePorts)
	}
}
