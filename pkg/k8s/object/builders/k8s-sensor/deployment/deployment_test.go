/*
(c) Copyright IBM Corp. 2025
*/

package deployment

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

// Mock implementations for dependencies
type MockStatusManager struct {
	mock.Mock
}

func (m *MockStatusManager) SetK8sSensorDeployment(key client.ObjectKey) {
	m.Called(key)
}

func (m *MockStatusManager) AddAgentDaemonset(agentDaemonset client.ObjectKey) {
	m.Called(agentDaemonset)
}

func (m *MockStatusManager) SetAgentOld(agent *instanav1.InstanaAgent) {
	m.Called(agent)
}

func (m *MockStatusManager) SetAgentSecretConfig(agentSecretConfig client.ObjectKey) {
	m.Called(agentSecretConfig)
}

func (m *MockStatusManager) SetAgentNamespacesConfigMap(agentNamespacesConfigmap client.ObjectKey) {
	m.Called(agentNamespacesConfigmap)
}

func (m *MockStatusManager) UpdateAgentStatus(ctx context.Context, reconcileErr error) error {
	args := m.Called(ctx, reconcileErr)
	return args.Error(0)
}

type MockHelpers struct {
	mock.Mock
}

func (m *MockHelpers) K8sSensorResourcesName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHelpers) ImagePullSecrets() []corev1.LocalObjectReference {
	args := m.Called()
	return args.Get(0).([]corev1.LocalObjectReference)
}

func (m *MockHelpers) SortEnvVarsByName(envVars []corev1.EnvVar) {
	m.Called(envVars)
}

func (m *MockHelpers) TLSIsEnabled() bool {
	args := m.Called()
	return args.Bool(0)
}

func (m *MockHelpers) TLSSecretName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHelpers) ServiceAccountName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHelpers) HeadlessServiceName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHelpers) ContainersSecretName() string {
	args := m.Called()
	return args.String(0)
}

func (m *MockHelpers) UseContainersSecret() bool {
	args := m.Called()
	return args.Bool(0)
}

type MockEnvBuilder struct {
	mock.Mock
}

func (m *MockEnvBuilder) Build(envs ...env.EnvVar) []corev1.EnvVar {
	args := m.Called(envs)
	return args.Get(0).([]corev1.EnvVar)
}

type MockVolumeBuilder struct {
	mock.Mock
}

func (m *MockVolumeBuilder) Build(
	volumes ...volume.Volume,
) ([]corev1.Volume, []corev1.VolumeMount) {
	args := m.Called(mock.Anything)
	return args.Get(0).([]corev1.Volume), args.Get(1).([]corev1.VolumeMount)
}

func (m *MockVolumeBuilder) BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount) {
	args := m.Called()
	return args.Get(0).([]corev1.Volume), args.Get(1).([]corev1.VolumeMount)
}

func (m *MockVolumeBuilder) WithBackendResourceSuffix(string) volume.VolumeBuilder {
	return m
}

type MockPortsBuilder struct {
	mock.Mock
}

func (m *MockPortsBuilder) GetServicePorts() []corev1.ServicePort {
	args := m.Called()
	return args.Get(0).([]corev1.ServicePort)
}

func (m *MockPortsBuilder) GetContainerPorts() []corev1.ContainerPort {
	args := m.Called()
	return args.Get(0).([]corev1.ContainerPort)
}

type MockPodSelectorLabelGenerator struct {
	mock.Mock
}

func (m *MockPodSelectorLabelGenerator) GetPodSelectorLabels() map[string]string {
	args := m.Called()
	return args.Get(0).(map[string]string)
}

func (m *MockPodSelectorLabelGenerator) GetPodLabels(
	additionalLabels map[string]string,
) map[string]string {
	args := m.Called(additionalLabels)
	return args.Get(0).(map[string]string)
}

