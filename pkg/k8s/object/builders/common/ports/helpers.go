/*
(c) Copyright IBM Corp. 2024, 2025
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
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func toServicePort(port InstanaAgentPort) corev1.ServicePort {
	return corev1.ServicePort{
		Name:       port.String(),
		Protocol:   corev1.ProtocolTCP,
		Port:       port.PortNumber(),
		TargetPort: intstr.FromString(port.String()),
	}
}

func toContainerPort(port InstanaAgentPort) corev1.ContainerPort {
	return corev1.ContainerPort{
		Name:          port.String(),
		ContainerPort: port.PortNumber(),
		Protocol:      corev1.ProtocolTCP,
	}
}
