package volume

import (
	"testing"

	"github.com/golang/mock/gomock"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
)

func testHostLiteralVolume(
	t *testing.T,
	expected *hostVolumeWithMountParams,
	f func() optional.Optional[VolumeWithMount],
) {
	assertions := require.New(t)

	VolumeWithMountOpt := f()

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

func Test_fromHostLiteral(t *testing.T) {
	expected := &hostVolumeWithMountParams{
		name:                 "erasgasd",
		path:                 "arehasdfasdf",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	}
	testHostLiteralVolume(t, expected, func() optional.Optional[VolumeWithMount] {
		return hostVolumeWithMountLiteral(expected)
	})
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
	f func(condition bool) optional.Optional[VolumeWithMount],
) {
	t.Run("not_condition", func(t *testing.T) {
		assertions := require.New(t)

		assertions.Empty(f(false))
	})
	t.Run("is_condition", func(t *testing.T) {
		testHostLiteralVolume(t, expected, func() optional.Optional[VolumeWithMount] {
			return f(true)
		})
	})
}

func Test_hostVolumeWithMountLiteralWhenNotCondition(t *testing.T) {
	expected := &hostVolumeWithMountParams{
		name:                 "erasgasd",
		path:                 "arehasdfasdf",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	}
	testHostLiteralOnlyWhenCondition(t, expected, func(condition bool) optional.Optional[VolumeWithMount] {
		return hostVolumeWithMountLiteralWhenCondition(condition, expected)
	})
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

func TestTlsVolume(t *testing.T) {
	t.Run("tls_not_enabled", func(t *testing.T) {
		assertions := require.New(t)
		ctrl := gomock.NewController(t)

		helpers := NewMockHelpers(ctrl)
		helpers.EXPECT().TLSIsEnabled().Return(false)

		assertions.Empty(TlsVolume(helpers))
	})
	t.Run("tls_is_enabled", func(t *testing.T) {
		assertions := require.New(t)
		ctrl := gomock.NewController(t)

		helpers := NewMockHelpers(ctrl)
		helpers.EXPECT().TLSIsEnabled().Return(true)
		helpers.EXPECT().TLSSecretName().Return("goisoijsoigjsd")

		assertions.Equal(
			optional.Of(VolumeWithMount{
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
			}),
			TlsVolume(helpers),
		)
	})
}
