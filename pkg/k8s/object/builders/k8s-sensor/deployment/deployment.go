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
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const componentName = constants.ComponentK8Sensor

type deploymentBuilder struct {
	*instanav1.InstanaAgent

	helpers.Helpers
	transformations.PodSelectorLabelGenerator
	env.EnvBuilder
	volume.VolumeBuilder
	ports.PortsBuilder
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
	return d.EnvBuilder.Build(
		env.AgentModeEnv,
		env.BackendEnv,
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
}

func (d *deploymentBuilder) build() *appsv1.Deployment {
	volumes, mounts := d.VolumeBuilder.Build(volume.ConfigVolume)

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.K8sSensorResourcesName(),
			Namespace: d.Namespace,
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: d.Spec.K8sSensor.DeploymentSpec.Replicas.GetDesired(),
			Selector: &metav1.LabelSelector{
				MatchLabels: d.GetPodSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.Spec.Agent.Pod.Annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.K8sSensorResourcesName(),
					NodeSelector:       d.Spec.K8sSensor.DeploymentSpec.Pod.NodeSelector,
					PriorityClassName:  d.Spec.K8sSensor.DeploymentSpec.Pod.PriorityClassName,
					ImagePullSecrets:   d.ImagePullSecrets(),
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

func (d *deploymentBuilder) Build() optional.Optional[client.Object] {
	switch (d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "") || (d.Spec.Zone.Name == "" && d.Spec.Cluster.Name == "") {
	case true:
		return optional.Empty[client.Object]()
	default:
		return optional.Of[client.Object](d.build())
	}
}

func NewDeploymentBuilder(agent *instanav1.InstanaAgent, isOpenShift bool) builder.ObjectBuilder {
	return &deploymentBuilder{
		InstanaAgent:              agent,
		Helpers:                   helpers.NewHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		EnvBuilder:                env.NewEnvBuilder(agent),
		VolumeBuilder:             volume.NewVolumeBuilder(agent, isOpenShift),
		PortsBuilder:              ports.NewPortsBuilder(agent),
	}
}
