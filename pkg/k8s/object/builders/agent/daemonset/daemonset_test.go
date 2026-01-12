/*
(c) Copyright IBM Corp. 2024, 2025
*/

package daemonset

import (
	"testing"

	"github.com/instana/instana-agent-operator/internal/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/apimachinery/pkg/util/rand"
	"k8s.io/client-go/tools/record"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/pointer"
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

				expected := map[string]string{
					"adsf":      "eroinsvd",
					"osdgoiego": "rwuriunsv",
					"e8uriunv":  "rrudsiu",
				}

				podSelector := &mocks.MockPodSelectorLabelGenerator{}
				defer podSelector.AssertExpectations(t)
				podSelector.On("GetPodLabels", test.getPodLabelsInput).Return(expected)

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

	envBuilder := &mocks.MockEnvBuilder{}
	defer envBuilder.AssertExpectations(t)
	envBuilder.On("Build",
		env.AgentModeEnv,
		env.ZoneNameEnv,
		env.ClusterNameEnv,
		env.AgentEndpointEnv,
		env.AgentEndpointPortEnv,
		env.MavenRepoURLEnv,
		env.MavenRepoFeaturesPath,
		env.MavenRepoSharedPath,
		env.MirrorReleaseRepoUrlEnv,
		env.MirrorReleaseRepoUsernameEnv,
		env.MirrorReleaseRepoPasswordEnv,
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
		env.NamespacesDetailsPathEnv,
	).Return(expected)

	// Create agent with no pod.env
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "some-agent"},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Pod: instanav1.AgentPodSpec{},
			},
		},
	}

	db := &daemonSetBuilder{
		EnvBuilder:   envBuilder,
		Helpers:      helpers.NewHelpers(agent),
		InstanaAgent: agent,
	}

	actual := db.getEnvVars()

	assertions.Equal(expected, actual)
}

func TestDaemonSetBuilder_getEnvVarsWithPodEnv(t *testing.T) {
	assertions := require.New(t)

	baseEnvVars := []corev1.EnvVar{
		{
			Name:  "foo",
			Value: "bar",
		},
		{
			Name:  "hello",
			Value: "world",
		},
	}

	podEnvVars := []corev1.EnvVar{
		{
			Name:  "TEST_ENV",
			Value: "test-value",
		},
		{
			Name: "TEST_ENV_FROM_FIELD",
			ValueFrom: &corev1.EnvVarSource{
				FieldRef: &corev1.ObjectFieldSelector{
					FieldPath: "metadata.name",
				},
			},
		},
	}

	expectedEnvVars := append(baseEnvVars, podEnvVars...)

	envBuilder := &mocks.MockEnvBuilder{}
	defer envBuilder.AssertExpectations(t)
	envBuilder.On("Build",
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
		mock.Anything,
	).Return(baseEnvVars)

	// Create agent with pod.env
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{Name: "some-agent"},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Pod: instanav1.AgentPodSpec{
					Env: podEnvVars,
				},
			},
		},
	}

	db := &daemonSetBuilder{
		EnvBuilder:   envBuilder,
		Helpers:      helpers.NewHelpers(agent),
		InstanaAgent: agent,
	}

	actual := db.getEnvVars()

	// Check that both base env vars and pod env vars are present
	assertions.Equal(len(expectedEnvVars), len(actual))

	// Check that pod env vars are present
	foundTestEnv := false
	foundTestEnvFromField := false

	for _, env := range actual {
		if env.Name == "TEST_ENV" {
			foundTestEnv = true
			assertions.Equal("test-value", env.Value)
		}
		if env.Name == "TEST_ENV_FROM_FIELD" {
			foundTestEnvFromField = true
			assertions.Equal("metadata.name", env.ValueFrom.FieldRef.FieldPath)
		}
	}

	assertions.True(foundTestEnv, "TEST_ENV not found in container environment variables")
	assertions.True(
		foundTestEnvFromField,
		"TEST_ENV_FROM_FIELD not found in container environment variables",
	)
}

func TestDaemonSetBuilder_getContainerPorts(t *testing.T) {
	assertions := require.New(t)

	expected := []corev1.ContainerPort{
		{
			Name:          "something",
			ContainerPort: 12345,
		},
	}

	portsBuilder := &mocks.MockPortsBuilder{}
	defer portsBuilder.AssertExpectations(t)
	portsBuilder.On("GetContainerPorts").Return(expected)

	db := &daemonSetBuilder{
		portsBuilder: portsBuilder,
	}

	actual := db.portsBuilder.GetContainerPorts()

	assertions.Equal(expected, actual)
}

