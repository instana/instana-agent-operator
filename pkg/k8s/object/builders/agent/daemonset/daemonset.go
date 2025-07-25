/*
(c) Copyright IBM Corp. 2024, 2025
*/

package daemonset

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/hash"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const (
	componentName = constants.ComponentInstanaAgent
)

func NewDaemonSetBuilder(
	agent *instanav1.InstanaAgent,
	isOpenshift bool,
	statusManager status.AgentStatusManager,
) builder.ObjectBuilder {
	return NewDaemonSetBuilderWithZoneInfo(agent, isOpenshift, statusManager, nil)
}

func NewDaemonSetBuilderWithZoneInfo(
	agent *instanav1.InstanaAgent,
	isOpenshift bool,
	statusManager status.AgentStatusManager,
	zone *instanav1.Zone,
) builder.ObjectBuilder {
	return &daemonSetBuilder{
		InstanaAgent:  agent,
		statusManager: statusManager,

		PodSelectorLabelGenerator: transformations.PodSelectorLabelsWithZoneInfo(agent, componentName, zone),
		JsonHasher:                hash.NewJsonHasher(),
		Helpers:                   helpers.NewHelpers(agent),
		portsBuilder:              ports.NewPortsBuilder(agent.Spec.OpenTelemetry),
		EnvBuilder:                env.NewEnvBuilder(agent, zone),
		VolumeBuilder:             volume.NewVolumeBuilder(agent, isOpenshift),
		zone:                      zone,
	}
}

type daemonSetBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager

	transformations.PodSelectorLabelGenerator
	hash.JsonHasher
	helpers.Helpers
	env.EnvBuilder
	volume.VolumeBuilder

	portsBuilder ports.PortsBuilder
	zone         *instanav1.Zone
}

func (d *daemonSetBuilder) ComponentName() string {
	return componentName
}

func (d *daemonSetBuilder) IsNamespaced() bool {
	return true
}

func (d *daemonSetBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Labels).GetOrDefault(map[string]string{})
	podLabels[constants.LabelAgentMode] = string(optional.Of(d.InstanaAgent.Spec.Agent.Mode).GetOrDefault(instanav1.APM))

	return d.GetPodLabels(podLabels)
}

func (d *daemonSetBuilder) getEnvVars() []corev1.EnvVar {
	// Get environment variables from the env builder (includes agent.env vars)
	baseEnvVars := d.EnvBuilder.Build(
		env.AgentModeEnv,
		env.ZoneNameEnv,
		env.ClusterNameEnv,
		env.AgentEndpointEnv,
		env.AgentEndpointPortEnv,
		env.MavenRepoURLEnv,
		env.MavenRepoFeaturesPath,
		env.MavenRepoSharedPath,
		env.MirrorReleaseRepoUrlEnv,
		env.MirrorReleaseRepoUsernameEnv,
		env.MirrorReleaseRepoPasswordEnv,
		env.MirrorSharedRepoUrlEnv,
		env.MirrorSharedRepoUsernameEnv,
		env.MirrorSharedRepoPasswordEnv,
		env.ProxyHostEnv,
		env.ProxyPortEnv,
		env.ProxyProtocolEnv,
		env.ProxyUserEnv,
		env.ProxyPasswordEnv,
		env.ProxyUseDNSEnv,
		env.ListenAddressEnv,
		env.RedactK8sSecretsEnv,
		env.ConfigPathEnv,
		env.EntrypointSkipBackendTemplateGeneration,
		env.InstanaAgentKeyEnv,
		env.DownloadKeyEnv,
		env.InstanaAgentPodNameEnv,
		env.PodIPEnv,
		env.K8sServiceDomainEnv,
		env.EnableAgentSocketEnv,
		env.NamespacesDetailsPathEnv,
	)

	// Create a map to track environment variables by name to ensure no duplicates
	// and to ensure pod.env values take precedence over agent.env values
	envVarMap := make(map[string]corev1.EnvVar)

	// Add base environment variables to the map
	for _, envVar := range baseEnvVars {
		envVarMap[envVar.Name] = envVar
	}

	// Add user-defined environment variables from the pod.env field
	// These will overwrite any existing variables with the same name
	for _, envVar := range d.InstanaAgent.Spec.Agent.Pod.Env {
		envVarMap[envVar.Name] = envVar
	}

	// Convert the map back to a slice
	result := make([]corev1.EnvVar, 0, len(envVarMap))
	for _, envVar := range envVarMap {
		result = append(result, envVar)
	}

	// Sort the environment variables by name for consistency
	d.Helpers.SortEnvVarsByName(result)
	return result
}