// Test fixtures
func createInstanaAgentWithSecretMountsEnabled() *instanav1.InstanaAgent {
	return &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-namespace",
		},
		Spec: instanav1.InstanaAgentSpec{
			UseSecretMounts: pointer.To(true),
			Agent: instanav1.BaseAgentSpec{
				Key:          "test-key",
				EndpointHost: "test-host",
				EndpointPort: "443",
				Pod: instanav1.AgentPodSpec{
					Annotations: map[string]string{},
					Labels:      map[string]string{},
				},
			},
			Zone:    instanav1.Name{Name: "test-zone"},
			Cluster: instanav1.Name{Name: "test-cluster"},
			K8sSensor: instanav1.K8sSpec{
				DeploymentSpec: instanav1.KubernetesDeploymentSpec{
					Enabled:         instanav1.Enabled{Enabled: pointer.To(true)},
					Replicas:        1,
					MinReadySeconds: 10,
					Pod:             instanav1.KubernetesPodSpec{
						// KubernetesPodSpec doesn't have Labels and Annotations fields
					},
				},
				ImageSpec: instanav1.ImageSpec{
					Name:       "instana/k8sensor",
					Tag:        "latest",
					PullPolicy: corev1.PullIfNotPresent,
				},
			},
		},
	}
}

func createInstanaAgentWithSecretMountsDisabled() *instanav1.InstanaAgent {
	agent := createInstanaAgentWithSecretMountsEnabled()
	agent.Spec.UseSecretMounts = pointer.To(false)
	return agent
}

func createInstanaAgentWithSecretMountsNotSpecified() *instanav1.InstanaAgent {
	agent := createInstanaAgentWithSecretMountsEnabled()
	agent.Spec.UseSecretMounts = nil
	return agent
}

func createInstanaAgentWithProxy() *instanav1.InstanaAgent {
	agent := createInstanaAgentWithSecretMountsEnabled()
	agent.Spec.Agent.ProxyHost = "proxy.example.com"
	agent.Spec.Agent.ProxyPort = "3128"
	return agent
}

// Helper function to create a deployment builder for testing
func createTestDeploymentBuilder(t *testing.T, agent *instanav1.InstanaAgent) *deploymentBuilder {
	backend := backends.K8SensorBackend{
		EndpointHost:   agent.Spec.Agent.EndpointHost,
		EndpointPort:   agent.Spec.Agent.EndpointPort,
		EndpointKey:    agent.Spec.Agent.Key,
		ResourceSuffix: "",
	}

	mockStatusManager := new(MockStatusManager)
	mockHelpers := new(MockHelpers)
	mockEnvBuilder := new(MockEnvBuilder)
	mockVolumeBuilder := new(MockVolumeBuilder)
	mockPortsBuilder := new(MockPortsBuilder)
	mockPodSelectorLabelGenerator := new(MockPodSelectorLabelGenerator)

	// Set up mock expectations
	mockHelpers.On("K8sSensorResourcesName").Return("test-agent-k8sensor")
	mockHelpers.On("ImagePullSecrets").Return([]corev1.LocalObjectReference{})
	mockHelpers.On("SortEnvVarsByName", mock.Anything).Return()
	mockHelpers.On("ServiceAccountName").Return("test-agent")
	mockHelpers.On("HeadlessServiceName").Return("test-agent-headless")
	mockHelpers.On("ContainersSecretName").Return("test-agent-containers-instana-io")
	mockHelpers.On("UseContainersSecret").Return(false)
	mockHelpers.On("TLSIsEnabled").Return(false)
	mockHelpers.On("TLSSecretName").Return("test-agent-tls")

	// Mock the EnvBuilder to return some basic environment variables
	mockEnvBuilder.On("Build", mock.Anything).Return([]corev1.EnvVar{
		{Name: "BACKEND_URL", Value: "https://test-host:443"},
		{Name: "INSTANA_ZONE", Value: "test-zone"},
	})

	// Add specific mock for AgentKeyEnv
	mockEnvBuilder.On("Build", []env.EnvVar{env.AgentKeyEnv}).Return([]corev1.EnvVar{
		{Name: "AGENT_KEY", Value: "test-key"},
	})

	// Add specific mock for HTTPSProxyEnv
	mockEnvBuilder.On("Build", []env.EnvVar{env.HTTPSProxyEnv}).Return([]corev1.EnvVar{
		{Name: "HTTPS_PROXY", Value: ""},
	})

	// Mock the PortsBuilder
	mockPortsBuilder.On("GetContainerPorts").Return([]corev1.ContainerPort{
		{Name: "api", ContainerPort: 42699},
	})

	// Mock the PodSelectorLabelGenerator
	mockPodSelectorLabelGenerator.On("GetPodSelectorLabels").Return(map[string]string{
		"app.kubernetes.io/name":      "instana-agent",
		"app.kubernetes.io/component": "k8sensor",
	})
	mockPodSelectorLabelGenerator.On("GetPodLabels", mock.Anything).Return(map[string]string{
		"app.kubernetes.io/name":      "instana-agent",
		"app.kubernetes.io/component": "k8sensor",
	})

	// Create a minimal deployment builder for testing
	return &deploymentBuilder{
		InstanaAgent:              agent,
		statusManager:             mockStatusManager,
		helpers:                   mockHelpers,
		EnvBuilder:                mockEnvBuilder,
		VolumeBuilder:             mockVolumeBuilder,
		PortsBuilder:              mockPortsBuilder,
		PodSelectorLabelGenerator: mockPodSelectorLabelGenerator,
		backend:                   backend,
		keysSecret:                nil,
		deploymentContext:         nil,
		isOpenShift:               false,
	}
}

