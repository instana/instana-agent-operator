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

package volume

import (
	"testing"

	"github.com/stretchr/testify/assert"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
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
		constants.ETCDMetricsCABundleName,
		volumes[0].VolumeSource.ConfigMap.LocalObjectReference.Name,
		"ConfigMap name should be etcd-metrics-ca-bundle",
	)
	assert.Len(
		t,
		volumes[0].VolumeSource.ConfigMap.Items,
		1,
		"Should have one item mapping",
	)
	assert.Equal(
		t,
		"ca-bundle.crt",
		volumes[0].VolumeSource.ConfigMap.Items[0].Key,
		"ConfigMap key should be ca-bundle.crt",
	)
	assert.Equal(
		t,
		"ca-bundle.crt",
		volumes[0].VolumeSource.ConfigMap.Items[0].Path,
		"ConfigMap path should be ca-bundle.crt",
	)

	assert.Equal(
		t,
		"etcd-ca",
		mounts[0].Name,
		"Mount name should match volume name",
	)
	assert.Equal(
		t,
		constants.ETCDMetricsCAMountPath,
		mounts[0].MountPath,
		"Mount path should be /etc/etcd-metrics-ca",
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
						Filename:   "ca.crt",
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
		"Secret key should match specified filename",
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

func TestVolumeBuilder_ETCDCAVolume_DefaultFilename(t *testing.T) {
	// Given
	agent := &instanav1.InstanaAgent{
		Spec: instanav1.InstanaAgentSpec{
			K8sSensor: instanav1.K8sSpec{
				ETCD: instanav1.ETCDSpec{
					CA: instanav1.CASpec{
						MountPath:  "/etc/ssl/certs",
						SecretName: "etcd-ca-cert",
						// No Filename specified, should default to "ca.crt"
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
		"Filename should default to ca.crt",
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
						Filename:   "ca.crt",
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