func TestDaemonSetBuilder_getVolumes(t *testing.T) {
	for _, test := range []struct {
		name                 string
		useSecretMounts      *bool
		includeSecretsVolume bool
	}{
		{
			name:                 "with_secret_mounts_enabled",
			useSecretMounts:      pointer.To(true),
			includeSecretsVolume: true,
		},
		{
			name:                 "with_secret_mounts_disabled",
			useSecretMounts:      pointer.To(false),
			includeSecretsVolume: false,
		},
		{
			name:                 "with_secret_mounts_nil",
			useSecretMounts:      nil,
			includeSecretsVolume: false,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			assertions := require.New(t)

			expectedVolumes := []corev1.Volume{{Name: rand.String(10)}}
			expectedVolumeMounts := []corev1.VolumeMount{{Name: rand.String(10)}}

			volumeBuilder := &mocks.MockVolumeBuilder{}
			defer volumeBuilder.AssertExpectations(t)

			// Create the base volumes list
			baseVolumes := []interface{}{
				volume.DevVolume,
				volume.RunVolume,
				volume.VarRunVolume,
				volume.VarRunKuboVolume,
				volume.VarRunContainerdVolume,
				volume.VarContainerdConfigVolume,
				volume.SysVolume,
				volume.VarLogVolume,
				//volume.VarLibVolume,(Removed as part of CSP)
				volume.VarDataVolume,
				volume.MachineIdVolume,
				volume.ConfigVolume,
				volume.TlsVolume,
				volume.RepoVolume,
				volume.NamespacesDetailsVolume,
			}

			// Add SecretsVolume if it should be included
			if test.includeSecretsVolume {
				baseVolumes = append(baseVolumes, volume.SecretsVolume)
			}

			// Set up the mock expectation with the appropriate volumes
			volumeBuilder.On("Build", baseVolumes...).Return(expectedVolumes, expectedVolumeMounts)

			db := &daemonSetBuilder{
				VolumeBuilder: volumeBuilder,
				InstanaAgent: &instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						UseSecretMounts: test.useSecretMounts,
					},
				},
			}

			actualVolumes, actualVolumeMounts := db.getVolumes()

			assertions.Equal(expectedVolumes, actualVolumes)
			assertions.Equal(expectedVolumeMounts, actualVolumeMounts)
		})
	}
}

func TestDaemonSetBuilder_getUserVolumes(t *testing.T) {
	assertions := require.New(t)

	volumeName := "testVolume"
	expectedVolumes := []corev1.Volume{{Name: volumeName}}
	expectedVolumeMounts := []corev1.VolumeMount{{Name: volumeName}}

	volumeBuilder := &mocks.MockVolumeBuilder{}
	defer volumeBuilder.AssertExpectations(t)
	volumeBuilder.On("BuildFromUserConfig").Return(expectedVolumes, expectedVolumeMounts)

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

	dsBuilder := NewDaemonSetBuilder(&instanav1.InstanaAgent{}, false, nil)

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
			name: "should_be_not_present",

			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					OpenTelemetry: instanav1.OpenTelemetry{
						Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
					},
				},
			},
			expectPresent: false,
		},
		{
			name: "should_be_present",
			agent: &instanav1.InstanaAgent{
				Spec: instanav1.InstanaAgentSpec{
					Agent:   instanav1.BaseAgentSpec{Key: "key"},
					Cluster: instanav1.Name{Name: "cluster"},
					OpenTelemetry: instanav1.OpenTelemetry{
						Enabled: instanav1.Enabled{Enabled: pointer.To(false)},
					},
				},
			},
			expectPresent: true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := assert.New(t)

				status := &mocks.MockAgentStatusManager{}
				defer status.AssertExpectations(t)
				if test.expectPresent {
					status.On("AddAgentDaemonset", mock.Anything)
				}

				dsBuilder := NewDaemonSetBuilder(test.agent, false, status)

				result := dsBuilder.Build()
				assertions.Equal(test.expectPresent, result.IsPresent())
			},
		)
	}
}