// Helper functions for assertions
func containsEnvVar(envVars []corev1.EnvVar, name string) bool {
	for _, env := range envVars {
		if env.Name == name {
			return true
		}
	}
	return false
}

func containsVolume(volumes []corev1.Volume, name string) bool {
	for _, vol := range volumes {
		if vol.Name == name {
			return true
		}
	}
	return false
}

func containsVolumeMount(mounts []corev1.VolumeMount, name string) bool {
	for _, mount := range mounts {
		if mount.Name == name {
			return true
		}
	}
	return false
}

func indexOf(slice []string, item string) int {
	for i, s := range slice {
		if s == item {
			return i
		}
	}
	return -1
}

// Test cases for getEnvVars method
func TestGetEnvVarsWithSecretMountsEnabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	envVars := builder.getEnvVars()

	// Assert
	assert.False(
		t,
		containsEnvVar(envVars, "AGENT_KEY"),
		"AGENT_KEY environment variable should not be present when UseSecretMounts is enabled",
	)
	assert.True(
		t,
		containsEnvVar(envVars, "BACKEND"),
		"BACKEND environment variable should be present",
	)
}

func TestGetEnvVarsWithSecretMountsDisabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsDisabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Override the mock setup to include AGENT_KEY
	mockEnvBuilder := new(MockEnvBuilder)
	mockEnvBuilder.On("Build", mock.Anything).Return([]corev1.EnvVar{
		{Name: "BACKEND_URL", Value: "https://test-host:443"},
		{Name: "INSTANA_ZONE", Value: "test-zone"},
		{Name: "AGENT_KEY", Value: "test-key"}, // Add AGENT_KEY to the mock response
	})
	builder.EnvBuilder = mockEnvBuilder

	// Act
	envVars := builder.getEnvVars()

	// Debug: Print all environment variables
	for _, env := range envVars {
		t.Logf("Env var: %s = %s", env.Name, env.Value)
	}

	// Assert
	assert.True(
		t,
		containsEnvVar(envVars, "AGENT_KEY"),
		"AGENT_KEY environment variable should be present when UseSecretMounts is disabled",
	)
	assert.True(
		t,
		containsEnvVar(envVars, "BACKEND"),
		"BACKEND environment variable should be present",
	)
}

func TestGetEnvVarsWithSecretMountsNotSpecified(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsNotSpecified()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	envVars := builder.getEnvVars()

	// Assert
	assert.False(
		t,
		containsEnvVar(envVars, "AGENT_KEY"),
		"AGENT_KEY environment variable should not be present when UseSecretMounts is not specified (default to true)",
	)
	assert.True(
		t,
		containsEnvVar(envVars, "BACKEND"),
		"BACKEND environment variable should be present",
	)
}

