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
