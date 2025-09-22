package deployment

import (
	"testing"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	backend "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func TestDeploymentBuilder_GetEnvVars_IncludesETCDEnvVars(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-ns",
		},
		Spec: instanav1.InstanaAgentSpec{
			Agent: instanav1.BaseAgentSpec{
				Key: "test-key",
			},
			Zone: instanav1.Name{
				Name: "test-zone",
			},
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					Insecure: pointer.To(false),
					CA: instanav1.CASpec{
						MountPath: "/etc/ssl/certs",
					},
					Targets: []string{"https://etcd-1:2379"},
				},
				RestClient: instanav1.RestClientSpec{
					HostAllowlist: []string{"localhost"},
					CA: instanav1.CASpec{
						MountPath: "/etc/ssl/control-plane",
					},
				},
			},
		},
	}

	mockStatusManager := &status.MockAgentStatusManager{}
	backendObj := backend.NewK8SensorBackend(
		"",
		"test-key",
		"",
		"test-host",
		"443",
	)

	builder := NewDeploymentBuilder(agent, true, mockStatusManager, *backendObj, nil, nil).(*deploymentBuilder)

	// When
	envVars := builder.getEnvVars()

	// Then
	etcdCAFileEnv := findEnvVar(envVars, "ETCD_CA_FILE")
	assert.NotNil(t, etcdCAFileEnv, "ETCD_CA_FILE env var should be present")
	assert.Equal(t, "/etc/ssl/certs/ca.crt", etcdCAFileEnv.Value)

	etcdInsecureEnv := findEnvVar(envVars, "ETCD_INSECURE")
	assert.NotNil(t, etcdInsecureEnv, "ETCD_INSECURE env var should be present")
	assert.Equal(t, "false", etcdInsecureEnv.Value)

	etcdTargetsEnv := findEnvVar(envVars, "ETCD_TARGETS")
	assert.NotNil(t, etcdTargetsEnv, "ETCD_TARGETS env var should be present")
	assert.Equal(t, "https://etcd-1:2379", etcdTargetsEnv.Value)

	hostAllowlistEnv := findEnvVar(envVars, "REST_CLIENT_HOST_ALLOWLIST")
	assert.NotNil(t, hostAllowlistEnv, "REST_CLIENT_HOST_ALLOWLIST env var should be present")
	assert.Equal(t, "localhost", hostAllowlistEnv.Value)

	controlPlaneCAFileEnv := findEnvVar(envVars, "CONTROL_PLANE_CA_FILE")
	assert.NotNil(t, controlPlaneCAFileEnv, "CONTROL_PLANE_CA_FILE env var should be present")
	assert.Equal(t, "/etc/ssl/control-plane/ca.crt", controlPlaneCAFileEnv.Value)
}

func TestDeploymentBuilder_GetVolumes_IncludesCAVolumes(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-agent",
			Namespace: "test-ns",
		},
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						MountPath:  "/etc/ssl/certs",
						SecretName: "etcd-ca-cert",
					},
				},
				RestClient: instanav1.RestClientSpec{
					CA: instanav1.CASpec{
						MountPath:  "/etc/ssl/control-plane",
						SecretName: "control-plane-ca",
					},
				},
			},
		},
	}

	mockStatusManager := &status.MockAgentStatusManager{}
	backendObj := backend.NewK8SensorBackend(
		"",
		"test-key",
		"",
		"test-host",
		"443",
	)

	builder := NewDeploymentBuilder(agent, false, mockStatusManager, *backendObj, nil, nil).(*deploymentBuilder)

	// When
	volumes, mounts := builder.getVolumes()

	// Then
	assert.GreaterOrEqual(t, len(volumes), 2, "Should have at least 2 volumes")
	assert.GreaterOrEqual(t, len(mounts), 2, "Should have at least 2 mounts")

	etcdVolume := findVolume(volumes, "etcd-ca")
	assert.NotNil(t, etcdVolume, "etcd-ca volume should be present")
	assert.NotNil(t, etcdVolume.Secret, "etcd-ca volume should be a Secret volume")
	assert.Equal(t, "etcd-ca-cert", etcdVolume.Secret.SecretName)

	controlPlaneVolume := findVolume(volumes, "control-plane-ca")
	assert.NotNil(t, controlPlaneVolume, "control-plane-ca volume should be present")
	assert.NotNil(t, controlPlaneVolume.Secret, "control-plane-ca volume should be a Secret volume")
	assert.Equal(t, "control-plane-ca", controlPlaneVolume.Secret.SecretName)

	etcdMount := findVolumeMount(mounts, "etcd-ca")
	assert.NotNil(t, etcdMount, "etcd-ca mount should be present")
	assert.Equal(t, "/etc/ssl/certs", etcdMount.MountPath)
	assert.True(t, etcdMount.ReadOnly)

	controlPlaneMount := findVolumeMount(mounts, "control-plane-ca")
	assert.NotNil(t, controlPlaneMount, "control-plane-ca mount should be present")
	assert.Equal(t, "/etc/ssl/control-plane", controlPlaneMount.MountPath)
	assert.True(t, controlPlaneMount.ReadOnly)
}

// Helper functions
func findEnvVar(envVars []corev1.EnvVar, name string) *corev1.EnvVar {
	for _, env := range envVars {
		if env.Name == name {
			return &env
		}
	}
	return nil
}

func findVolume(volumes []corev1.Volume, name string) *corev1.Volume {
	for _, vol := range volumes {
		if vol.Name == name {
			return &vol
		}
	}
	return nil
}

func findVolumeMount(mounts []corev1.VolumeMount, name string) *corev1.VolumeMount {
	for _, mount := range mounts {
		if mount.Name == name {
			return &mount
		}
	}
	return nil
}
