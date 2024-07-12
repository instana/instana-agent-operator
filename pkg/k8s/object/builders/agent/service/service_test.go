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

package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestServiceBuilder_ComponentName_IsNamespaced(t *testing.T) {
	assertions := require.New(t)

	sb := NewServiceBuilder(&instanav1.InstanaAgent{})

	assertions.True(sb.IsNamespaced())
	assertions.Equal(constants.ComponentInstanaAgent, sb.ComponentName())
}

func TestServiceBuilder_Build(t *testing.T) {
	for _, serviceCreate := range []*bool{nil, pointer.To(true), pointer.To(false)} {
		for _, remoteWriteEnabled := range []instanav1.Enabled{
			{Enabled: pointer.To(true)},
			{Enabled: pointer.To(false)},
			{Enabled: nil},
		} {
			for _, otlpIsEnabled := range []bool{true, false} {
				t.Run(
					fmt.Sprintf(
						"service.create=%v,prometheus.remoteWrite.enabled=%v,otlpEnabled=%v",
						serviceCreate,
						remoteWriteEnabled,
						otlpIsEnabled,
					),
					func(t *testing.T) {
						assertions := require.New(t)
						ctrl := gomock.NewController(t)

						name := rand.String(10)
						namespace := rand.String(10)

						agent := instanav1.InstanaAgent{
							ObjectMeta: metav1.ObjectMeta{
								Name:      name,
								Namespace: namespace,
							},
							Spec: instanav1.InstanaAgentSpec{
								Service: instanav1.Create{Create: serviceCreate},
								Prometheus: instanav1.Prometheus{
									RemoteWrite: remoteWriteEnabled,
								},
							},
						}

						var otlpSettings instanav1.OpenTelemetry
						if !pointer.DerefOrEmpty(serviceCreate) && (remoteWriteEnabled.Enabled == nil || !*remoteWriteEnabled.Enabled) {
							otlpSettings = instanav1.OpenTelemetry{
								Enabled: instanav1.Enabled{Enabled: &otlpIsEnabled},
								GRPC:    &instanav1.Enabled{Enabled: &otlpIsEnabled},
								HTTP:    &instanav1.Enabled{Enabled: &otlpIsEnabled},
							}
						} else {
							otlpSettings = instanav1.OpenTelemetry{
								Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
								GRPC:    &instanav1.Enabled{Enabled: pointer.To(false)},
								HTTP:    &instanav1.Enabled{Enabled: pointer.To(false)},
							}
						}

						podSelectorLabelGenerator := mocks.NewMockPodSelectorLabelGenerator(ctrl)

						portsBuilder := mocks.NewMockPortsBuilder(ctrl)

						sb := &serviceBuilder{
							instanaAgent:              &agent,
							podSelectorLabelGenerator: podSelectorLabelGenerator,
							portsBuilder:              portsBuilder,
							openTelemetrySettings:     otlpSettings,
						}

						if pointer.DerefOrEmpty(serviceCreate) || (remoteWriteEnabled.Enabled != nil && *remoteWriteEnabled.Enabled) || otlpIsEnabled {
							expectedSelectorLabels := map[string]string{
								rand.String(rand.IntnRange(1, 15)): rand.String(rand.IntnRange(1, 15)),
								rand.String(rand.IntnRange(1, 15)): rand.String(rand.IntnRange(1, 15)),
								rand.String(rand.IntnRange(1, 15)): rand.String(rand.IntnRange(1, 15)),
							}
							podSelectorLabelGenerator.EXPECT().GetPodSelectorLabels().Return(expectedSelectorLabels)

							expectedServicePorts := []corev1.ServicePort{
								{Name: rand.String(rand.IntnRange(1, 15))},
								{Name: rand.String(rand.IntnRange(1, 15))},
								{Name: rand.String(rand.IntnRange(1, 15))},
							}
							portsBuilder.EXPECT().GetServicePorts(
								ports.AgentAPIsPort,
								ports.OpenTelemetryLegacyPort,
								ports.OpenTelemetryGRPCPort,
								ports.OpenTelemetryHTTPPort,
							).Return(expectedServicePorts)

							expected := optional.Of[client.Object](
								&corev1.Service{
									TypeMeta: metav1.TypeMeta{
										APIVersion: "v1",
										Kind:       "Service",
									},
									ObjectMeta: metav1.ObjectMeta{
										Name:      name,
										Namespace: namespace,
									},
									Spec: corev1.ServiceSpec{
										Selector:              expectedSelectorLabels,
										Ports:                 expectedServicePorts,
										InternalTrafficPolicy: pointer.To(corev1.ServiceInternalTrafficPolicyLocal),
									},
								},
							)

							actual := sb.Build()

							assertions.Equal(expected, actual)
						} else {
							res := sb.Build()

							assertions.Empty(res)
						}
					},
				)
			}
		}
	}
}