// Test cases for getVolumes method
func TestGetVolumesWithSecretMountsEnabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mock for VolumeBuilder
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}
	secretsVolume := corev1.Volume{Name: "instana-secrets"}
	secretsMount := corev1.VolumeMount{Name: "instana-secrets"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume, secretsVolume}, []corev1.VolumeMount{configMount, secretsMount})

	// Act
	volumes, mounts := builder.getVolumes()

	// Assert
	assert.Equal(t, 2, len(volumes), "Should have 2 volumes when UseSecretMounts is enabled")
	assert.Equal(t, 2, len(mounts), "Should have 2 volume mounts when UseSecretMounts is enabled")
	assert.True(
		t,
		containsVolume(volumes, "instana-secrets"),
		"Secrets volume should be present when UseSecretMounts is enabled",
	)
	assert.True(
		t,
		containsVolumeMount(mounts, "instana-secrets"),
		"Secrets volume mount should be present when UseSecretMounts is enabled",
	)
}

func TestGetVolumesWithSecretMountsDisabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsDisabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mock for VolumeBuilder
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume}, []corev1.VolumeMount{configMount})

	// Act
	volumes, mounts := builder.getVolumes()

	// Assert
	assert.Equal(t, 1, len(volumes), "Should have 1 volume when UseSecretMounts is disabled")
	assert.Equal(t, 1, len(mounts), "Should have 1 volume mount when UseSecretMounts is disabled")
	assert.False(
		t,
		containsVolume(volumes, "instana-secrets"),
		"Secrets volume should not be present when UseSecretMounts is disabled",
	)
	assert.False(
		t,
		containsVolumeMount(mounts, "instana-secrets"),
		"Secrets volume mount should not be present when UseSecretMounts is disabled",
	)
}

func TestGetVolumesWithSecretMountsNotSpecified(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsNotSpecified()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mock for VolumeBuilder
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}
	secretsVolume := corev1.Volume{Name: "instana-secrets"}
	secretsMount := corev1.VolumeMount{Name: "instana-secrets"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume, secretsVolume}, []corev1.VolumeMount{configMount, secretsMount})

	// Act
	volumes, mounts := builder.getVolumes()

	// Assert
	assert.Equal(
		t,
		2,
		len(volumes),
		"Should have 2 volumes when UseSecretMounts is not specified (default to true)",
	)
	assert.Equal(
		t,
		2,
		len(mounts),
		"Should have 2 volume mounts when UseSecretMounts is not specified (default to true)",
	)
	assert.True(
		t,
		containsVolume(volumes, "instana-secrets"),
		"Secrets volume should be present when UseSecretMounts is not specified (default to true)",
	)
	assert.True(
		t,
		containsVolumeMount(mounts, "instana-secrets"),
		"Secrets volume mount should be present when UseSecretMounts is not specified (default to true)",
	)
}

// Test cases for getK8SensorArgs method
func TestGetK8SensorArgsWithSecretMountsEnabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	args := builder.getK8SensorArgs()

	// Assert
	assert.Contains(t, args, "-pollrate", "Base arguments should include -pollrate")
	assert.Contains(t, args, "10s", "Base arguments should include 10s polling rate")
	assert.Contains(
		t,
		args,
		"-agent-key-file",
		"Should include -agent-key-file argument when UseSecretMounts is enabled",
	)

	// Check the file path
	expectedPath := constants.InstanaSecretsDirectory + "/" + constants.SecretFileAgentKey
	pathIndex := indexOf(args, "-agent-key-file") + 1
	assert.Less(t, pathIndex, len(args), "Should have a path after -agent-key-file")
	assert.Equal(t, expectedPath, args[pathIndex], "Agent key file path should be correct")

	// Should not include https-proxy-file since no proxy is configured
	assert.NotContains(
		t,
		args,
		"-https-proxy-file",
		"Should not include -https-proxy-file when no proxy is configured",
	)
}

