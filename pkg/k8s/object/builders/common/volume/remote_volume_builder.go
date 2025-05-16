/*
(c) Copyright IBM Corp. 2024
*/

package volume

import (
	"errors"

	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const RemoteConfigDirectory = "/opt/instana/agent/etc/remote-instana-config-yml"

type RemoteVolume int

const (
	DevVolumeRemote RemoteVolume = iota
	RunVolumeRemote
	VarRunVolumeRemote
	VarRunKuboVolumeRemote
	VarRunContainerdVolumeRemote
	VarContainerdConfigVolumeRemote
	SysVolumeRemote
	VarLogVolumeRemote
	VarLibVolumeRemote
	VarDataVolumeRemote
	MachineIdVolumeRemote
	ConfigVolumeRemote
	TlsVolumeRemote
	RepoVolumeRemote
)

type VolumeBuilderRemote interface {
	Build(volumes ...Volume) ([]corev1.Volume, []corev1.VolumeMount)
	BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount)
}

type volumeBuilderRemote struct {
	remoteAgent    *instanav1.RemoteAgent
	helpers        helpers.RemoteHelpers
	isNotOpenShift bool
}

func NewVolumeBuilderRemote(agent *instanav1.RemoteAgent) VolumeBuilderRemote {
	return &volumeBuilderRemote{
		remoteAgent: agent,
		helpers:     helpers.NewRemoteHelpers(agent),
	}
}

func (v *volumeBuilderRemote) Build(volumes ...Volume) ([]corev1.Volume, []corev1.VolumeMount) {
	volumeSpecs := []corev1.Volume{}
	volumeMounts := []corev1.VolumeMount{}
	for _, volumeNumber := range volumes {
		volume, volumeMount := v.getBuilder(volumeNumber)
		if volume != nil {
			volumeSpecs = append(volumeSpecs, *volume)
		}
		if volumeMount != nil {
			volumeMounts = append(volumeMounts, *volumeMount)
		}
	}
	return volumeSpecs, volumeMounts
}

func (v *volumeBuilderRemote) BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount) {
	return v.remoteAgent.Spec.Agent.Pod.Volumes, v.remoteAgent.Spec.Agent.Pod.VolumeMounts
}

func (v *volumeBuilderRemote) getBuilder(volume Volume) (*corev1.Volume, *corev1.VolumeMount) {
	mountPropagationHostToContainer := corev1.MountPropagationHostToContainer
	mkdir := corev1.HostPathDirectoryOrCreate

	switch volume {
	case DevVolume:
		return v.hostVolumeWithMount("dev", "/dev", &mountPropagationHostToContainer, nil)
	case RunVolume:
		return v.hostVolumeWithMount("run", "/run", &mountPropagationHostToContainer, nil)
	case VarRunVolume:
		return v.hostVolumeWithMount("var-run", "/var/run", &mountPropagationHostToContainer, nil)
	case VarRunKuboVolume:
		return v.hostVolumeWithMountLiteralWhenCondition(
			v.isNotOpenShift,
			"var-run-kubo",
			"/var/vcap/sys/run/docker",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case VarRunContainerdVolume:
		return v.hostVolumeWithMountLiteralWhenCondition(
			v.isNotOpenShift,
			"var-run-containerd",
			"/var/vcap/sys/run/containerd",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case VarContainerdConfigVolume:
		return v.hostVolumeWithMountLiteralWhenCondition(
			v.isNotOpenShift,
			"var-containerd-config",
			"/var/vcap/jobs/containerd/config",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case SysVolume:
		return v.hostVolumeWithMount("sys", "/sys", &mountPropagationHostToContainer, nil)
	case VarLogVolume:
		return v.hostVolumeWithMount("var-log", "/var/log", &mountPropagationHostToContainer, nil)
	case VarLibVolume:
		return v.hostVolumeWithMount("var-lib", "/var/lib", &mountPropagationHostToContainer, nil)
	case VarDataVolume:
		return v.hostVolumeWithMount(
			"var-data",
			"/var/data",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case MachineIdVolume:
		return v.hostVolumeWithMount("machine-id", "/etc/machine-id", nil, nil)
	case ConfigVolume:
		return v.configVolume()
	case TlsVolume:
		return v.tlsVolume()
	case RepoVolume:
		return v.repoVolume()
	default:
		panic(errors.New("unknown volume requested"))
	}
}

func (v *volumeBuilderRemote) hostVolumeWithMount(
	name string,
	path string,
	mountPropagationMode *corev1.MountPropagationMode,
	hostPathType *corev1.HostPathType,
) (*corev1.Volume, *corev1.VolumeMount) {
	volume := corev1.Volume{
		Name: name,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: path,
				Type: hostPathType,
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:             name,
		MountPath:        path,
		MountPropagation: mountPropagationMode,
	}

	return &volume, &volumeMount
}

func (v *volumeBuilderRemote) hostVolumeWithMountLiteralWhenCondition(
	condition bool,
	name string,
	path string,
	mountPropagationMode *corev1.MountPropagationMode,
	hostPathType *corev1.HostPathType,
) (*corev1.Volume, *corev1.VolumeMount) {
	if condition {
		return v.hostVolumeWithMount(name, path, mountPropagationMode, hostPathType)
	}

	return nil, nil
}

func (v *volumeBuilderRemote) configVolume() (*corev1.Volume, *corev1.VolumeMount) {
	volumeName := "config"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  v.remoteAgent.Name + "-config",
				DefaultMode: pointer.To[int32](0440),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: InstanaConfigDirectory,
	}
	return &volume, &volumeMount
}

func (v *volumeBuilderRemote) tlsVolume() (*corev1.Volume, *corev1.VolumeMount) {
	if !v.helpers.TLSIsEnabled() {
		return nil, nil
	}

	volumeName := "instana-agent-tls"
	defaultMode := int32(0440)

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  v.helpers.TLSSecretName(),
				DefaultMode: &defaultMode,
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/opt/instana/agent/etc/certs",
		ReadOnly:  true,
	}
	return &volume, &volumeMount

}

func (v *volumeBuilderRemote) repoVolume() (*corev1.Volume, *corev1.VolumeMount) {
	if v.remoteAgent.Spec.Agent.Host.Repository == "" {
		return nil, nil
	}
	volumeName := "repo"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: v.remoteAgent.Spec.Agent.Host.Repository,
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/opt/instana/agent/data/repo",
	}

	return &volume, &volumeMount
}
