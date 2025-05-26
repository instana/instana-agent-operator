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

const RemoteConfigDirectory = "/opt/instana/agent/etc/remote-config-yml"

type RemoteVolume int

const (
	ConfigVolumeRemote RemoteVolume = iota
	TlsVolumeRemote
	RepoVolumeRemote
)

type VolumeBuilderRemote interface {
	Build(volumes ...Volume) ([]corev1.Volume, []corev1.VolumeMount)
	BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount)
}

type volumeBuilderRemote struct {
	remoteAgent *instanav1.RemoteAgent
	helpers     helpers.RemoteHelpers
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
	switch volume {
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
		MountPath: RemoteConfigDirectory,
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
	// If the repository field is not provided (or empty), do not mount any volume.
	if v.remoteAgent.Spec.Agent.Host.Repository == "" {
		return nil, nil
	}
	volumeName := "repo"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			PersistentVolumeClaim: &corev1.PersistentVolumeClaimVolumeSource{
				ClaimName: v.remoteAgent.Spec.Agent.Host.Repository,
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/opt/instana/agent/data/repo",
	}
	return &volume, &volumeMount
}
