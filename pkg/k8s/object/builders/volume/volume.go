package volume

import (
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
	corev1 "k8s.io/api/core/v1"
)

type hostVolumeWithMountParams struct {
	name string
	path string
	*corev1.MountPropagationMode
}

type VolumeWithMount struct {
	Volume      corev1.Volume
	VolumeMount corev1.VolumeMount
}

func hostVolumeWithMount(params *hostVolumeWithMountParams) VolumeWithMount {
	return VolumeWithMount{
		Volume: corev1.Volume{
			Name: params.name,
			VolumeSource: corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: params.path,
				},
			},
		},
		VolumeMount: corev1.VolumeMount{
			Name:             params.name,
			MountPath:        params.path,
			MountPropagation: params.MountPropagationMode,
		},
	}
}

func hostVolumeWithMountLiteral(params *hostVolumeWithMountParams) optional.Optional[VolumeWithMount] {
	return optional.Of(hostVolumeWithMount(params))
}

func DevVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "dev",
		path:                 "/dev",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func RunVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "run",
		path:                 "/run",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func VarRunVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "var-run",
		path:                 "/var/run",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func hostVolumeWithMountLiteralWhenOpenShift(
	isOpenShift bool,
	params *hostVolumeWithMountParams,
) optional.Optional[VolumeWithMount] {
	switch isOpenShift {
	case true:
		return hostVolumeWithMountLiteral(params)
	default:
		return optional.Empty[VolumeWithMount]()
	}
} // TODO: rest of these

func VarRunKuboVolume(isOpenShift bool) optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenOpenShift(isOpenShift, &hostVolumeWithMountParams{
		name:                 "var-run-kubo",
		path:                 "/var/vcap/sys/run/docker",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}
