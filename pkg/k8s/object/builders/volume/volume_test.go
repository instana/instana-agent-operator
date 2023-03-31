package volume

import (
	"testing"

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

func testHostLiteralOnlyWhenNotOpenShift(
	t *testing.T,
	expected *hostVolumeWithMountParams,
	f func(isOpenShift bool) optional.Optional[VolumeWithMount],
) {
	t.Run("is_OpenShift", func(t *testing.T) {
		assertions := require.New(t)

		assertions.Empty(f(true))
	})
	t.Run("not_OpenShift", func(t *testing.T) {
		testHostLiteralVolume(t, expected, func() optional.Optional[VolumeWithMount] {
			return f(false)
		})
	})
}

func Test_hostVolumeWithMountLiteralWhenNotOpenShift(t *testing.T) {
	expected := &hostVolumeWithMountParams{
		name:                 "erasgasd",
		path:                 "arehasdfasdf",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	}
	testHostLiteralOnlyWhenNotOpenShift(t, expected, func(isOpenShift bool) optional.Optional[VolumeWithMount] {
		return hostVolumeWithMountLiteralWhenNotOpenShift(isOpenShift, expected)
	})
}

func TestVarRunKuboVolume(t *testing.T) {
	testHostLiteralOnlyWhenNotOpenShift(
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
	testHostLiteralOnlyWhenNotOpenShift(
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
	testHostLiteralOnlyWhenNotOpenShift(
		t,
		&hostVolumeWithMountParams{
			name:                 "var-containerd-config",
			path:                 "/var/vcap/jobs/containerd/config",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
		VarContainerdConfigVolume,
	)
}
