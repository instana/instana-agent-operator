/*
(c) Copyright IBM Corp. 2025

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package volume

import (
	"errors"

	corev1 "k8s.io/api/core/v1"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	RemoteConfigDirectory   = "/opt/instana/agent/etc/remote-config-yml"
	InstanaSecretsDirectory = "/opt/instana/agent/etc/instana/secrets"
)

type RemoteVolume int

const (
	ConfigVolumeRemote RemoteVolume = iota
	TlsVolumeRemote
	RepoVolumeRemote
	SecretsVolumeRemote
)

type VolumeBuilderRemote interface {
	Build(volumes ...RemoteVolume) ([]corev1.Volume, []corev1.VolumeMount)
	BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount)
}

type volumeBuilderRemote struct {
	remoteAgent *instanav1.InstanaAgentRemote
	helpers     helpers.RemoteHelpers
}

func NewVolumeBuilderRemote(agent *instanav1.InstanaAgentRemote) VolumeBuilderRemote {
	return &volumeBuilderRemote{
		remoteAgent: agent,
		helpers:     helpers.NewRemoteHelpers(agent),
	}
}

func (v *volumeBuilderRemote) Build(volumes ...RemoteVolume) ([]corev1.Volume, []corev1.VolumeMount) {
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

func (v *volumeBuilderRemote) getBuilder(volume RemoteVolume) (*corev1.Volume, *corev1.VolumeMount) {
	switch volume {
	case ConfigVolumeRemote:
		return v.configVolume()
	case TlsVolumeRemote:
		return v.tlsVolume()
	case RepoVolumeRemote:
		return v.repoVolume()
	case SecretsVolumeRemote:
		return v.secretsVolume()
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
				SecretName:  "instana-agent-r-" + v.remoteAgent.Name + "-config",
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

	volumeName := "remote-agent-tls"
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

func (v *volumeBuilderRemote) secretsVolume() (*corev1.Volume, *corev1.VolumeMount) {
	// Only create the secrets volume if useSecretMounts is enabled or nil (default to true)
	if v.remoteAgent.Spec.UseSecretMounts != nil && !*v.remoteAgent.Spec.UseSecretMounts {
		return nil, nil
	}

	volumeName := "instana-secrets"
	secretName := v.remoteAgent.Spec.Agent.KeysSecret
	if secretName == "" {
		secretName = v.remoteAgent.Name
	}

	// Create a volume with specific items for remote agent
	items := []corev1.KeyToPath{
		{
			Key:  constants.AgentKey,
			Path: constants.SecretFileAgentKey,
		},
	}

	// Add download key mapping only if it's specified
	if v.remoteAgent.Spec.Agent.DownloadKey != "" {
		items = append(items, corev1.KeyToPath{
			Key:  constants.DownloadKey,
			Path: constants.SecretFileDownloadKey,
		})
	}

	// Add proxy-related secrets if proxy is configured
	if v.remoteAgent.Spec.Agent.ProxyHost != "" {
		if v.remoteAgent.Spec.Agent.ProxyUser != "" {
			items = append(items, corev1.KeyToPath{
				Key:  "proxyUser",
				Path: constants.SecretFileProxyUser,
			})
		}
		if v.remoteAgent.Spec.Agent.ProxyPassword != "" {
			items = append(items, corev1.KeyToPath{
				Key:  "proxyPassword",
				Path: constants.SecretFileProxyPassword,
			})
		}
		// Add HTTPS_PROXY if needed
		items = append(items, corev1.KeyToPath{
			Key:  "httpsProxy",
			Path: constants.SecretFileHttpsProxy,
		})
	}

	// Add repository mirror credentials if configured
	if v.remoteAgent.Spec.Agent.MirrorReleaseRepoUsername != "" {
		items = append(items, corev1.KeyToPath{
			Key:  "mirrorReleaseRepoUsername",
			Path: constants.SecretFileMirrorReleaseRepoUsername,
		})
	}
	if v.remoteAgent.Spec.Agent.MirrorReleaseRepoPassword != "" {
		items = append(items, corev1.KeyToPath{
			Key:  "mirrorReleaseRepoPassword",
			Path: constants.SecretFileMirrorReleaseRepoPassword,
		})
	}
	if v.remoteAgent.Spec.Agent.MirrorSharedRepoUsername != "" {
		items = append(items, corev1.KeyToPath{
			Key:  "mirrorSharedRepoUsername",
			Path: constants.SecretFileMirrorSharedRepoUsername,
		})
	}
	if v.remoteAgent.Spec.Agent.MirrorSharedRepoPassword != "" {
		items = append(items, corev1.KeyToPath{
			Key:  "mirrorSharedRepoPassword",
			Path: constants.SecretFileMirrorSharedRepoPassword,
		})
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: pointer.To[int32](0400), // Read-only for owner
				Items:       items,
				Optional:    pointer.To(false),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: constants.InstanaSecretsDirectory,
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
