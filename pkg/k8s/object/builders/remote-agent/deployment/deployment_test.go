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

package deployment

import (
	"testing"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/mocks"
	backend "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"
)

func TestDeploymentBuilder_getPodTemplateLabels(t *testing.T) {
	for _, test := range []struct {
		name              string
		getPodLabelsInput map[string]string
		agentSpec         instanav1.RemoteAgentSpec
	}{
		{
			name: "agent_mode_unset",
			getPodLabelsInput: map[string]string{
				"instana/agent-mode": string(instanav1.APM),
			},
			agentSpec: instanav1.RemoteAgentSpec{},
		},
		{
			name: "agent_mode_set_by_user",
			getPodLabelsInput: map[string]string{
				"instana/agent-mode": string(instanav1.KUBERNETES),
			},
			agentSpec: instanav1.RemoteAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Mode: instanav1.KUBERNETES,
				},
			},
		},
		{
			name: "agent_mode_unset_with_user_given_pod_labels",
			getPodLabelsInput: map[string]string{
				"asdfasdf":           "eoisdgoinv",
				"reoirionv":          "98458hgoisjdf",
				"instana/agent-mode": string(instanav1.APM),
			},
			agentSpec: instanav1.RemoteAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Pod: instanav1.AgentPodSpec{
						Labels: map[string]string{
							"asdfasdf":  "eoisdgoinv",
							"reoirionv": "98458hgoisjdf",
						},
					},
				},
			},
		},
		{
			name: "agent_mode_set_by_user_with_user_given_pod_labels",
			getPodLabelsInput: map[string]string{
				"asdfasdf":           "eoisdgoinv",
				"reoirionv":          "98458hgoisjdf",
				"instana/agent-mode": string(instanav1.KUBERNETES),
			},
			agentSpec: instanav1.RemoteAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					Mode: instanav1.KUBERNETES,
					Pod: instanav1.AgentPodSpec{
						Labels: map[string]string{
							"asdfasdf":  "eoisdgoinv",
							"reoirionv": "98458hgoisjdf",
						},
					},
				},
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				expected := map[string]string{
					"adsf":      "eroinsvd",
					"osdgoiego": "rwuriunsv",
					"e8uriunv":  "rrudsiu",
				}

				podSelector := mocks.NewMockPodSelectorLabelGenerator(ctrl)
				podSelector.EXPECT().GetPodLabels(gomock.Eq(test.getPodLabelsInput)).Return(expected)

				d := &deploymentBuilder{
					RemoteAgent: &instanav1.RemoteAgent{
						Spec: test.agentSpec,
					},
					PodSelectorLabelGenerator: podSelector,
				}

				actual := d.getPodTemplateLabels()

				assertions.Equal(expected, actual)
			},
		)
	}
}

func TestDeploymentBuilder_getEnvVars(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expected := []corev1.EnvVar{
		{
			Name:  "foo",
			Value: "bar",
		},
		{
			Name:  "hello",
			Value: "world",
		},
	}

	envBuilder := mocks.NewMockEnvBuilderRemote(ctrl)
	envBuilder.EXPECT().Build(
		env.AgentModeEnvRemote,
		env.ZoneNameEnvRemote,
		env.ClusterNameEnvRemote,
		env.AgentEndpointEnvRemote,
		env.AgentEndpointPortEnvRemote,
		env.MavenRepoURLEnvRemote,
		env.MavenRepoFeaturesPathRemote,
		env.MavenRepoSharedPathRemote,
		env.MirrorReleaseRepoUrlEnvRemote,
		env.MirrorReleaseRepoUsernameEnvRemote,
		env.MirrorReleaseRepoPasswordEnvRemote,
		env.MirrorSharedRepoUrlEnvRemote,
		env.MirrorSharedRepoUsernameEnvRemote,
		env.MirrorSharedRepoPasswordEnvRemote,
		env.ProxyHostEnvRemote,
		env.ProxyPortEnvRemote,
		env.ProxyProtocolEnvRemote,
		env.ProxyUserEnvRemote,
		env.ProxyPasswordEnvRemote,
		env.ProxyUseDNSEnvRemote,
		env.ListenAddressEnvRemote,
		env.RedactK8sSecretsEnvRemote,
		env.ConfigPathEnvRemote,
		env.EntrypointSkipBackendTemplateGenerationRemote,
		env.InstanaAgentKeyEnvRemote,
		env.DownloadKeyEnvRemote,
		env.InstanaAgentPodNameEnvRemote,
		env.PodIPEnvRemote,
		env.K8sServiceDomainEnvRemote,
		env.EnableAgentSocketEnvRemote,
	).
		Return(expected)

	agent := &instanav1.RemoteAgent{ObjectMeta: metav1.ObjectMeta{Name: "some-agent"}}
	db := &deploymentBuilder{
		EnvBuilderRemote: envBuilder,
		RemoteHelpers:    helpers.NewRemoteHelpers(agent),
	}

	actual := db.getEnvVars()

	assertions.Equal(expected, actual)
}

