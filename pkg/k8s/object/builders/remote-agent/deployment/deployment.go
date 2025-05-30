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
package deployment

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/hash"
	backends "github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/backends"
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
	componentName = constants.ComponentRemoteAgent
)

type deploymentBuilder struct {
	*instanav1.RemoteAgent
	statusManager status.RemoteAgentStatusManager
	helpers.RemoteHelpers
	transformations.PodSelectorLabelGenerator
	hash.JsonHasher
	ports.PortsBuilderRemote
	env.EnvBuilderRemote
	volume.VolumeBuilderRemote
	backend    backends.K8SensorBackend
	keysSecret *corev1.Secret
	zone       *instanav1.Zone
}

func (d *deploymentBuilder) ComponentName() string {
	return componentName
}

func (d *deploymentBuilder) IsNamespaced() bool {
	return true
}

func (d *deploymentBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.RemoteAgent.Spec.Agent.Pod.Labels).GetOrDefault(map[string]string{})
	podLabels[constants.LabelAgentMode] = string(optional.Of(d.RemoteAgent.Spec.Agent.Mode).GetOrDefault(instanav1.APM))

	return d.GetPodLabels(podLabels)
}

func (d *deploymentBuilder) getEnvVars() []corev1.EnvVar {
	envVars := d.EnvBuilderRemote.Build(
		env.AgentModeEnvRemote,
		env.ZoneNameEnvRemote,
		env.ClusterNameEnvRemote,
		env.AgentEndpointEnvRemote,
		env.AgentEndpointPortEnvRemote,
		env.MavenRepoURLEnvRemote,
		env.MavenRepoFeaturesPathRemote,
		env.MavenRepoSharedPathRemote,
		env.MirrorReleaseRepoUrlEnvRemote,
		env.MirrorReleaseRepoUsernameEnvRemote,
		env.MirrorReleaseRepoPasswordEnvRemote,
		env.MirrorSharedRepoUrlEnvRemote,
		env.MirrorSharedRepoUsernameEnvRemote,
		env.MirrorSharedRepoPasswordEnvRemote,
		env.ProxyHostEnvRemote,
		env.ProxyPortEnvRemote,
		env.ProxyProtocolEnvRemote,
		env.ProxyUserEnvRemote,
		env.ProxyPasswordEnvRemote,
		env.ProxyUseDNSEnvRemote,
		env.ListenAddressEnvRemote,
		env.RedactK8sSecretsEnvRemote,
		env.ConfigPathEnvRemote,
		env.EntrypointSkipBackendTemplateGenerationRemote,
		env.InstanaAgentKeyEnvRemote,
		env.DownloadKeyEnvRemote,
		env.InstanaAgentPodNameEnvRemote,
		env.PodIPEnvRemote,
		env.K8sServiceDomainEnvRemote,
		env.EnableAgentSocketEnvRemote,
	)
	d.SortEnvVarsByName(envVars)
	return envVars
}

func (d *deploymentBuilder) getContainerPorts() []corev1.ContainerPort {
	return d.GetContainerPorts(
		ports.AgentAPIsPort,
	)
}

func (d *deploymentBuilder) getVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilderRemote.Build(
		volume.ConfigVolumeRemote,
		volume.TlsVolumeRemote,
		volume.RepoVolumeRemote,
	)
}

func (d *deploymentBuilder) getUserVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilderRemote.BuildFromUserConfig()
}

func (d *deploymentBuilder) getName() string {
	switch d.zone {
	case nil:
		return d.RemoteAgent.Name
	default:
		return fmt.Sprintf("%s-%s", d.RemoteAgent.Name, d.zone.Name.Name)
	}
}

func (d *deploymentBuilder) getNonStandardLabels() map[string]string {
	switch d.zone {
	case nil:
		return nil
	default:
		return map[string]string{
			transformations.ZoneLabel: d.zone.Name.Name,
		}
	}
}

func (d *deploymentBuilder) getAffinity() *corev1.Affinity {
	switch d.zone {
	case nil:
		return &d.RemoteAgent.Spec.Agent.Pod.Affinity
	default:
		return &d.zone.Affinity
	}
}

func (d *deploymentBuilder) getTolerations() []corev1.Toleration {
	switch d.zone {
	case nil:
		return d.RemoteAgent.Spec.Agent.Pod.Tolerations
	default:
		return d.zone.Tolerations
	}
}

func (d *deploymentBuilder) build() *appsv1.Deployment {
	volumes, volumeMounts := d.getVolumes()
	userVolumes, userVolumeMounts := d.getUserVolumes()
	name := fmt.Sprintf("remote-agent-%s", d.getName())

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: d.Namespace,
			Labels:    d.getNonStandardLabels(),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.To(int32(1)), // Set the number of replicas here
			Selector: &metav1.LabelSelector{
				MatchLabels: d.GetPodSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.RemoteAgent.Spec.Agent.Pod.Annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: "remote-agent",
					Volumes:            append(volumes, userVolumes...),
					NodeSelector:       d.Spec.Agent.Pod.NodeSelector,
					PriorityClassName:  d.Spec.Agent.Pod.PriorityClassName,
					DNSPolicy:          corev1.DNSClusterFirst,
					ImagePullSecrets:   d.ImagePullSecrets(),
					Containers: []corev1.Container{
						{
							Name:            d.getName(),
							Image:           d.Spec.Agent.Image(),
							ImagePullPolicy: d.Spec.Agent.PullPolicy,
							VolumeMounts:    append(volumeMounts, userVolumeMounts...),
							Env:             d.getEnvVars(),
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Host: "127.0.0.1",
										Path: "/status",
										Port: intstr.FromString(string(ports.AgentAPIsPort)),
									},
								},
								InitialDelaySeconds: 600,
								TimeoutSeconds:      5,
								PeriodSeconds:       10,
								FailureThreshold:    3,
							},
							Resources: d.Spec.Agent.Pod.GetOrDefault(),
							Ports:     d.getContainerPorts(),
						},
					},
					Tolerations: d.getTolerations(),
					Affinity:    d.getAffinity(),
				},
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RollingUpdateDeploymentStrategyType,
			},
		},
	}
}

func (d *deploymentBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(dp client.Object) {
				d.statusManager.AddAgentDeployment(client.ObjectKeyFromObject(dp))
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

func NewDeploymentBuilder(
	agent *instanav1.RemoteAgent,
	statusManager status.RemoteAgentStatusManager,
	backend backends.K8SensorBackend,
	keysSecret *corev1.Secret,
) builder.ObjectBuilder {
	return &deploymentBuilder{
		RemoteAgent:               agent,
		statusManager:             statusManager,
		RemoteHelpers:             helpers.NewRemoteHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabelsRemote(agent, componentName),
		EnvBuilderRemote:          env.NewEnvBuilderRemote(agent, nil),
		VolumeBuilderRemote:       volume.NewVolumeBuilderRemote(agent),
		PortsBuilderRemote:        ports.NewPortsBuilderRemote(agent),
		backend:                   backend,
		keysSecret:                keysSecret,
	}
}