func (d *daemonSetBuilder) getVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilder.Build(
		volume.DevVolume,
		volume.RunVolume,
		volume.VarRunVolume,
		volume.VarRunKuboVolume,
		volume.VarRunContainerdVolume,
		volume.VarContainerdConfigVolume,
		volume.SysVolume,
		volume.VarLogVolume,
		volume.VarLibVolume,
		volume.VarDataVolume,
		volume.MachineIdVolume,
		volume.ConfigVolume,
		volume.TlsVolume,
		volume.RepoVolume,
		volume.NamespacesDetailsVolume,
	)
}

func (d *daemonSetBuilder) getUserVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilder.BuildFromUserConfig()
}

func (d *daemonSetBuilder) getName() string {
	switch d.zone {
	case nil:
		return d.InstanaAgent.Name
	default:
		return fmt.Sprintf("%s-%s", d.InstanaAgent.Name, d.zone.Name.Name)
	}
}

func (d *daemonSetBuilder) getNonStandardLabels() map[string]string {
	switch d.zone {
	case nil:
		return nil
	default:
		return map[string]string{
			transformations.ZoneLabel: d.zone.Name.Name,
		}
	}
}

func (d *daemonSetBuilder) getAffinity() *corev1.Affinity {
	switch d.zone {
	case nil:
		return &d.InstanaAgent.Spec.Agent.Pod.Affinity
	default:
		return &d.zone.Affinity
	}
}

func (d *daemonSetBuilder) getTolerations() []corev1.Toleration {
	switch d.zone {
	case nil:
		return d.InstanaAgent.Spec.Agent.Pod.Tolerations
	default:
		return d.zone.Tolerations
	}
}

func (d *daemonSetBuilder) build() *appsv1.DaemonSet {
	volumes, volumeMounts := d.getVolumes()
	userVolumes, userVolumeMounts := d.getUserVolumes()

	return &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.getName(),
			Namespace: d.Namespace,
			Labels:    d.getNonStandardLabels(),
		},
		Spec: appsv1.DaemonSetSpec{
			MinReadySeconds: int32(d.Spec.Agent.MinReadySeconds),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.GetPodSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.InstanaAgent.Spec.Agent.Pod.Annotations,
				},
				Spec: corev1.PodSpec{
					Volumes:            append(volumes, userVolumes...),
					ServiceAccountName: d.ServiceAccountName(),
					NodeSelector:       d.Spec.Agent.Pod.NodeSelector,
					HostNetwork:        true,
					HostPID:            true,
					PriorityClassName:  d.Spec.Agent.Pod.PriorityClassName,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					ImagePullSecrets:   d.ImagePullSecrets(),
					Containers: []corev1.Container{
						{
							Name:            "instana-agent",
							Image:           d.Spec.Agent.Image(),
							ImagePullPolicy: d.Spec.Agent.PullPolicy,
							VolumeMounts:    append(volumeMounts, userVolumeMounts...),
							Env:             d.getEnvVars(),
							SecurityContext: &corev1.SecurityContext{
								Privileged: pointer.To(true),
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Host: "127.0.0.1",
										Path: "/status",
										Port: intstr.FromInt32(ports.InstanaAgentAPIPortConfig.Port),
									},
								},
								InitialDelaySeconds: 600,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								FailureThreshold:    3,
							},
							Resources: d.Spec.Agent.Pod.ResourceRequirements.GetOrDefault(),
							Ports:     d.portsBuilder.GetContainerPorts(),
						},
					},
					Tolerations: d.getTolerations(),
					Affinity:    d.getAffinity(),
				},
			},
			UpdateStrategy: d.InstanaAgent.Spec.Agent.UpdateStrategy,
		},
	}
}

func (d *daemonSetBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(ds client.Object) {
				d.statusManager.AddAgentDaemonset(client.ObjectKeyFromObject(ds))
			},
		)
	}()

	switch {
	case d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "":
		fallthrough
	case d.zone == nil && d.Spec.Zone.Name == "" && d.Spec.Cluster.Name == "":
		fallthrough
	case d.zone != nil && d.Spec.Cluster.Name == "":
		return optional.Empty[client.Object]()
	default:
		return optional.Of[client.Object](d.build())
	}
}