func TestDeploymentBuilder_getContainerPorts(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expected := []corev1.ContainerPort{
		{
			Name:          "something",
			ContainerPort: 12345,
		},
	}

	portsBuilder := mocks.NewMockPortsBuilderRemote(ctrl)
	portsBuilder.EXPECT().GetContainerPorts(
		ports.AgentAPIsPort,
	).Return(expected)

	db := &deploymentBuilder{
		PortsBuilderRemote: portsBuilder,
	}

	actual := db.getContainerPorts()

	assertions.Equal(expected, actual)
}

func TestDeploymentBuilder_getVolumes(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expectedVolumes := []corev1.Volume{{Name: rand.String(10)}}
	expectedVolumeMounts := []corev1.VolumeMount{{Name: rand.String(10)}}

	volumeBuilder := mocks.NewMockVolumeBuilderRemote(ctrl)
	volumeBuilder.EXPECT().Build(
		gomock.Eq(volume.ConfigVolumeRemote),
		gomock.Eq(volume.TlsVolumeRemote),
		gomock.Eq(volume.RepoVolumeRemote),
	).Return(expectedVolumes, expectedVolumeMounts)

	db := &deploymentBuilder{
		VolumeBuilderRemote: volumeBuilder,
	}

	actualVolumes, actualVolumeMounts := db.getVolumes()

	assertions.Equal(expectedVolumes, actualVolumes)
	assertions.Equal(expectedVolumeMounts, actualVolumeMounts)
}

func TestDeploymentBuilder_getUserVolumes(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	volumeName := "testVolume"
	expectedVolumes := []corev1.Volume{{Name: volumeName}}
	expectedVolumeMounts := []corev1.VolumeMount{{Name: volumeName}}

	volumeBuilder := mocks.NewMockVolumeBuilderRemote(ctrl)
	volumeBuilder.EXPECT().BuildFromUserConfig().Return(expectedVolumes, expectedVolumeMounts)

	agent := &instanav1.RemoteAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testAgent",
		},
		Spec: instanav1.RemoteAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Pod: instanav1.AgentPodSpec{
					Volumes: []corev1.Volume{
						{
							Name: volumeName,
						},
					},
					VolumeMounts: []corev1.VolumeMount{
						{
							Name: volumeName,
						},
					},
				},
			},
		},
	}
	db := &deploymentBuilder{
		VolumeBuilderRemote: volumeBuilder,
		RemoteAgent:         agent,
	}

	actualVolumes, actualVolumeMounts := db.getUserVolumes()

	assertions.Equal(expectedVolumes, actualVolumes)
	assertions.Equal(expectedVolumeMounts, actualVolumeMounts)
}

func TestDeploymentBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := assert.New(t)

	emptyBackend := backend.K8SensorBackend{}
	dBuilder := NewDeploymentBuilder(nil, nil, emptyBackend, nil)

	assertions.True(dBuilder.IsNamespaced())
	assertions.Equal(constants.ComponentRemoteAgent, dBuilder.ComponentName())
}

