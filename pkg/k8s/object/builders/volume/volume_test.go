package volume

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/rand"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"

	"github.com/golang/mock/gomock"

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

	expected := []VolumeWithMount{
		{
			Volume: corev1.Volume{
				Name: "config",
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: agentName,
						},
					},
				},
			},
			VolumeMount: corev1.VolumeMount{
				Name:      "config",
				MountPath: "/opt/instana/agent/etc/instana",
				// Must be false since we need to copy the tpl files into here
				ReadOnly: false,
			},
		},
	}

	v := NewVolumeBuilder(
		&instanav1.InstanaAgent{
			ObjectMeta: metav1.ObjectMeta{
				Name: agentName,
			},
		}, false,
	)

	actual := v.Build(ConfigVolume)

	assertions.Equal(expected, actual)
}

func TestTPLFilesTmpVolume(t *testing.T) {
	assertions := require.New(t)

	expected := []VolumeWithMount{
		{
			Volume: corev1.Volume{
				Name: "tpl-files-volume",
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			VolumeMount: corev1.VolumeMount{
				Name:      "tpl-files-volume",
				MountPath: "/tmp/agent_tpl_files",
				ReadOnly:  false,
			},
		},
	}

	v := NewVolumeBuilder(nil, false)

	actual := v.Build(TPLFilesTmpVolume)

	assertions.Equal(expected, actual)
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