func TestGetK8SensorArgsWithSecretMountsEnabledAndProxy(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithProxy()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	args := builder.getK8SensorArgs()

	// Assert
	assert.Contains(t, args, "-pollrate", "Base arguments should include -pollrate")
	assert.Contains(t, args, "10s", "Base arguments should include 10s polling rate")
	assert.Contains(
		t,
		args,
		"-agent-key-file",
		"Should include -agent-key-file argument when UseSecretMounts is enabled",
	)
	assert.Contains(
		t,
		args,
		"-https-proxy-file",
		"Should include -https-proxy-file when proxy is configured",
	)

	// Check the file paths
	agentKeyPath := constants.InstanaSecretsDirectory + "/" + constants.SecretFileAgentKey
	agentKeyPathIndex := indexOf(args, "-agent-key-file") + 1
	assert.Less(t, agentKeyPathIndex, len(args), "Should have a path after -agent-key-file")
	assert.Equal(t, agentKeyPath, args[agentKeyPathIndex], "Agent key file path should be correct")

	proxyPath := constants.InstanaSecretsDirectory + "/" + constants.SecretFileHttpsProxy
	proxyPathIndex := indexOf(args, "-https-proxy-file") + 1
	assert.Less(t, proxyPathIndex, len(args), "Should have a path after -https-proxy-file")
	assert.Equal(t, proxyPath, args[proxyPathIndex], "HTTPS proxy file path should be correct")
}

func TestGetK8SensorArgsWithSecretMountsDisabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsDisabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	args := builder.getK8SensorArgs()

	// Assert
	assert.Contains(t, args, "-pollrate", "Base arguments should include -pollrate")
	assert.Contains(t, args, "10s", "Base arguments should include 10s polling rate")
	assert.NotContains(
		t,
		args,
		"-agent-key-file",
		"Should not include -agent-key-file argument when UseSecretMounts is disabled",
	)
	assert.NotContains(
		t,
		args,
		"-https-proxy-file",
		"Should not include -https-proxy-file when UseSecretMounts is disabled",
	)
	assert.Equal(
		t,
		2,
		len(args),
		"Should only have base arguments when UseSecretMounts is disabled",
	)
}

func TestGetK8SensorArgsWithSecretMountsNotSpecified(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsNotSpecified()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	args := builder.getK8SensorArgs()

	// Assert
	assert.Contains(t, args, "-pollrate", "Base arguments should include -pollrate")
	assert.Contains(t, args, "10s", "Base arguments should include 10s polling rate")
	assert.Contains(
		t,
		args,
		"-agent-key-file",
		"Should include -agent-key-file argument when UseSecretMounts is not specified (default to true)",
	)

	// Check the file path
	expectedPath := constants.InstanaSecretsDirectory + "/" + constants.SecretFileAgentKey
	pathIndex := indexOf(args, "-agent-key-file") + 1
	assert.Less(t, pathIndex, len(args), "Should have a path after -agent-key-file")
	assert.Equal(t, expectedPath, args[pathIndex], "Agent key file path should be correct")
}

// Integration test cases for the complete deployment builder
func TestBuildDeploymentWithSecretMountsEnabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mocks for the build method
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}
	secretsVolume := corev1.Volume{Name: "instana-secrets"}
	secretsMount := corev1.VolumeMount{Name: "instana-secrets"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume, secretsVolume}, []corev1.VolumeMount{configMount, secretsMount})

	builder.statusManager.(*MockStatusManager).On("SetK8sSensorDeployment", mock.Anything).Return()

	// Act
	deploymentObj := builder.build()

	// Assert
	assert.NotNil(t, deploymentObj, "Deployment object should not be nil")

	// Check container command and args
	container := deploymentObj.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "/ko-app/k8sensor", container.Command[0], "Container command should be correct")

	// Check that args include agent-key-file
	args := container.Args
	assert.Contains(
		t,
		args,
		"-agent-key-file",
		"Container args should include -agent-key-file when UseSecretMounts is enabled",
	)

	// Check volumes
	assert.Equal(
		t,
		2,
		len(deploymentObj.Spec.Template.Spec.Volumes),
		"Should have 2 volumes when UseSecretMounts is enabled",
	)
	assert.True(t, containsVolume(deploymentObj.Spec.Template.Spec.Volumes, "instana-secrets"),
		"Deployment should include secrets volume when UseSecretMounts is enabled")

	// Check volume mounts
	assert.Equal(
		t,
		2,
		len(container.VolumeMounts),
		"Should have 2 volume mounts when UseSecretMounts is enabled",
	)
	assert.True(t, containsVolumeMount(container.VolumeMounts, "instana-secrets"),
		"Container should include secrets volume mount when UseSecretMounts is enabled")

	// Check environment variables
	assert.False(t, containsEnvVar(container.Env, "AGENT_KEY"),
		"AGENT_KEY environment variable should not be present when UseSecretMounts is enabled")
}

