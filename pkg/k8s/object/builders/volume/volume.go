package volume

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
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

func hostVolumeWithMountLiteralWhenCondition(
	condition bool,
	params *hostVolumeWithMountParams,
) optional.Optional[VolumeWithMount] {
	switch condition {
	case true:
		return hostVolumeWithMountLiteral(params)
	default:
		return optional.Empty[VolumeWithMount]()
	}
}

func VarRunKuboVolume(isNotOpenShift bool) optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenCondition(
		isNotOpenShift,
		&hostVolumeWithMountParams{
			name:                 "var-run-kubo",
			path:                 "/var/vcap/sys/run/docker",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func VarRunContainerdVolume(isNotOpenShift bool) optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenCondition(
		isNotOpenShift,
		&hostVolumeWithMountParams{
			name:                 "var-run-containerd",
			path:                 "/var/vcap/sys/run/containerd",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func VarContainerdConfigVolume(isNotOpenShift bool) optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenCondition(
		isNotOpenShift,
		&hostVolumeWithMountParams{
			name:                 "var-containerd-config",
			path:                 "/var/vcap/jobs/containerd/config",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func SysVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "sys",
		path:                 "/sys",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func VarLogVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "var-log",
		path:                 "/var/log",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func VarLibVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "var-lib",
		path:                 "/var/lib",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func VarDataVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name:                 "var-data",
		path:                 "/var/data",
		MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
	})
}

func MachineIdVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(&hostVolumeWithMountParams{
		name: "machine-id",
		path: "/etc/machine-id",
	})
}

// TODO: Resolve config volumes

func TlsVolume(helpers helpers.Helpers) optional.Optional[VolumeWithMount] {
	const volumeName = "instana-agent-tls"

	switch helpers.TLSIsEnabled() {
	case true:
		return optional.Of(VolumeWithMount{
			Volume: corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					Secret: &corev1.SecretVolumeSource{
						SecretName:  helpers.TLSSecretName(),
						DefaultMode: pointer.To[int32](0440),
					},
				},
			},
			VolumeMount: corev1.VolumeMount{
				Name:      volumeName,
				MountPath: "/opt/instana/agent/etc/certs",
				ReadOnly:  true,
			},
		})
	default:
		return optional.Empty[VolumeWithMount]()
	}
}

func RepoVolume(agent *instanav1.InstanaAgent) optional.Optional[VolumeWithMount] {
	const volumeName = "repo"

	return optional.Map[string, VolumeWithMount](
		optional.Of(agent.Spec.Agent.Host.Repository),
		func(path string) VolumeWithMount {
			return VolumeWithMount{
				Volume: corev1.Volume{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						HostPath: &corev1.HostPathVolumeSource{
							Path: path,
						},
					},
				},
				VolumeMount: corev1.VolumeMount{
					Name:      volumeName,
					MountPath: "/opt/instana/agent/data/repo",
				},
			}
		},
	)
}

// TODO: Additional Backends -> potentially part of conifguration volume
