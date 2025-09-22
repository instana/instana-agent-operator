package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func TestVolumeBuilder_ETCDCAVolume_OpenShift(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						MountPath: "/var/run/secrets/kubernetes.io/serviceaccount",
					},
				},
			},
		},
	}
	builder := NewVolumeBuilder(agent, true) // isOpenShift = true

	// When
	volumes, mounts := builder.Build(ETCDCAVolume)

	// Then
	assert.Len(t, volumes, 1, "Should create one volume")
	assert.Len(t, mounts, 1, "Should create one volume mount")

	assert.Equal(
		t,
		"etcd-ca",
		volumes[0].Name,
		"Volume name should be etcd-ca",
	)
	assert.NotNil(
		t,
		volumes[0].VolumeSource.ConfigMap,
		"Volume source should be ConfigMap for OpenShift",
	)
	assert.Equal(
		t,
		"etcd-ca",
		volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name,
		"ConfigMap name should be etcd-ca",
	)
	assert.Len(
		t,
		volumes[0].VolumeSource.ConfigMap.Items,
		1,
		"Should have one item mapping",
	)
	assert.Equal(
		t,
		"service-ca.crt",
		volumes[0].VolumeSource.ConfigMap.Items[0].Key,
		"ConfigMap key should be service-ca.crt",
	)
	assert.Equal(
		t,
		"service-ca.crt",
		volumes[0].VolumeSource.ConfigMap.Items[0].Path,
		"ConfigMap path should be service-ca.crt",
	)

	assert.Equal(
		t,
		"etcd-ca",
		mounts[0].Name,
		"Mount name should match volume name",
	)
	assert.Equal(
		t,
		"/etc/service-ca",
		mounts[0].MountPath,
		"Mount path should be /etc/service-ca",
	)
	assert.True(
		t,
		mounts[0].ReadOnly,
		"Mount should be read-only",
	)
}

func TestVolumeBuilder_ETCDCAVolume_CustomSecret(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						MountPath:  "/etc/ssl/certs",
						SecretName: "etcd-ca-cert",
						SecretKey:  "ca.crt",
					},
				},
			},
		},
	}
	builder := NewVolumeBuilder(agent, false) // isOpenShift = false

	// When
	volumes, mounts := builder.Build(ETCDCAVolume)

	// Then
	assert.Len(t, volumes, 1, "Should create one volume")
	assert.Len(t, mounts, 1, "Should create one volume mount")

	assert.Equal(
		t,
		"etcd-ca",
		volumes[0].Name,
		"Volume name should be etcd-ca",
	)
	assert.NotNil(
		t,
		volumes[0].VolumeSource.Secret,
		"Volume source should be Secret",
	)
	assert.Equal(
		t,
		"etcd-ca-cert",
		volumes[0].VolumeSource.Secret.SecretName,
		"Secret name should match specified name",
	)
	assert.Len(
		t,
		volumes[0].VolumeSource.Secret.Items,
		1,
		"Should have one item mapping",
	)
	assert.Equal(
		t,
		"ca.crt",
		volumes[0].VolumeSource.Secret.Items[0].Key,
		"Secret key should match specified key",
	)
	assert.Equal(
		t,
		"ca.crt",
		volumes[0].VolumeSource.Secret.Items[0].Path,
		"Secret path should be ca.crt",
	)

	assert.Equal(
		t,
		"etcd-ca",
		mounts[0].Name,
		"Mount name should match volume name",
	)
	assert.Equal(
		t,
		"/etc/ssl/certs",
		mounts[0].MountPath,
		"Mount path should match specified path",
	)
	assert.True(
		t,
		mounts[0].ReadOnly,
		"Mount should be read-only",
	)
}

func TestVolumeBuilder_ETCDCAVolume_DefaultSecretKey(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						MountPath:  "/etc/ssl/certs",
						SecretName: "etcd-ca-cert",
						// No SecretKey specified, should default to "ca.crt"
					},
				},
			},
		},
	}
	builder := NewVolumeBuilder(agent, false)

	// When
	volumes, _ := builder.Build(ETCDCAVolume)

	// Then
	assert.Len(t, volumes, 1, "Should create one volume")
	assert.Equal(
		t,
		"ca.crt",
		volumes[0].VolumeSource.Secret.Items[0].Key,
		"Secret key should default to ca.crt",
	)
}

func TestVolumeBuilder_ETCDCAVolume_NoConfig(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{},
			},
		},
	}
	builder := NewVolumeBuilder(agent, false) // isOpenShift = false

	// When
	volumes, mounts := builder.Build(ETCDCAVolume)

	// Then
	assert.Len(t, volumes, 0, "Should not create any volumes")
	assert.Len(t, mounts, 0, "Should not create any mounts")
}

func TestVolumeBuilder_ControlPlaneCAVolume(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				RestClient: instanav1.RestClientSpec{
					CA: instanav1.CASpec{
						MountPath:  "/etc/ssl/control-plane",
						SecretName: "control-plane-ca",
						SecretKey:  "ca.crt",
					},
				},
			},
		},
	}
	builder := NewVolumeBuilder(agent, false)

	// When
	volumes, mounts := builder.Build(ControlPlaneCAVolume)

	// Then
	assert.Len(t, volumes, 1, "Should create one volume")
	assert.Len(t, mounts, 1, "Should create one volume mount")

	assert.Equal(
		t,
		"control-plane-ca",
		volumes[0].Name,
		"Volume name should be control-plane-ca",
	)
	assert.NotNil(
		t,
		volumes[0].VolumeSource.Secret,
		"Volume source should be Secret",
	)
	assert.Equal(
		t,
		"control-plane-ca",
		volumes[0].VolumeSource.Secret.SecretName,
		"Secret name should match specified name",
	)

	assert.Equal(
		t,
		"control-plane-ca",
		mounts[0].Name,
		"Mount name should match volume name",
	)
	assert.Equal(
		t,
		"/etc/ssl/control-plane",
		mounts[0].MountPath,
		"Mount path should match specified path",
	)
	assert.True(
		t,
		mounts[0].ReadOnly,
		"Mount should be read-only",
	)
}

func TestVolumeBuilder_ControlPlaneCAVolume_NoConfig(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				RestClient: instanav1.RestClientSpec{},
			},
		},
	}
	builder := NewVolumeBuilder(agent, false)

	// When
	volumes, mounts := builder.Build(ControlPlaneCAVolume)

	// Then
	assert.Len(t, volumes, 0, "Should not create any volumes")
	assert.Len(t, mounts, 0, "Should not create any mounts")
}