func TestZoning(t *testing.T) {
	agentName := rand.String(10)
	zoneName := rand.String(10)

	for _, test := range []struct {
		name                      string
		expectedName              string
		hasZoneSet                bool
		expectedNonStandardLabels map[string]string
		expectedAffinity          *corev1.Affinity
		expectedTolerations       []corev1.Toleration
	}{
		{
			name:                      "no_zone_set",
			expectedName:              agentName,
			hasZoneSet:                false,
			expectedNonStandardLabels: nil,
			expectedTolerations:       []corev1.Toleration{{Key: agentName}},
		},
		{
			name:         "with_zone_set",
			expectedName: agentName + "-" + zoneName,
			hasZoneSet:   true,
			expectedNonStandardLabels: map[string]string{
				transformations.ZoneLabel: zoneName,
			},
			expectedTolerations: []corev1.Toleration{{Key: zoneName}},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				agent := &instanav1.RemoteAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name: agentName,
					},
					Spec: instanav1.RemoteAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Pod: instanav1.AgentPodSpec{
								Affinity: corev1.Affinity{},
								Tolerations: []corev1.Toleration{
									{
										Key: agentName,
									},
								},
							},
						},
					},
				}
				zone := &instanav1.Zone{
					Name: instanav1.Name{
						Name: zoneName,
					},
					Affinity: corev1.Affinity{},
					Tolerations: []corev1.Toleration{
						{
							Key: zoneName,
						},
					},
				}

				dBuilder := &deploymentBuilder{
					RemoteAgent: agent,
				}

				if test.hasZoneSet {
					dBuilder.zone = zone
				}

				t.Run(
					"getName", func(t *testing.T) {
						actualName := dBuilder.getName()
						assertions.Equal(test.expectedName, actualName)
					},
				)

				t.Run(
					"getNonStandardLabels", func(t *testing.T) {
						actualNonStandardLabels := dBuilder.getNonStandardLabels()
						assertions.Equal(test.expectedNonStandardLabels, actualNonStandardLabels)
					},
				)

				t.Run(
					"getAffinity", func(t *testing.T) {
						assertions.NotSame(&zone.Affinity, &agent.Spec.Agent.Pod.Affinity)

						expectedAffinity := func() *corev1.Affinity {
							switch test.hasZoneSet {
							case true:
								return &zone.Affinity
							default:
								return &agent.Spec.Agent.Pod.Affinity
							}
						}()

						actualAffinity := dBuilder.getAffinity()
						assertions.Same(expectedAffinity, actualAffinity)
					},
				)

				t.Run(
					"getTolerations", func(t *testing.T) {
						actualTolerations := dBuilder.getTolerations()
						assertions.Equal(test.expectedTolerations, actualTolerations)
					},
				)
			},
		)
	}
}

func TestDeploymentBuilder_Build(t *testing.T) {
	tests := []struct {
		name          string
		agent         *instanav1.RemoteAgent
		expectPresent bool
	}{
		{
			name:          "should_be_not_present",
			agent:         &instanav1.RemoteAgent{},
			expectPresent: false,
		},
		{
			name: "should_be_present",
			agent: &instanav1.RemoteAgent{
				Spec: instanav1.RemoteAgentSpec{
					Agent: instanav1.BaseAgentSpec{
						Key: "key",
					},
					Cluster: instanav1.Name{
						Name: "cluster",
					},
					Zone: instanav1.Name{
						Name: "zone-a",
					},
				},
			},
			expectPresent: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertions := assert.New(t)
			ctrl := gomock.NewController(t)
			defer ctrl.Finish()

			status := mocks.NewMockRemoteAgentStatusManager(ctrl)
			if test.expectPresent {
				status.EXPECT().AddAgentDeployment(gomock.Any())
			}

			emptyBackend := backend.K8SensorBackend{}
			dBuilder := NewDeploymentBuilder(test.agent, status, emptyBackend, nil)

			result := dBuilder.Build()
			assertions.Equal(test.expectPresent, result.IsPresent())
		})
	}
}
