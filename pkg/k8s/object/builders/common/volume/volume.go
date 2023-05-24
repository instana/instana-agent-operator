package volume

import (
	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	InstanaConfigDirectory            = "/opt/instana/agent/etc/instana"
	InstanaConfigTPLFilesTmpDirectory = "/tmp/agent_tpl_files"
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

func (v *volumeBuilder) devVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "dev",
			path:                 "/dev",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) runVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "run",
			path:                 "/run",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) varRunVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "var-run",
			path:                 "/var/run",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
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

func (v *volumeBuilder) varRunKuboVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenCondition(
		v.isNotOpenShift,
		&hostVolumeWithMountParams{
			name:                 "var-run-kubo",
			path:                 "/var/vcap/sys/run/docker",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) varRunContainerdVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenCondition(
		v.isNotOpenShift,
		&hostVolumeWithMountParams{
			name:                 "var-run-containerd",
			path:                 "/var/vcap/sys/run/containerd",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) varContainerdConfigVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteralWhenCondition(
		v.isNotOpenShift,
		&hostVolumeWithMountParams{
			name:                 "var-containerd-config",
			path:                 "/var/vcap/jobs/containerd/config",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) sysVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "sys",
			path:                 "/sys",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) varLogVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "var-log",
			path:                 "/var/log",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) varLibVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "var-lib",
			path:                 "/var/lib",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) varDataVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name:                 "var-data",
			path:                 "/var/data",
			MountPropagationMode: pointer.To(corev1.MountPropagationHostToContainer),
		},
	)
}

func (v *volumeBuilder) machineIdVolume() optional.Optional[VolumeWithMount] {
	return hostVolumeWithMountLiteral(
		&hostVolumeWithMountParams{
			name: "machine-id",
			path: "/etc/machine-id",
		},
	)
}

func (v *volumeBuilder) configVolume() optional.Optional[VolumeWithMount] {
	const volumeName = "config"

	return optional.Of[VolumeWithMount](
		VolumeWithMount{
			Volume: corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: v.Name,
						},
					},
				},
			},
			VolumeMount: corev1.VolumeMount{
				Name:      volumeName,
				MountPath: InstanaConfigDirectory,
				// Must be false since we need to copy the tpl files into here
				ReadOnly: false,
			},
		},
	)
}

func (v *volumeBuilder) tplFilesTmpVolume() optional.Optional[VolumeWithMount] {
	const volumeName = "tpl-files-volume"

	return optional.Of(
		VolumeWithMount{
			Volume: corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					EmptyDir: &corev1.EmptyDirVolumeSource{},
				},
			},
			VolumeMount: corev1.VolumeMount{
				Name:      volumeName,
				MountPath: InstanaConfigTPLFilesTmpDirectory,
				// Must be false since we need to copy the tpl files into here
				ReadOnly: false,
			},
		},
	)
}

func (v *volumeBuilder) tlsVolume() optional.Optional[VolumeWithMount] {
	const volumeName = "instana-agent-tls"

	switch v.TLSIsEnabled() {
	case true:
		return optional.Of(
			VolumeWithMount{
				Volume: corev1.Volume{
					Name: volumeName,
					VolumeSource: corev1.VolumeSource{
						Secret: &corev1.SecretVolumeSource{
							SecretName:  v.TLSSecretName(),
							DefaultMode: pointer.To[int32](0440),
						},
					},
				},
				VolumeMount: corev1.VolumeMount{
					Name:      volumeName,
					MountPath: "/opt/instana/agent/etc/certs",
					ReadOnly:  true,
				},
			},
		)
	default:
		return optional.Empty[VolumeWithMount]()
	}
}

func (v *volumeBuilder) repoVolume() optional.Optional[VolumeWithMount] {
	const volumeName = "repo"

	return optional.Map[string, VolumeWithMount](
		optional.Of(v.Spec.Agent.Host.Repository),
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