func TestGetLivenessProbe_DefaultValues(t *testing.T) {
	// Create a minimal InstanaAgent without custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert default values
	require.NotNil(t, probe)
	assert.Equal(t, int32(600), probe.InitialDelaySeconds)
	assert.Equal(t, int32(5), probe.TimeoutSeconds)
	assert.Equal(t, int32(10), probe.PeriodSeconds)
	assert.Equal(t, int32(6), probe.FailureThreshold)

	// Assert HTTPGet configuration
	require.NotNil(t, probe.HTTPGet)
	assert.Equal(t, "127.0.0.1", probe.HTTPGet.Host)
	assert.Equal(t, "/status", probe.HTTPGet.Path)
	assert.Equal(t, intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port), probe.HTTPGet.Port)
}

func TestGetLivenessProbe_CustomValues(t *testing.T) {
	// Create a custom liveness probe
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/health",
				Port: intstr.FromInt32(42699),
			},
		},
		InitialDelaySeconds: 900,
		TimeoutSeconds:      10,
		PeriodSeconds:       15,
		FailureThreshold:    5,
	}

	// Create an InstanaAgent with custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert custom values
	require.NotNil(t, probe)
	assert.Equal(t, int32(900), probe.InitialDelaySeconds)
	assert.Equal(t, int32(10), probe.TimeoutSeconds)
	assert.Equal(t, int32(15), probe.PeriodSeconds)
	assert.Equal(t, int32(5), probe.FailureThreshold)

	// Assert custom HTTPGet configuration
	require.NotNil(t, probe.HTTPGet)
	assert.Equal(t, "127.0.0.1", probe.HTTPGet.Host)
	assert.Equal(t, "/health", probe.HTTPGet.Path)
	assert.Equal(t, intstr.FromInt32(42699), probe.HTTPGet.Port)
}

func TestGetLivenessProbe_PartialCustomValues(t *testing.T) {
	// Create a custom liveness probe with only some fields set
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/status",
				Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
			},
		},
		InitialDelaySeconds: 1200,
		// Other fields will use zero values
	}

	// Create an InstanaAgent with partial custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert that the custom probe is returned as-is
	require.NotNil(t, probe)
	assert.Equal(t, int32(1200), probe.InitialDelaySeconds)
	assert.Equal(t, int32(0), probe.TimeoutSeconds)   // Zero value since not set
	assert.Equal(t, int32(0), probe.PeriodSeconds)    // Zero value since not set
	assert.Equal(t, int32(0), probe.FailureThreshold) // Zero value since not set
}

// theoretically should not be used with the agent, but the kubernetes spec would allow to define it, so adding tests
func TestGetLivenessProbe_TCPSocket(t *testing.T) {
	// Create a custom liveness probe using TCP socket instead of HTTP
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt32(42699),
			},
		},
		InitialDelaySeconds: 300,
		TimeoutSeconds:      3,
		PeriodSeconds:       5,
		FailureThreshold:    2,
	}

	// Create an InstanaAgent with TCP socket liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert TCP socket configuration
	require.NotNil(t, probe)
	require.NotNil(t, probe.TCPSocket)
	assert.Nil(t, probe.HTTPGet) // HTTPGet should be nil when using TCPSocket
	assert.Equal(t, intstr.FromInt32(42699), probe.TCPSocket.Port)
	assert.Equal(t, int32(300), probe.InitialDelaySeconds)
	assert.Equal(t, int32(3), probe.TimeoutSeconds)
	assert.Equal(t, int32(5), probe.PeriodSeconds)
	assert.Equal(t, int32(2), probe.FailureThreshold)
}

// theoretically should not be used with the agent, but the kubernetes spec would allow to define it, so adding tests
func TestGetLivenessProbe_ExecAction(t *testing.T) {
	// Create a custom liveness probe using Exec action
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			Exec: &corev1.ExecAction{
				Command: []string{"/bin/sh", "-c", "curl -f http://localhost:42699/status"},
			},
		},
		InitialDelaySeconds: 450,
		TimeoutSeconds:      8,
		PeriodSeconds:       20,
		FailureThreshold:    4,
	}

	// Create an InstanaAgent with Exec liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert Exec configuration
	require.NotNil(t, probe)
	require.NotNil(t, probe.Exec)
	assert.Nil(t, probe.HTTPGet) // HTTPGet should be nil when using Exec
	assert.Equal(
		t,
		[]string{"/bin/sh", "-c", "curl -f http://localhost:42699/status"},
		probe.Exec.Command,
	)
	assert.Equal(t, int32(450), probe.InitialDelaySeconds)
	assert.Equal(t, int32(8), probe.TimeoutSeconds)
	assert.Equal(t, int32(20), probe.PeriodSeconds)
	assert.Equal(t, int32(4), probe.FailureThreshold)
}