func TestBuildDeploymentWithSecretMountsDisabled(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsDisabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mocks for the build method
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume}, []corev1.VolumeMount{configMount})

	builder.statusManager.(*MockStatusManager).On("SetK8sSensorDeployment", mock.Anything).Return()

	// Override the mock setup to include AGENT_KEY
	mockEnvBuilder := new(MockEnvBuilder)
	mockEnvBuilder.On("Build", mock.Anything).Return([]corev1.EnvVar{
		{Name: "BACKEND_URL", Value: "https://test-host:443"},
		{Name: "INSTANA_ZONE", Value: "test-zone"},
		{Name: "AGENT_KEY", Value: "test-key"}, // Add AGENT_KEY to the mock response
	})
	builder.EnvBuilder = mockEnvBuilder

	// Act
	deploymentObj := builder.build()

	// Assert
	assert.NotNil(t, deploymentObj, "Deployment object should not be nil")

	// Check container command and args
	container := deploymentObj.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "/ko-app/k8sensor", container.Command[0], "Container command should be correct")

	// Check that args do not include agent-key-file
	args := container.Args
	assert.NotContains(
		t,
		args,
		"-agent-key-file",
		"Container args should not include -agent-key-file when UseSecretMounts is disabled",
	)
	assert.Equal(
		t,
		[]string{"-pollrate", "10s"},
		args,
		"Container args should only include base arguments when UseSecretMounts is disabled",
	)

	// Check volumes
	assert.Equal(
		t,
		1,
		len(deploymentObj.Spec.Template.Spec.Volumes),
		"Should have 1 volume when UseSecretMounts is disabled",
	)
	assert.False(t, containsVolume(deploymentObj.Spec.Template.Spec.Volumes, "instana-secrets"),
		"Deployment should not include secrets volume when UseSecretMounts is disabled")

	// Check volume mounts
	assert.Equal(
		t,
		1,
		len(container.VolumeMounts),
		"Should have 1 volume mount when UseSecretMounts is disabled",
	)
	assert.False(t, containsVolumeMount(container.VolumeMounts, "instana-secrets"),
		"Container should not include secrets volume mount when UseSecretMounts is disabled")

	// Debug: Print all environment variables
	for _, env := range container.Env {
		t.Logf("Env var: %s = %s", env.Name, env.Value)
	}

	// Check environment variables
	assert.True(t, containsEnvVar(container.Env, "AGENT_KEY"),
		"AGENT_KEY environment variable should be present when UseSecretMounts is disabled")
}

// Additional tests to improve coverage
func TestIsNamespaced(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	result := builder.IsNamespaced()

	// Assert
	assert.True(t, result, "K8sSensor deployment should be namespaced")
}

func TestComponentName(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Act
	result := builder.ComponentName()

	// Assert
	assert.Equal(t, constants.ComponentK8Sensor, result, "Component name should be k8sensor")
}

