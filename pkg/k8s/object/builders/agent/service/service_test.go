/*
(c) Copyright IBM Corp. 2024, 2025
*/

package service

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
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
								GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: &otlpIsEnabled},
								HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: &otlpIsEnabled},
							}
						} else {
							otlpSettings = instanav1.OpenTelemetry{
								Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
								GRPC:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(false)},
								HTTP:    instanav1.OpenTelemetryPortConfig{Enabled: pointer.To(false)},
							}
						}

						podSelectorLabelGenerator := &mocks.MockPodSelectorLabelGenerator{}
						defer podSelectorLabelGenerator.AssertExpectations(t)

						portsBuilder := &mocks.MockPortsBuilder{}
						defer portsBuilder.AssertExpectations(t)

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
							podSelectorLabelGenerator.On("GetPodSelectorLabels").Return(expectedSelectorLabels)

							expectedServicePorts := []corev1.ServicePort{
								{Name: rand.String(rand.IntnRange(1, 15))},
								{Name: rand.String(rand.IntnRange(1, 15))},
								{Name: rand.String(rand.IntnRange(1, 15))},
							}
							portsBuilder.On("GetServicePorts").Return(expectedServicePorts)

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
