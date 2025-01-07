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

package daemonset

import (
	"testing"

	"github.com/instana/instana-agent-operator/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gomock "go.uber.org/mock/gomock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
)

func TestDaemonSetBuilder_getPodTemplateLabels(t *testing.T) {
	for _, test := range []struct {
		name              string
		getPodLabelsInput map[string]string
		agentSpec         instanav1.InstanaAgentSpec
	}{
		{
			name: "agent_mode_unset",
			getPodLabelsInput: map[string]string{
				"instana/agent-mode": string(instanav1.APM),
			},
			agentSpec: instanav1.InstanaAgentSpec{},
		},
		{
			name: "agent_mode_set_by_user",
			getPodLabelsInput: map[string]string{
				"instana/agent-mode": string(instanav1.KUBERNETES),
			},
			agentSpec: instanav1.InstanaAgentSpec{
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
			agentSpec: instanav1.InstanaAgentSpec{
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
			agentSpec: instanav1.InstanaAgentSpec{
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

				d := &daemonSetBuilder{
					InstanaAgent: &instanav1.InstanaAgent{
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

func TestDaemonSetBuilder_getEnvVars(t *testing.T) {
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

	envBuilder := mocks.NewMockEnvBuilder(ctrl)
	envBuilder.EXPECT().Build(
		env.AgentModeEnv,
		env.ZoneNameEnv,
		env.ClusterNameEnv,
		env.AgentEndpointEnv,
		env.AgentEndpointPortEnv,
		env.MavenRepoURLEnv,
		env.MavenRepoFeaturesPath,
		env.MavenRepoSharedPath,
		env.MirrorFeaturesRepoUrlEnv,
		env.MirrorFeaturesRepoUsernameEnv,
		env.MirrorFeaturesRepoPasswordEnv,
		env.MirrorSharedRepoUrlEnv,
		env.MirrorSharedRepoUsernameEnv,
		env.MirrorSharedRepoPasswordEnv,
		env.ProxyHostEnv,
		env.ProxyPortEnv,
		env.ProxyProtocolEnv,
		env.ProxyUserEnv,
		env.ProxyPasswordEnv,
		env.ProxyUseDNSEnv,
		env.ListenAddressEnv,
		env.RedactK8sSecretsEnv,
		env.ConfigPathEnv,
		env.EntrypointSkipBackendTemplateGeneration,
		env.InstanaAgentKeyEnv,
		env.DownloadKeyEnv,
		env.InstanaAgentPodNameEnv,
		env.PodIPEnv,
		env.K8sServiceDomainEnv,
		env.EnableAgentSocketEnv,
	).
		Return(expected)

	db := &daemonSetBuilder{
		EnvBuilder: envBuilder,
	}

	actual := db.getEnvVars()

	assertions.Equal(expected, actual)
}

func TestDaemonSetBuilder_getContainerPorts(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expected := []corev1.ContainerPort{
		{
			Name:          "something",
			ContainerPort: 12345,
		},
	}

	portsBuilder := mocks.NewMockPortsBuilder(ctrl)
	portsBuilder.EXPECT().GetContainerPorts(
		ports.AgentAPIsPort,
		ports.OpenTelemetryLegacyPort,
		ports.OpenTelemetryGRPCPort,
		ports.OpenTelemetryHTTPPort,
	).Return(expected)

	db := &daemonSetBuilder{
		PortsBuilder: portsBuilder,
	}

	actual := db.getContainerPorts()

	assertions.Equal(expected, actual)
}

func TestDaemonSetBuilder_getVolumes(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expectedVolumes := []corev1.Volume{{Name: rand.String(10)}}
	expectedVolumeMounts := []corev1.VolumeMount{{Name: rand.String(10)}}

	volumeBuilder := mocks.NewMockVolumeBuilder(ctrl)
	volumeBuilder.EXPECT().Build(
		gomock.Eq(volume.DevVolume),
		gomock.Eq(volume.RunVolume),
		gomock.Eq(volume.VarRunVolume),
		gomock.Eq(volume.VarRunKuboVolume),
		gomock.Eq(volume.VarRunContainerdVolume),
		gomock.Eq(volume.VarContainerdConfigVolume),
		gomock.Eq(volume.SysVolume),
		gomock.Eq(volume.VarLogVolume),
		gomock.Eq(volume.VarLibVolume),
		gomock.Eq(volume.VarDataVolume),
		gomock.Eq(volume.MachineIdVolume),
		gomock.Eq(volume.ConfigVolume),
		gomock.Eq(volume.TlsVolume),
		gomock.Eq(volume.RepoVolume),
	).Return(expectedVolumes, expectedVolumeMounts)

	db := &daemonSetBuilder{
		VolumeBuilder: volumeBuilder,
	}

	actualVolumes, actualVolumeMounts := db.getVolumes()

	assertions.Equal(expectedVolumes, actualVolumes)
	assertions.Equal(expectedVolumeMounts, actualVolumeMounts)
}

func TestDaemonSetBuilder_getUserVolumes(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	volumeName := "testVolume"
	expectedVolumes := []corev1.Volume{{Name: volumeName}}
	expectedVolumeMounts := []corev1.VolumeMount{{Name: volumeName}}

	volumeBuilder := mocks.NewMockVolumeBuilder(ctrl)
	volumeBuilder.EXPECT().BuildFromUserConfig().Return(expectedVolumes, expectedVolumeMounts)

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name: "testAgent",
		},
		Spec: instanav1.InstanaAgentSpec{
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
	db := &daemonSetBuilder{
		VolumeBuilder: volumeBuilder,
		InstanaAgent:  agent,
	}

	actualVolumes, actualVolumeMounts := db.getUserVolumes()

	assertions.Equal(expectedVolumes, actualVolumes)
	assertions.Equal(expectedVolumeMounts, actualVolumeMounts)
}

func TestDaemonSetBuilder_IsNamespaced_ComponentName(t *testing.T) {
	assertions := assert.New(t)

	dsBuilder := NewDaemonSetBuilder(nil, false, nil)

	assertions.True(dsBuilder.IsNamespaced())
	assertions.Equal(constants.ComponentInstanaAgent, dsBuilder.ComponentName())
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

				agent := &instanav1.InstanaAgent{
					ObjectMeta: metav1.ObjectMeta{
						Name: agentName,
					},
					Spec: instanav1.InstanaAgentSpec{
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

				dsBuilder := &daemonSetBuilder{
					InstanaAgent: agent,
				}

				if test.hasZoneSet {
					dsBuilder.zone = zone
				}

				t.Run(
					"getName", func(t *testing.T) {
						actualName := dsBuilder.getName()
						assertions.Equal(test.expectedName, actualName)
					},
				)

				t.Run(
					"getNonStandardLabels", func(t *testing.T) {
						actualNonStandardLabels := dsBuilder.getNonStandardLabels()
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

						actualAffinity := dsBuilder.getAffinity()
						assertions.Same(expectedAffinity, actualAffinity)
					},
				)

				t.Run(
					"getTolerations", func(t *testing.T) {
						actualTolerations := dsBuilder.getTolerations()
						assertions.Equal(test.expectedTolerations, actualTolerations)
					},
				)
			},
		)
	}
}

func TestDaemonSetBuilder_Build(t *testing.T) {
	for _, test := range []struct {
		name          string
		agent         *instanav1.InstanaAgent
		expectPresent bool
	}{
		{
			name:          "should_be_not_present",
			agent:         &instanav1.InstanaAgent{},
			expectPresent: false,
		},
		{
			name: "should_be_present",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					Agent:   instanav1.BaseAgentSpec{Key: "key"},
					Cluster: instanav1.Name{Name: "cluster"},
				},
			},
			expectPresent: true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := assert.New(t)
				ctrl := gomock.NewController(t)

				status := mocks.NewMockAgentStatusManager(ctrl)
				if test.expectPresent {
					status.EXPECT().AddAgentDaemonset(gomock.Any())
				}

				dsBuilder := NewDaemonSetBuilder(test.agent, false, status)

				result := dsBuilder.Build()
				assertions.Equal(test.expectPresent, result.IsPresent())
			},
		)
	}
}