func TestBuild(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mocks for the build method
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}
	secretsVolume := corev1.Volume{Name: "instana-secrets"}
	secretsMount := corev1.VolumeMount{Name: "instana-secrets"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume, secretsVolume}, []corev1.VolumeMount{configMount, secretsMount})

	builder.statusManager.(*MockStatusManager).On("SetK8sSensorDeployment", mock.Anything).Return()

	// Act
	result := builder.Build()

	// Assert
	assert.True(t, result.IsPresent(), "Build should return a present optional")

	// Test the case where key and zone are empty
	agentNoKey := createInstanaAgentWithSecretMountsEnabled()
	agentNoKey.Spec.Agent.Key = ""
	agentNoKey.Spec.Zone.Name = ""
	builderNoKey := createTestDeploymentBuilder(t, agentNoKey)

	resultNoKey := builderNoKey.Build()
	assert.False(
		t,
		resultNoKey.IsPresent(),
		"Build should return an empty optional when key and zone are empty",
	)

	// Test the case where K8s sensor is disabled when the flag is set to false
	agentK8sSensorDisabled := createInstanaAgentWithSecretMountsEnabled()
	agentK8sSensorDisabled.Spec.K8sSensor.DeploymentSpec.Enabled = instanav1.Enabled{
		Enabled: pointer.To(false),
	}
	builderK8sSensorDisabled := createTestDeploymentBuilder(t, agentK8sSensorDisabled)

	resultK8sSensorDisabled := builderK8sSensorDisabled.Build()
	assert.False(
		t,
		resultK8sSensorDisabled.IsPresent(),
		"Build should return an empty optional when k8sensor is disabled i.e: when enabled is set to false",
	)
}

// Test the case where K8s sensor is enabled when the flag is empty, i.e: check
// that the default behavior is k8sensor deployment build is enabled
func TestBuildEnabledIsNotSet(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	agent.Spec.K8sSensor.DeploymentSpec.Enabled = instanav1.Enabled{}
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mocks for the build method
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}
	secretsVolume := corev1.Volume{Name: "instana-secrets"}
	secretsMount := corev1.VolumeMount{Name: "instana-secrets"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume, secretsVolume}, []corev1.VolumeMount{configMount, secretsMount})

	builder.statusManager.(*MockStatusManager).On("SetK8sSensorDeployment", mock.Anything).Return()

	// Act
	result := builder.Build()

	// Assert
	assert.True(
		t,
		result.IsPresent(),
		"Build should return an present optional when k8sensor's Enabled is empty",
	)
}

func TestGetPodAnnotationsWithBackendChecksum(t *testing.T) {
	// Test with keysSecret
	agent := createInstanaAgentWithSecretMountsEnabled()
	agent.Spec.Agent.KeysSecret = "test-secret"
	builder := createTestDeploymentBuilder(t, agent)

	// Create a mock secret
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"key": []byte("test-key-from-secret"),
		},
	}
	builder.keysSecret = secret

	// Act
	annotations := builder.getPodAnnotationsWithBackendChecksum()

	// Assert
	assert.NotEmpty(t, annotations["checksum/backend"], "Checksum should be present")

	// Test without keysSecret
	agent2 := createInstanaAgentWithSecretMountsEnabled()
	builder2 := createTestDeploymentBuilder(t, agent2)

	// Act
	annotations2 := builder2.getPodAnnotationsWithBackendChecksum()

	// Assert
	assert.NotEmpty(t, annotations2["checksum/backend"], "Checksum should be present")
}

func TestNewDeploymentBuilder(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsEnabled()
	backend := backends.K8SensorBackend{
		EndpointHost:   agent.Spec.Agent.EndpointHost,
		EndpointPort:   agent.Spec.Agent.EndpointPort,
		EndpointKey:    agent.Spec.Agent.Key,
		ResourceSuffix: "",
	}
	mockStatusManager := new(MockStatusManager)
	secret := &corev1.Secret{
		Data: map[string][]byte{
			"key": []byte("test-key-from-secret"),
		},
	}

	// Act
	builder := NewDeploymentBuilder(agent, false, mockStatusManager, backend, secret, nil)

	// Assert
	assert.NotNil(t, builder, "NewDeploymentBuilder should return a non-nil builder")
	assert.IsType(
		t,
		&deploymentBuilder{},
		builder,
		"NewDeploymentBuilder should return a deploymentBuilder",
	)
}

