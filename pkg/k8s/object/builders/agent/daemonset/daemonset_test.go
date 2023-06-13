package daemonset

import (
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
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

				podSelector := NewMockPodSelectorLabelGenerator(ctrl)
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

	envBuilder := NewMockEnvBuilder(ctrl)
	envBuilder.EXPECT().Build(
		env.AgentModeEnv,
		env.ZoneNameEnv,
		env.ClusterNameEnv,
		env.AgentEndpointEnv,
		env.AgentEndpointPortEnv,
		env.MavenRepoURLEnv,
		env.ProxyHostEnv,
		env.ProxyPortEnv,
		env.ProxyProtocolEnv,
		env.ProxyUserEnv,
		env.ProxyPasswordEnv,
		env.ProxyUseDNSEnv,
		env.ListenAddressEnv,
		env.RedactK8sSecretsEnv,
		env.AgentKeyEnv,
		env.DownloadKeyEnv,
		env.PodNameEnv,
		env.PodIPEnv,
		env.K8sServiceDomainEnv,
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

	portsBuilder := NewMockPortsBuilder(ctrl)
	portsBuilder.EXPECT().GetContainerPorts(
		ports.AgentAPIsPort,
		ports.AgentSocketPort,
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

func TestDaemonSetBuilder_getInitContainerVolumeMounts(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expected := []corev1.VolumeMount{{Name: rand.String(10)}}

	volumeBuilder := NewMockVolumeBuilder(ctrl)
	volumeBuilder.EXPECT().Build(gomock.Eq(volume.TPLFilesTmpVolume)).Return(nil, expected)

	db := &daemonSetBuilder{
		VolumeBuilder: volumeBuilder,
	}

	actual := db.getInitContainerVolumeMounts()

	assertions.Equal(expected, actual)
}

func TestDaemonSetBuilder_getVolumes(t *testing.T) {
	assertions := require.New(t)
	ctrl := gomock.NewController(t)

	expectedVolumes := []corev1.Volume{{Name: rand.String(10)}}
	expectedVolumeMounts := []corev1.VolumeMount{{Name: rand.String(10)}}

	volumeBuilder := NewMockVolumeBuilder(ctrl)
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
		gomock.Eq(volume.TPLFilesTmpVolume),
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
