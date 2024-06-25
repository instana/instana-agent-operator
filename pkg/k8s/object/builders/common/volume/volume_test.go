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

package volume

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"

	gomock "go.uber.org/mock/gomock"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

func testHostLiteralVolume(
	t *testing.T,
	expected *hostVolumeWithMountParams,
	volume Volume,
) {
	assertions := require.New(t)

	v := &volumeBuilder{
		isNotOpenShift: true,
	}

	VolumeWithMountOpt := v.getBuilder(volume)()

	assertions.Equal(
		corev1.Volume{
			Name: expected.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: expected.path,
					Type: expected.hostPathType,
				},
			},
		},
		VolumeWithMountOpt.Get().Volume,
	)
	assertions.Equal(
		corev1.VolumeMount{
			Name:             expected.name,
			MountPath:        expected.path,
			MountPropagation: expected.MountPropagationMode,
		},
		VolumeWithMountOpt.Get().VolumeMount,
	)
}

func TestDevVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "dev",
			path:                 "/dev",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		DevVolume,
	)
}

func TestRunVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "run",
			path:                 "/run",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		RunVolume,
	)
}

func TestVarRunVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-run",
			path:                 "/var/run",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		VarRunVolume,
	)
}

func testHostLiteralOnlyWhenCondition(
	t *testing.T,
	expected *hostVolumeWithMountParams,
	volume Volume,
) {
	t.Run(
		"not_condition", func(t *testing.T) {
			assertions := require.New(t)

			v := &volumeBuilder{}

			assertions.Empty(v.getBuilder(volume)())
		},
	)
	t.Run(
		"is_condition", func(t *testing.T) {
			testHostLiteralVolume(
				t, expected, volume,
			)
		},
	)
}

func TestVarRunKuboVolume(t *testing.T) {
	testHostLiteralOnlyWhenCondition(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-run-kubo",
			path:                 "/var/vcap/sys/run/docker",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
			hostPathType:         pointer.To(corev1.HostPathDirectoryOrCreate),
		},
		VarRunKuboVolume,
	)
}

func TestVarRunContainerdVolume(t *testing.T) {
	testHostLiteralOnlyWhenCondition(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-run-containerd",
			path:                 "/var/vcap/sys/run/containerd",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
			hostPathType:         pointer.To(corev1.HostPathDirectoryOrCreate),
		},
		VarRunContainerdVolume,
	)
}

func TestVarContainerdConfigVolume(t *testing.T) {
	testHostLiteralOnlyWhenCondition(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-containerd-config",
			path:                 "/var/vcap/jobs/containerd/config",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
			hostPathType:         pointer.To(corev1.HostPathDirectoryOrCreate),
		},
		VarContainerdConfigVolume,
	)
}

func TestSysVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "sys",
			path:                 "/sys",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		SysVolume,
	)
}

func TestVarLogVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-log",
			path:                 "/var/log",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		VarLogVolume,
	)
}

func TestVarLibVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-lib",
			path:                 "/var/lib",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		VarLibVolume,
	)
}

func TestVarDataVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-data",
			path:                 "/var/data",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
			hostPathType:         pointer.To(corev1.HostPathDirectoryOrCreate),
		},
		VarDataVolume,
	)
}

func TestMachineIdVolume(t *testing.T) {
	testHostLiteralVolume(
		t,
		&hostVolumeWithMountParams{
			name:                 "machine-id",
			path:                 "/etc/machine-id",
			MountPropagationMode: nil,
		},
		MachineIdVolume,
	)
}

func TestConfigVolume(t *testing.T) {
	assertions := require.New(t)

	agentName := rand.String(10)

	expectedVolume := []corev1.Volume{
		{
			Name: "config",
			VolumeSource: corev1.VolumeSource{
				ConfigMap: &corev1.ConfigMapVolumeSource{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: agentName,
					},
				},
			},
		},
	}
	expectedVolumeMount := []corev1.VolumeMount{
		{
			Name:      "config",
			MountPath: "/opt/instana/agent/etc/instana-config-yml",
		},
	}

	v := NewVolumeBuilder(
		&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: agentName,
			},
		}, false,
	)

	actualVolume, actualVolumeMount := v.Build(ConfigVolume)

	assertions.Equal(expectedVolume, actualVolume)
	assertions.Equal(expectedVolumeMount, actualVolumeMount)
}

func TestTlsVolume(t *testing.T) {
	t.Run(
		"tls_not_enabled", func(t *testing.T) {
			assertions := require.New(t)
			ctrl := gomock.NewController(t)

			helpers := NewMockHelpers(ctrl)
			helpers.EXPECT().TLSIsEnabled().Return(false)

			v := &volumeBuilder{
				Helpers: helpers,
			}

			assertions.Empty(v.tlsVolume())
		},
	)
	t.Run(
		"tls_is_enabled", func(t *testing.T) {
			assertions := require.New(t)
			ctrl := gomock.NewController(t)

			helpers := NewMockHelpers(ctrl)
			helpers.EXPECT().TLSIsEnabled().Return(true)
			helpers.EXPECT().TLSSecretName().Return("goisoijsoigjsd")

			v := &volumeBuilder{
				Helpers: helpers,
			}

			assertions.Equal(
				optional.Of(
					VolumeWithMount{
						Volume: corev1.Volume{
							Name: "instana-agent-tls",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName:  "goisoijsoigjsd",
									DefaultMode: pointer.To[int32](0440),
								},
							},
						},
						VolumeMount: corev1.VolumeMount{
							Name:      "instana-agent-tls",
							MountPath: "/opt/instana/agent/etc/certs",
							ReadOnly:  true,
						},
					},
				),
				v.tlsVolume(),
			)
		},
	)
}

func TestRepoVolume(t *testing.T) {
	t.Run(
		"host_repo_not_set", func(t *testing.T) {
			assertions := require.New(t)

			v := &volumeBuilder{
				InstanaAgent: &instanav1.InstanaAgent{},
			}

			assertions.Empty(v.repoVolume())
		},
	)
	t.Run(
		"host_repo_is_set", func(t *testing.T) {
			assertions := require.New(t)

			v := &volumeBuilder{
				InstanaAgent: &instanav1.InstanaAgent{
					Spec: instanav1.InstanaAgentSpec{
						Agent: instanav1.BaseAgentSpec{
							Host: instanav1.HostSpec{
								Repository: "eiosoijdsgih",
							},
						},
					},
				},
			}

			actual := v.repoVolume()
			assertions.Equal(
				optional.Of(
					VolumeWithMount{
						Volume: corev1.Volume{
							Name: "repo",
							VolumeSource: corev1.VolumeSource{
								HostPath: &corev1.HostPathVolumeSource{
									Path: "eiosoijdsgih",
								},
							},
						},
						VolumeMount: corev1.VolumeMount{
							Name:      "repo",
							MountPath: "/opt/instana/agent/data/repo",
						},
					},
				),
				actual,
			)
		},
	)
}
