/*
(c) Copyright IBM Corp. 2024
(c) Copyright Instana Inc. 2024
*/

package deployment

import (
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/ports"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/volume"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const componentName = constants.ComponentK8Sensor

type deploymentBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager

	helpers helpers.Helpers
	transformations.PodSelectorLabelGenerator
	env.EnvBuilder
	volume.VolumeBuilder
	ports.PortsBuilder
	backendResourceSuffix string
}

func (d *deploymentBuilder) IsNamespaced() bool {
	return true
}

func (d *deploymentBuilder) ComponentName() string {
	return componentName
}

func (d *deploymentBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.Spec.Agent.Pod.Labels).GetOrDefault(make(map[string]string, 3))
	podLabels[constants.LabelAgentMode] = string(instanav1.KUBERNETES)
	return d.GetPodLabels(podLabels)
}

func (d *deploymentBuilder) getEnvVars() []corev1.EnvVar {
	envVars := d.EnvBuilder.Build(
		env.BackendURLEnv,
		env.AgentZoneEnv,
		env.PodUIDEnv,
		env.PodNamespaceEnv,
		env.PodNameEnv,
		env.PodIPEnv,
		env.HTTPSProxyEnv,
		env.NoProxyEnv,
		env.RedactK8sSecretsEnv,
		env.ConfigPathEnv,
	)
	backendEnvVars := []corev1.EnvVar{
		{
			Name: "BACKEND",
			ValueFrom: &corev1.EnvVarSource{
				ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: d.helpers.K8sSensorResourcesName() + d.backendResourceSuffix,
					},
					Key: constants.BackendKey,
				},
			},
		},
		{
			Name: "AGENT_KEY",
			ValueFrom: &corev1.EnvVarSource{
				SecretKeyRef: &corev1.SecretKeySelector{
					LocalObjectReference: corev1.LocalObjectReference{
						Name: optional.Of(d.Spec.Agent.KeysSecret).GetOrDefault(d.Name) + d.backendResourceSuffix,
					},
					Key: constants.AgentKey,
				},
			},
		},
	}
	return append(backendEnvVars, envVars...)
}

func (d *deploymentBuilder) getVolumes() ([]corev1.Volume, []corev1.VolumeMount) {
	return d.VolumeBuilder.Build(volume.ConfigVolume)
}

// K8Sensor relies on this label for internal sharding logic for some reason, if you remove it the k8sensor will break
func addAppLabel(labels map[string]string) map[string]string {
	labelsDefaulter := map_defaulter.NewMapDefaulter(&labels)
	labelsDefaulter.SetIfEmpty("app", "k8sensor")
	return labels
}

func (d *deploymentBuilder) build() *appsv1.Deployment {
	volumes, mounts := d.getVolumes()

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.helpers.K8sSensorResourcesName() + d.backendResourceSuffix,
			Namespace: d.Namespace,
			Labels:    addAppLabel(nil),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas:        pointer.To(int32(d.Spec.K8sSensor.DeploymentSpec.Replicas)),
			MinReadySeconds: int32(d.Spec.K8sSensor.DeploymentSpec.MinReadySeconds),
			Selector: &metav1.LabelSelector{
				MatchLabels: addAppLabel(d.GetPodSelectorLabels()),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      addAppLabel(d.getPodTemplateLabels()),
					Annotations: d.Spec.Agent.Pod.Annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.helpers.K8sSensorResourcesName(),
					NodeSelector:       d.Spec.K8sSensor.DeploymentSpec.Pod.NodeSelector,
					PriorityClassName:  d.Spec.K8sSensor.DeploymentSpec.Pod.PriorityClassName,
					ImagePullSecrets:   d.helpers.ImagePullSecrets(),
					Containers: []corev1.Container{
						{
							Name:            "instana-agent",
							Image:           d.Spec.K8sSensor.ImageSpec.Image(),
							ImagePullPolicy: d.Spec.K8sSensor.ImageSpec.PullPolicy,
							Env:             d.getEnvVars(),
							VolumeMounts:    mounts,
							Resources:       d.Spec.K8sSensor.DeploymentSpec.Pod.ResourceRequirements.GetOrDefault(),
							Ports:           d.PortsBuilder.GetContainerPorts(ports.AgentAPIsPort),
						},
					},
					Volumes:     volumes,
					Tolerations: d.Spec.K8sSensor.DeploymentSpec.Pod.Tolerations,
					Affinity: pointer.To(
						optional.Of(d.Spec.K8sSensor.DeploymentSpec.Pod.Affinity).GetOrDefault(
							corev1.Affinity{
								PodAntiAffinity: &corev1.PodAntiAffinity{
									PreferredDuringSchedulingIgnoredDuringExecution: []corev1.WeightedPodAffinityTerm{
										{
											Weight: 100,
											PodAffinityTerm: corev1.PodAffinityTerm{
												LabelSelector: &metav1.LabelSelector{
													MatchExpressions: []metav1.LabelSelectorRequirement{
														{
															Key:      constants.LabelAgentMode,
															Operator: metav1.LabelSelectorOpIn,
															Values:   []string{string(instanav1.KUBERNETES)},
														},
													},
												},
												TopologyKey: corev1.LabelHostname,
											},
										},
									},
								},
							},
						),
					),
				},
			},
		},
	}
}

func (d *deploymentBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(dpl client.Object) {
				d.statusManager.SetK8sSensorDeployment(client.ObjectKeyFromObject(dpl))
			},
		)
	}()

	switch (d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "") || (d.Spec.Zone.Name == "" && d.Spec.Cluster.Name == "") {
	case true:
		return optional.Empty[client.Object]()
	default:
		return optional.Of[client.Object](d.build())
	}
}

func NewDeploymentBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
	backendResourceSuffix string,
) builder.ObjectBuilder {
	return &deploymentBuilder{
		InstanaAgent:              agent,
		statusManager:             statusManager,
		helpers:                   helpers.NewHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		EnvBuilder:                env.NewEnvBuilder(agent, nil),
		VolumeBuilder:             volume.NewVolumeBuilder(agent, isOpenShift),
		PortsBuilder:              ports.NewPortsBuilder(agent),
		backendResourceSuffix:     backendResourceSuffix,
	}
}