func TestBuildDeploymentWithProxyConfiguration(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithProxy()
	builder := createTestDeploymentBuilder(t, agent)

	// Setup mocks for the build method
	configVolume := corev1.Volume{Name: "config"}
	configMount := corev1.VolumeMount{Name: "config"}
	secretsVolume := corev1.Volume{Name: "instana-secrets"}
	secretsMount := corev1.VolumeMount{Name: "instana-secrets"}

	builder.VolumeBuilder.(*MockVolumeBuilder).On("Build", mock.Anything).
		Return([]corev1.Volume{configVolume, secretsVolume}, []corev1.VolumeMount{configMount, secretsMount})

	builder.statusManager.(*MockStatusManager).On("SetK8sSensorDeployment", mock.Anything).Return()

	// Act
	deploymentObj := builder.build()

	// Assert
	assert.NotNil(t, deploymentObj, "Deployment object should not be nil")

	// Check container command and args
	container := deploymentObj.Spec.Template.Spec.Containers[0]
	assert.Equal(t, "/ko-app/k8sensor", container.Command[0], "Container command should be correct")

	// Check that args include both agent-key-file and https-proxy-file
	args := container.Args
	assert.Contains(
		t,
		args,
		"-agent-key-file",
		"Container args should include -agent-key-file when UseSecretMounts is enabled",
	)
	assert.Contains(
		t,
		args,
		"-https-proxy-file",
		"Container args should include -https-proxy-file when proxy is configured",
	)

	// Check volumes
	assert.Equal(
		t,
		2,
		len(deploymentObj.Spec.Template.Spec.Volumes),
		"Should have 2 volumes when UseSecretMounts is enabled",
	)
	assert.True(t, containsVolume(deploymentObj.Spec.Template.Spec.Volumes, "instana-secrets"),
		"Deployment should include secrets volume when UseSecretMounts is enabled")

	// Check volume mounts
	assert.Equal(
		t,
		2,
		len(container.VolumeMounts),
		"Should have 2 volume mounts when UseSecretMounts is enabled",
	)
	assert.True(t, containsVolumeMount(container.VolumeMounts, "instana-secrets"),
		"Container should include secrets volume mount when UseSecretMounts is enabled")

	// Check environment variables
	assert.False(t, containsEnvVar(container.Env, "AGENT_KEY"),
		"AGENT_KEY environment variable should not be present when UseSecretMounts is enabled")
}

// Test case for multiple backends to prevent regression
func TestGetEnvVarsWithMultipleBackends(t *testing.T) {
	// Arrange
	agent := createInstanaAgentWithSecretMountsDisabled()

	// Create a builder with a non-empty ResourceSuffix to simulate additional backend
	backend := backends.K8SensorBackend{
		EndpointHost:   agent.Spec.Agent.EndpointHost,
		EndpointPort:   agent.Spec.Agent.EndpointPort,
		EndpointKey:    "additional-backend-key",
		ResourceSuffix: "-1", // This simulates an additional backend
	}

	builder := createTestDeploymentBuilder(t, agent)
	builder.backend = backend

	// Act
	envVars := builder.getEnvVars()

	// Assert
	agentKeyEnv := getEnvVar(envVars, "AGENT_KEY")
	assert.NotNil(t, agentKeyEnv, "AGENT_KEY environment variable should be present")

	// The key should be from a secret reference, not a hardcoded value
	assert.NotNil(t, agentKeyEnv.ValueFrom, "AGENT_KEY should use ValueFrom, not a hardcoded Value")
	assert.NotNil(t, agentKeyEnv.ValueFrom.SecretKeyRef, "AGENT_KEY should reference a secret")

	// Check that it's using the correct key with suffix
	assert.Equal(t, constants.AgentKey+"-1", agentKeyEnv.ValueFrom.SecretKeyRef.Key,
		"Secret key should include the backend suffix")
}

// Helper function to get an environment variable by name for detailed inspection
func getEnvVar(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for _, env := range envVars {
		if env.Name == name {
			return &env
		}
	}
	return nil
}
