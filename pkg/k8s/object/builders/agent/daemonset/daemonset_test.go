package daemonset

import (
	"fmt"
	"testing"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
	"github.com/instana/instana-agent-operator/pkg/optional"
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

func TestDaemonSetBuilder_getPodTemplateAnnotations(t *testing.T) {
	const expectedHash = "49845soidghoijw09"

	for _, test := range []struct {
		name                    string
		userProvidedAnnotations map[string]string
		expected                map[string]string
	}{
		{
			name:                    "no_user_provided_annotations",
			userProvidedAnnotations: nil,
			expected: map[string]string{
				"instana-configuration-hash": expectedHash,
			},
		},
		{
			name: "with_user_provided_annotations",
			userProvidedAnnotations: map[string]string{
				"498hroihsg":             "4589fdoighjsoijs",
				"flkje489h309sd":         "oie409ojifg",
				"4509ufdoigjselkjweoihg": "g059pojw9jwpoijd",
			},
			expected: map[string]string{
				"instana-configuration-hash": expectedHash,
				"498hroihsg":                 "4589fdoighjsoijs",
				"flkje489h309sd":             "oie409ojifg",
				"4509ufdoigjselkjweoihg":     "g059pojw9jwpoijd",
			},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)
				ctrl := gomock.NewController(t)

				agent := instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Pod: instanav1.AgentPodSpec{
								Annotations: test.userProvidedAnnotations,
							},
						},
					},
				}

				hasher := NewMockJsonHasher(ctrl)
				hasher.EXPECT().HashJsonOrDie(gomock.Eq(&agent.Spec)).Return(expectedHash)

				db := &daemonSetBuilder{
					InstanaAgent: &agent,
					JsonHasher:   hasher,
				}

				actual := db.getPodTemplateAnnotations()
				assertions.Equal(test.expected, actual)
			},
		)
	}
}

func TestDaemonSetBuilder_getImagePullSecrets(t *testing.T) {
	testCases := []struct {
		name             string
		instanaAgentSpec instanav1.InstanaAgentSpec
		expectedSecrets  []corev1.LocalObjectReference
	}{
		{
			name: "no_user_secrets_and_image_not_from_instana_io",
			instanaAgentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ImageSpec: instanav1.ImageSpec{},
				},
			},
			expectedSecrets: nil,
		},
		{
			name: "with_user_secrets_and_image_not_from_instana_io",
			instanaAgentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ImageSpec: instanav1.ImageSpec{
						PullSecrets: []corev1.LocalObjectReference{
							{
								Name: "oirewigojsdf",
							},
							{
								Name: "o4gpoijsfd",
							},
							{
								Name: "po5hpojdfijs",
							},
						},
					},
				},
			},
			expectedSecrets: []corev1.LocalObjectReference{
				{
					Name: "oirewigojsdf",
				},
				{
					Name: "o4gpoijsfd",
				},
				{
					Name: "po5hpojdfijs",
				},
			},
		},
		{
			name: "no_user_secrets_and_image_is_from_instana_io",
			instanaAgentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ImageSpec: instanav1.ImageSpec{
						Name: "containers.instana.io/instana-agent",
					},
				},
			},
			expectedSecrets: []corev1.LocalObjectReference{
				{
					Name: "containers-instana-io",
				},
			},
		},
		{
			name: "with_user_secrets_and_image_is_from_instana_io",
			instanaAgentSpec: instanav1.InstanaAgentSpec{
				Agent: instanav1.BaseAgentSpec{
					ImageSpec: instanav1.ImageSpec{
						Name: "containers.instana.io/instana-agent",
						PullSecrets: []corev1.LocalObjectReference{
							{
								Name: "oirewigojsdf",
							},
							{
								Name: "o4gpoijsfd",
							},
							{
								Name: "po5hpojdfijs",
							},
						},
					},
				},
			},
			expectedSecrets: []corev1.LocalObjectReference{
				{
					Name: "oirewigojsdf",
				},
				{
					Name: "o4gpoijsfd",
				},
				{
					Name: "po5hpojdfijs",
				},
				{
					Name: "containers-instana-io",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				db := &daemonSetBuilder{
					InstanaAgent: &instanav1.InstanaAgent{
						Spec: tc.instanaAgentSpec,
					},
				}

				actualSecrets := db.getImagePullSecrets()

				assertions.Equal(tc.expectedSecrets, actualSecrets)
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

func TestDaemonSetBuilder_getResourceRequirements(t *testing.T) {
	metaAssertions := require.New(t)

	type testParams struct {
		providedMemRequest string
		providedCpuRequest string
		providedMemLimit   string
		providedCpuLimit   string

		expectedMemRequest string
		expectedCpuRequest string
		expectedMemLimit   string
		expectedCpuLimit   string
	}

	tests := make([]testParams, 0, 16)
	for _, providedMemRequest := range []string{"", "123Mi"} {
		for _, providedCpuRequest := range []string{"", "1.2"} {
			for _, providedMemLimit := range []string{"", "456Mi"} {
				for _, providedCpuLimit := range []string{"", "4.5"} {
					tests = append(
						tests, testParams{
							expectedMemRequest: optional.Of(providedMemRequest).GetOrDefault("512Mi"),
							expectedCpuRequest: optional.Of(providedCpuRequest).GetOrDefault("0.5"),
							expectedMemLimit:   optional.Of(providedMemLimit).GetOrDefault("768Mi"),
							expectedCpuLimit:   optional.Of(providedCpuLimit).GetOrDefault("1.5"),

							providedMemRequest: providedMemRequest,
							providedCpuRequest: providedCpuRequest,
							providedMemLimit:   providedMemLimit,
							providedCpuLimit:   providedCpuLimit,
						},
					)
				}
			}
		}
	}

	metaAssertions.Len(tests, 16)

	for _, test := range tests {
		t.Run(
			fmt.Sprintf("%+v", test), func(t *testing.T) {
				assertions := require.New(t)

				provided := corev1.ResourceRequirements{}

				setIfNotEmpty := func(providedVal string, key corev1.ResourceName, resourceList *corev1.ResourceList) {
					if providedVal != "" {
						map_defaulter.NewMapDefaulter((*map[corev1.ResourceName]resource.Quantity)(resourceList)).SetIfEmpty(
							key,
							resource.MustParse(providedVal),
						)
					}
				}

				setIfNotEmpty(test.providedMemLimit, corev1.ResourceMemory, &provided.Limits)
				setIfNotEmpty(test.providedCpuLimit, corev1.ResourceCPU, &provided.Limits)
				setIfNotEmpty(test.providedMemRequest, corev1.ResourceMemory, &provided.Requests)
				setIfNotEmpty(test.providedCpuRequest, corev1.ResourceCPU, &provided.Requests)

				db := &daemonSetBuilder{
					InstanaAgent: &instanav1.InstanaAgent{
						Spec: instanav1.InstanaAgentSpec{
							Agent: instanav1.BaseAgentSpec{
								Pod: instanav1.AgentPodSpec{
									ResourceRequirements: provided,
								},
							},
						},
					},
				}
				actual := db.getResourceRequirements()

				assertions.Equal(resource.MustParse(test.expectedMemLimit), actual.Limits[corev1.ResourceMemory])
				assertions.Equal(resource.MustParse(test.expectedCpuLimit), actual.Limits[corev1.ResourceCPU])
				assertions.Equal(resource.MustParse(test.expectedMemRequest), actual.Requests[corev1.ResourceMemory])
				assertions.Equal(resource.MustParse(test.expectedCpuRequest), actual.Requests[corev1.ResourceCPU])
			},
		)
	}
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
