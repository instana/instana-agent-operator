/*
(c) Copyright IBM Corp. 2024, 2025
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

type Volume int

const (
	DevVolume Volume = iota
	RunVolume
	VarRunVolume
	VarRunKuboVolume
	VarRunContainerdVolume
	VarContainerdConfigVolume
	SysVolume
	VarLogVolume
	//VarLibVolume(Removed for CSP requirement)
	VarDataVolume
	MachineIdVolume
	ConfigVolume
	TlsVolume
	RepoVolume
	NamespacesDetailsVolume
	SecretsVolume
	K8SensorSecretsVolume
	ETCDCAVolume
	ETCDClientCertVolume
	ControlPlaneCAVolume
)

type VolumeBuilder interface {
	Build(volumes ...Volume) ([]corev1.Volume, []corev1.VolumeMount)
	BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount)
	WithBackendResourceSuffix(string) VolumeBuilder
}

type volumeBuilder struct {
	instanaAgent          *instanav1.InstanaAgent
	helpers               helpers.Helpers
	isOpenShift           bool
	backendResourceSuffix string
}

func NewVolumeBuilder(agent *instanav1.InstanaAgent, isOpenShift bool) VolumeBuilder {
	return &volumeBuilder{
		instanaAgent: agent,
		helpers:      helpers.NewHelpers(agent),
		isOpenShift:  isOpenShift,
	}
}

func (v *volumeBuilder) WithBackendResourceSuffix(suffix string) VolumeBuilder {
	v.backendResourceSuffix = suffix
	return v
}

func (v *volumeBuilder) Build(volumes ...Volume) ([]corev1.Volume, []corev1.VolumeMount) {
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

func (v *volumeBuilder) BuildFromUserConfig() ([]corev1.Volume, []corev1.VolumeMount) {
	return v.instanaAgent.Spec.Agent.Pod.Volumes, v.instanaAgent.Spec.Agent.Pod.VolumeMounts
}

func (v *volumeBuilder) getBuilder(volume Volume) (*corev1.Volume, *corev1.VolumeMount) {
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
			!v.isOpenShift,
			"var-run-kubo",
			"/var/vcap/sys/run/docker",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case VarRunContainerdVolume:
		return v.hostVolumeWithMountLiteralWhenCondition(
			!v.isOpenShift,
			"var-run-containerd",
			"/var/vcap/sys/run/containerd",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case VarContainerdConfigVolume:
		return v.hostVolumeWithMountLiteralWhenCondition(
			!v.isOpenShift,
			"var-containerd-config",
			"/var/vcap/jobs/containerd/config",
			&mountPropagationHostToContainer,
			&mkdir,
		)
	case SysVolume:
		return v.hostVolumeWithMount("sys", "/sys", &mountPropagationHostToContainer, nil)
	case VarLogVolume:
		return v.hostVolumeWithMount("var-log", "/var/log", &mountPropagationHostToContainer, nil)
	//case VarLibVolume:(Removed for CSP requirement)
	//return v.hostVolumeWithMount("var-lib", "/var/lib", &mountPropagationHostToContainer, nil)
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
	case NamespacesDetailsVolume:
		return v.namespacesDetailsVolume()
	case TlsVolume:
		return v.tlsVolume()
	case RepoVolume:
		return v.repoVolume()
	case SecretsVolume:
		return v.secretsVolume()
	case K8SensorSecretsVolume:
		return v.k8sensorSecretsVolume()
	case ETCDCAVolume:
		return v.etcdCAVolume()
	case ETCDClientCertVolume:
		return v.etcdClientCertVolume()
	case ControlPlaneCAVolume:
		return v.controlPlaneCAVolume()
	default:
		panic(errors.New("unknown volume requested"))
	}
}

func (v *volumeBuilder) hostVolumeWithMount(
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

func (v *volumeBuilder) hostVolumeWithMountLiteralWhenCondition(
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

func (v *volumeBuilder) configVolume() (*corev1.Volume, *corev1.VolumeMount) {
	volumeName := "config"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  v.instanaAgent.Name + "-config",
				DefaultMode: pointer.To[int32](0440),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: constants.InstanaConfigDirectory,
	}
	return &volume, &volumeMount
}

func (v *volumeBuilder) namespacesDetailsVolume() (*corev1.Volume, *corev1.VolumeMount) {
	volumeName := "namespaces-details"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			ConfigMap: &corev1.ConfigMapVolumeSource{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: v.instanaAgent.Name + "-namespaces",
				},
				DefaultMode: pointer.To[int32](0440),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: constants.InstanaNamespacesDetailsDirectory,
	}
	return &volume, &volumeMount
}

func (v *volumeBuilder) tlsVolume() (*corev1.Volume, *corev1.VolumeMount) {
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

func (v *volumeBuilder) repoVolume() (*corev1.Volume, *corev1.VolumeMount) {
	if v.instanaAgent.Spec.Agent.Host.Repository == "" {
		return nil, nil
	}
	volumeName := "repo"
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			HostPath: &corev1.HostPathVolumeSource{
				Path: v.instanaAgent.Spec.Agent.Host.Repository,
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: "/opt/instana/agent/data/repo",
	}

	return &volume, &volumeMount
}

func (v *volumeBuilder) k8sensorSecretsVolume() (*corev1.Volume, *corev1.VolumeMount) {
	// Only create the secrets volume if useSecretMounts is enabled or nil (default to true)
	if v.instanaAgent.Spec.UseSecretMounts != nil && !*v.instanaAgent.Spec.UseSecretMounts {
		return nil, nil
	}

	volumeName := "instana-secrets"
	secretName := v.instanaAgent.Spec.Agent.KeysSecret
	if secretName == "" {
		secretName = v.instanaAgent.Name
	}

	// Create a volume with specific items for k8sensor
	agentKeySecretKey := constants.SecretFileAgentKey + v.backendResourceSuffix
	proxySecretKey := constants.SecretFileHttpsProxy

	// When the user provides an external secret, the keys follow the "key", "key-1" pattern.
	// Remap them to the expected file names so run.sh can find INSTANA_AGENT_KEY.
	if v.instanaAgent.Spec.Agent.KeysSecret != "" {
		agentKeySecretKey = constants.AgentKey + v.backendResourceSuffix
		proxySecretKey = constants.SecretKeyHttpsProxy
	}

	items := []corev1.KeyToPath{
		{
			Key:  agentKeySecretKey,
			Path: constants.SecretFileAgentKey + v.backendResourceSuffix,
		},
	}

	// Only include HTTPS_PROXY if ProxyHost is set
	if v.instanaAgent.Spec.Agent.ProxyHost != "" {
		items = append(items, corev1.KeyToPath{
			Key:  proxySecretKey,
			Path: constants.SecretFileHttpsProxy,
		})
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: pointer.To[int32](0400), // Read-only for owner
				Items:       items,
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

func (v *volumeBuilder) secretsVolume() (*corev1.Volume, *corev1.VolumeMount) {
	// Only create the secrets volume if useSecretMounts is enabled or nil (default to true)
	if v.instanaAgent.Spec.UseSecretMounts != nil && !*v.instanaAgent.Spec.UseSecretMounts {
		return nil, nil
	}

	volumeName := "instana-secrets"
	secretName := v.instanaAgent.Spec.Agent.KeysSecret
	if secretName == "" {
		secretName = v.instanaAgent.Name
	}

	var items []corev1.KeyToPath
	if v.instanaAgent.Spec.Agent.KeysSecret != "" {
		agentKeySecretKey := constants.AgentKey + v.backendResourceSuffix
		items = append(items, corev1.KeyToPath{
			Key:  agentKeySecretKey,
			Path: constants.SecretFileAgentKey,
		})
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName:  secretName,
				DefaultMode: pointer.To[int32](0400), // Read-only for owner
				Items:       items,
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

func (v *volumeBuilder) etcdCAVolume() (*corev1.Volume, *corev1.VolumeMount) {
	if v.instanaAgent.Spec.K8sSensor.ETCD.CA.SecretName == "" {
		// For OpenShift, use the etcd-metrics-ca-bundle ConfigMap from openshift-etcd namespace
		if v.isOpenShift {
			volumeName := constants.ETCDCASecretName
			volume := corev1.Volume{
				Name: volumeName,
				VolumeSource: corev1.VolumeSource{
					ConfigMap: &corev1.ConfigMapVolumeSource{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: constants.ETCDMetricsCABundleName,
						},
						Items: []corev1.KeyToPath{
							{
								Key:  constants.ETCDCABundleFileName,
								Path: constants.ETCDCABundleFileName,
							},
						},
						DefaultMode: pointer.To[int32](0440),
					},
				},
			}
			volumeMount := corev1.VolumeMount{
				Name:      volumeName,
				MountPath: constants.ETCDMetricsCAMountPath,
				ReadOnly:  true,
			}
			return &volume, &volumeMount
		}
		return nil, nil
	}

	// For custom CA from secret
	volumeName := "etcd-ca"
	filename := v.instanaAgent.Spec.K8sSensor.ETCD.CA.Filename
	if filename == "" {
		filename = "ca.crt"
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: v.instanaAgent.Spec.K8sSensor.ETCD.CA.SecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  filename,
						Path: "ca.crt",
					},
				},
				DefaultMode: pointer.To[int32](0440),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: v.instanaAgent.Spec.K8sSensor.ETCD.CA.MountPath,
		ReadOnly:  true,
	}
	return &volume, &volumeMount
}

func (v *volumeBuilder) etcdClientCertVolume() (*corev1.Volume, *corev1.VolumeMount) {
	// Only for OpenShift - mount etcd-metric-client secret from openshift-etcd namespace
	if !v.isOpenShift {
		return nil, nil
	}

	volumeName := constants.ETCDClientCertSecretName
	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: constants.ETCDMetricClientSecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  constants.ETCDClientCertFileName,
						Path: constants.ETCDClientCertFileName,
					},
					{
						Key:  constants.ETCDClientKeyFileName,
						Path: constants.ETCDClientKeyFileName,
					},
				},
				DefaultMode: pointer.To[int32](0440),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: constants.ETCDClientCertMountPath,
		ReadOnly:  true,
	}
	return &volume, &volumeMount
}

func (v *volumeBuilder) controlPlaneCAVolume() (*corev1.Volume, *corev1.VolumeMount) {
	if v.instanaAgent.Spec.K8sSensor.RestClient.CA.SecretName == "" {
		return nil, nil
	}

	volumeName := "control-plane-ca"
	filename := v.instanaAgent.Spec.K8sSensor.RestClient.CA.Filename
	if filename == "" {
		filename = "ca.crt"
	}

	volume := corev1.Volume{
		Name: volumeName,
		VolumeSource: corev1.VolumeSource{
			Secret: &corev1.SecretVolumeSource{
				SecretName: v.instanaAgent.Spec.K8sSensor.RestClient.CA.SecretName,
				Items: []corev1.KeyToPath{
					{
						Key:  filename,
						Path: "ca.crt",
					},
				},
				DefaultMode: pointer.To[int32](0440),
			},
		},
	}
	volumeMount := corev1.VolumeMount{
		Name:      volumeName,
		MountPath: v.instanaAgent.Spec.K8sSensor.RestClient.CA.MountPath,
		ReadOnly:  true,
	}
	return &volume, &volumeMount
}
