/*
(c) Copyright IBM Corp. 2025
(c) Copyright Instana Inc. 2025
*/

package deployment

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/k8s/operator/status"
	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const componentName = constants.ComponentAutoTraceWebhook

type deploymentBuilder struct {
	*instanav1.InstanaAgent
	statusManager status.AgentStatusManager
	helpers       helpers.Helpers
	env.EnvBuilder
	isOpenShift bool
	transformations.PodSelectorLabelGenerator
}

func (d *deploymentBuilder) IsNamespaced() bool {
	return true
}

func (d *deploymentBuilder) ComponentName() string {
	return componentName
}

func (d *deploymentBuilder) getEnvVars() []corev1.EnvVar {
	envVars := d.EnvBuilder.Build(
		env.WebhookPodNamespace,
		env.WebhookPodName,
		env.WebhookSeverPort,
		env.WebhookInstanaIgnore,
		env.WebhookInstrumentationInitContainerImage,
		env.WebhookInstrumentationInitContainerPullPolicy,
		env.WebhookAutotraceNodejs,
		env.WebhookAutotraceNetcore,
		env.WebhookAutotraceRuby,
		env.WebhookAutotracePython,
		env.WebhookAutotraceAce,
		env.WebhookAutotraceIbmmq,
		env.WebhookAutotraceNodejsEsm,
		env.WebhookAutotraceNodejsAppType,
		env.WebhookAutotraceIngressNginx,
		env.WebhookAutotraceIngressNginxStatus,
		env.WebhookAutotraceIngressNginxStatusAllow,
		env.WebhookAutotraceLibInstanaInit,
		env.WebhookAutotraceInitMemoryLimit,
		env.WebhookAutotraceInitCPULimit,
		env.WebhookAutotraceInitMemoryRequest,
		env.WebhookAutotraceInitCPURequest,
		env.WebhookLogLevel,
	)
	return envVars
}

func addAppLabel(labels map[string]string) map[string]string {
	labelsDefaulter := map_defaulter.NewMapDefaulter(&labels)
	labelsDefaulter.SetIfEmpty("instana-autotrace-ignore", "true")
	return labels
}

func (d *deploymentBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Labels).GetOrDefault(map[string]string{})
	podLabels[constants.LabelAgentMode] = string(optional.Of(d.InstanaAgent.Spec.Agent.Mode).GetOrDefault(instanav1.APM))

	return d.GetPodLabels(podLabels)
}

// TODO: allow for multiple pull secrets?
func (d *deploymentBuilder) getWebhookImagePullSecret() []corev1.LocalObjectReference {
	var secretName string
	if d.InstanaAgent.Spec.AutotraceWebhook.PullSecret != "" {
		secretName = d.InstanaAgent.Spec.AutotraceWebhook.PullSecret
	} else {
		secretName = "containers-instana-io"
	}
	return []corev1.LocalObjectReference{
		{
			Name: secretName,
		},
	}
}

func (d *deploymentBuilder) getSecurityContext() *corev1.SecurityContext {
	securityContext := &corev1.SecurityContext{
		Privileged:               pointer.To(false),
		AllowPrivilegeEscalation: pointer.To(false),
		Capabilities: &corev1.Capabilities{
			Drop: []corev1.Capability{"ALL"},
		},
	}

	if d.isOpenShift {
		securityContext.SeccompProfile = &corev1.SeccompProfile{
			Type: corev1.SeccompProfileTypeRuntimeDefault,
		}
		securityContext.ReadOnlyRootFilesystem = pointer.To(true)
		securityContext.RunAsNonRoot = pointer.To(true)
		securityContext.RunAsUser = int64Ptr(1001)
	}

	return securityContext
}

func int64Ptr(i int64) *int64 {
	return &i
}

func (d *deploymentBuilder) build() *appsv1.Deployment {

	fmt.Println("debug starting")
	if d == nil {
		fmt.Println("d is nil")
	}
	if d.helpers.AutotraceWebhookResourcesName() == "" {
		fmt.Println("AutotraceWebhookResourcesName is nil")
	}
	fmt.Println("d.Spec.AutotraceWebhook.Replicas is ")
	fmt.Println(d.Spec.AutotraceWebhook.Replicas)
	if d.GetPodSelectorLabels() == nil {
		fmt.Println("GetPodSelectorLabels is nil")
	}
	if addAppLabel(nil) == nil {
		fmt.Println("addAppLabel is nil")
	}
	if addAppLabel(d.getPodTemplateLabels()) == nil {
		fmt.Println("addAppLabel(d.getPodTemplateLabels()) is nil")
	}
	if d.getWebhookImagePullSecret() == nil {
		fmt.Println("d.getWebhookImagePullSecret()")
	}
	if d.getSecurityContext() == nil {
		fmt.Println("d.getSecurityContext()")
	}

	return &appsv1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployment",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.helpers.AutotraceWebhookResourcesName(),
			Namespace: d.helpers.AutotraceWebhookResourcesName(),
			Labels:    addAppLabel(nil),
		},
		Spec: appsv1.DeploymentSpec{
			Replicas: pointer.To(int32(d.Spec.AutotraceWebhook.Replicas)),
			Selector: &metav1.LabelSelector{
				MatchLabels: addAppLabel(d.GetPodSelectorLabels()),
			},
			Strategy: appsv1.DeploymentStrategy{
				Type: appsv1.RecreateDeploymentStrategyType,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Name:        d.helpers.AutotraceWebhookResourcesName(),
					Labels:      addAppLabel(d.getPodTemplateLabels()), // TODO: add additional labels
					Annotations: d.Spec.Agent.Pod.Annotations,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.helpers.AutotraceWebhookResourcesName(),
					ImagePullSecrets:   d.getWebhookImagePullSecret(),
					Containers: []corev1.Container{
						{
							Name:            d.helpers.AutotraceWebhookResourcesName(),
							Image:           d.Spec.AutotraceWebhook.ImageSpec.Image(),
							ImagePullPolicy: d.Spec.AutotraceWebhook.ImageSpec.PullPolicy,
							SecurityContext: d.getSecurityContext(),
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceLimitsMemory: resource.MustParse("512Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceLimitsMemory: resource.MustParse("1Gi"),
								},
							},
							Env: d.getEnvVars(),
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "certificates",
									MountPath: "/app/certs",
								},
							},
							Ports: []corev1.ContainerPort{
								{
									ContainerPort: 42650,
								},
							},
							// skipping tolerations and affinity for now

						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "certificates",
							VolumeSource: corev1.VolumeSource{
								Secret: &corev1.SecretVolumeSource{
									SecretName: "instana-autotrace-webhook-certs",
								},
							},
						},
					},
				},
			},
		},
	}
}

func (d *deploymentBuilder) Build() (res optional.Optional[client.Object]) {
	defer func() {
		res.IfPresent(
			func(dpl client.Object) {
				d.statusManager.SetAutoTraceWebhookDeployment(client.ObjectKeyFromObject(dpl))
			},
		)
	}()

	// TODO: introduce webhook secret
	switch d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "" {
	case true:
		return optional.Empty[client.Object]()
	default:
		return optional.Of[client.Object](d.build())
	}
}

func NewWebhookBuilder(
	agent *instanav1.InstanaAgent,
	isOpenShift bool,
	statusManager status.AgentStatusManager,
) builder.ObjectBuilder {
	return &deploymentBuilder{
		InstanaAgent:              agent,
		helpers:                   helpers.NewHelpers(agent),
		EnvBuilder:                env.NewEnvBuilder(agent, nil),
		isOpenShift:               isOpenShift,
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		statusManager:             statusManager,
	}
}
