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
	corev1.Volume
	corev1.VolumeMount
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

//type whenOpenshiftn