func TestBuild_LivenessProbeInDaemonSet(t *testing.T) {
	// Create an InstanaAgent with custom liveness probe
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/status",
				Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
			},
		},
		InitialDelaySeconds: 800,
		TimeoutSeconds:      7,
		PeriodSeconds:       12,
		FailureThreshold:    4,
	}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Build the DaemonSet
	ds := builder.build()

	// Assert that the DaemonSet contains the custom liveness probe
	require.NotNil(t, ds)
	require.Len(t, ds.Spec.Template.Spec.Containers, 1)

	container := ds.Spec.Template.Spec.Containers[0]
	require.NotNil(t, container.LivenessProbe)
	assert.Equal(t, int32(800), container.LivenessProbe.InitialDelaySeconds)
	assert.Equal(t, int32(7), container.LivenessProbe.TimeoutSeconds)
	assert.Equal(t, int32(12), container.LivenessProbe.PeriodSeconds)
	assert.Equal(t, int32(4), container.LivenessProbe.FailureThreshold)
}

func TestBuild_DefaultLivenessProbeInDaemonSet(t *testing.T) {
	// Create an InstanaAgent without custom liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Build the DaemonSet
	ds := builder.build()

	// Assert that the DaemonSet contains the default liveness probe
	require.NotNil(t, ds)
	require.Len(t, ds.Spec.Template.Spec.Containers, 1)

	container := ds.Spec.Template.Spec.Containers[0]
	require.NotNil(t, container.LivenessProbe)
	assert.Equal(t, int32(600), container.LivenessProbe.InitialDelaySeconds)
	assert.Equal(t, int32(5), container.LivenessProbe.TimeoutSeconds)
	assert.Equal(t, int32(10), container.LivenessProbe.PeriodSeconds)
	assert.Equal(t, int32(6), container.LivenessProbe.FailureThreshold)
}

func TestGetLivenessProbe_NilPointer(t *testing.T) {
	// Create an InstanaAgent with explicitly nil liveness probe
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: nil,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert default values are returned
	require.NotNil(t, probe)
	assert.Equal(t, int32(600), probe.InitialDelaySeconds)
	assert.Equal(t, int32(5), probe.TimeoutSeconds)
	assert.Equal(t, int32(10), probe.PeriodSeconds)
	assert.Equal(t, int32(6), probe.FailureThreshold)
}

func TestGetLivenessProbe_WithSuccessThreshold(t *testing.T) {
	// Create a custom liveness probe with SuccessThreshold set
	customProbe := &corev1.Probe{
		ProbeHandler: corev1.ProbeHandler{
			HTTPGet: &corev1.HTTPGetAction{
				Host: "127.0.0.1",
				Path: "/status",
				Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
			},
		},
		InitialDelaySeconds: 600,
		TimeoutSeconds:      5,
		PeriodSeconds:       10,
		FailureThreshold:    3,
		SuccessThreshold:    2, // Custom success threshold
	}

	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				EndpointHost: "ingress-red-saas.instana.io",
				EndpointPort: "443",
				Key:          "test-key",
				Pod: instanav1.AgentPodSpec{
					LivenessProbe: customProbe,
				},
			},
			Cluster: instanav1.Name{
				Name: "test-cluster",
			},
		},
	}

	// Apply defaults
	agent.Default()

	// Create the builder
	mockClient := &mocks.MockInstanaAgentClient{}
	eventRecorder := record.NewFakeRecorder(10)
	statusManager := status.NewAgentStatusManager(mockClient, eventRecorder)
	builder := NewDaemonSetBuilder(agent, false, statusManager).(*daemonSetBuilder)

	// Get the liveness probe
	probe := builder.getLivenessProbe()

	// Assert that SuccessThreshold is preserved
	require.NotNil(t, probe)
	assert.Equal(t, int32(2), probe.SuccessThreshold)
}
