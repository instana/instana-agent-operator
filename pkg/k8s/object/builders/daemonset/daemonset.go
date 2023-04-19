package daemonset

import (
	"fmt"
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/hash"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/env"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/map_defaulter"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

// TODO: Test and finish

// TODO: use an init container to copy original contents of etc/instana to an empty volume then copy them onto cm mounted contents before executing run.sh

const (
	componentName = constants.ComponentInstanaAgent
)

type daemonSetBuilder struct {
	*instanav1.InstanaAgent
	isOpenShift bool

	transformations.PodSelectorLabelGenerator
	hash.JsonHasher
	helpers.Helpers
}

func (d *daemonSetBuilder) ComponentName() string {
	return componentName
}

func (d *daemonSetBuilder) IsNamespaced() bool {
	return true
}

func (d *daemonSetBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Labels).GetOrDefault(map[string]string{})
	podLabels["instana/agent-mode"] = string(optional.Of(d.InstanaAgent.Spec.Agent.Mode).GetOrDefault(instanav1.APM))

	return d.GetPodLabels(podLabels)
}

func (d *daemonSetBuilder) getPodTemplateAnnotations() map[string]string {
	podAnnotations := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Annotations).GetOrDefault(map[string]string{})
	podAnnotations["instana-configuration-hash"] = d.HashJsonOrDie(&d.Spec) // TODO: do we really need to restart on any change?
	return podAnnotations
}

func (d *daemonSetBuilder) getImagePullSecrets() []corev1.LocalObjectReference {
	res := d.Spec.Agent.ImageSpec.PullSecrets

	if strings.HasPrefix(d.Spec.Agent.ImageSpec.Name, constants.ContainersInstanaIoRegistry) {
		res = append(
			res, corev1.LocalObjectReference{
				Name: constants.ContainersInstanaIoSecretName,
			},
		)
	}

	return res
}

func (d *daemonSetBuilder) getEnvVars() []optional.Optional[corev1.EnvVar] {
	return append(
		[]optional.Optional[corev1.EnvVar]{
			env.AgentModeEnv(d.InstanaAgent),
			env.ZoneNameEnv(d.InstanaAgent),
			env.ClusterNameEnv(d.InstanaAgent),
			env.AgentEndpointEnv(d.InstanaAgent),
			env.AgentEndpointPortEnv(d.InstanaAgent),
			env.MavenRepoUrlEnv(d.InstanaAgent),
			env.ProxyHostEnv(d.InstanaAgent),
			env.ProxyPortEnv(d.InstanaAgent),
			env.ProxyProtocolEnv(d.InstanaAgent),
			env.ProxyUserEnv(d.InstanaAgent),
			env.ProxyPasswordEnv(d.InstanaAgent),
			env.ProxyUseDNSEnv(d.InstanaAgent),
			env.ListenAddressEnv(d.InstanaAgent),
			env.RedactK8sSecretsEnv(d.InstanaAgent),
			env.AgentKeyEnv(d.Helpers),
			env.DownloadKeyEnv(d.Helpers),
			env.PodNameEnv(),
			env.PodIpEnv(),
			env.K8sServiceDomainEnv(d.InstanaAgent, d.Helpers),
		},
		env.UserProvidedEnv(d.InstanaAgent)...,
	)
}

// TODO: Add Volumes and VolumeMounts once fully done

func (d *daemonSetBuilder) getResourceRequirements() corev1.ResourceRequirements {
	res := d.Spec.Agent.Pod.ResourceRequirements

	requestsDefaulter := map_defaulter.NewMapDefaulter((*map[corev1.ResourceName]resource.Quantity)(&res.Requests))
	requestsDefaulter.SetIfEmpty(corev1.ResourceMemory, resource.MustParse("512Mi"))
	requestsDefaulter.SetIfEmpty(corev1.ResourceCPU, resource.MustParse("0.5"))

	limitsDefaulter := map_defaulter.NewMapDefaulter((*map[corev1.ResourceName]resource.Quantity)(&res.Limits))
	limitsDefaulter.SetIfEmpty(corev1.ResourceMemory, resource.MustParse("768Mi"))
	limitsDefaulter.SetIfEmpty(corev1.ResourceCPU, resource.MustParse("1.5"))

	return res
}

func (d *daemonSetBuilder) getContainerPorts() []corev1.ContainerPort {
	return []corev1.ContainerPort{
		{
			Name:          constants.AgentAPIsPort,
			ContainerPort: 42699,
		},
	} // TODO: Include other ports and build in a common way with service ports, etc. + Test
}

// TODO: test Build()

func (d *daemonSetBuilder) Build() optional.Optional[client.Object] {
	if d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "" {
		return optional.Empty[client.Object]()
	}
	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: d.GetPodSelectorLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.getPodTemplateAnnotations(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.ServiceAccountName(),
					NodeSelector:       d.Spec.Agent.Pod.NodeSelector,
					HostNetwork:        true, // TODO: Test for ServiceEntry later: may not be needed on 1.26+ with internal traffic policy
					HostPID:            true,
					PriorityClassName:  d.Spec.Agent.Pod.PriorityClassName,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					ImagePullSecrets:   d.getImagePullSecrets(),
					Containers: []corev1.Container{
						{
							Name:            "instana-agent",
							Image:           d.Spec.Agent.Image(),
							ImagePullPolicy: d.Spec.Agent.PullPolicy,
							Env:             optional.NewNonEmptyOptionalMapper[corev1.EnvVar]().AllNonEmpty(d.getEnvVars()),
							SecurityContext: &corev1.SecurityContext{
								Privileged: pointer.To(true),
							},
							LivenessProbe: &corev1.Probe{
								ProbeHandler: corev1.ProbeHandler{
									HTTPGet: &corev1.HTTPGetAction{
										Host: "127.0.0.1", // TODO: Because of HostNet usage supposedly, but shouldn't be needed I think
										Path: "/status",
										Port: intstr.FromString(constants.AgentAPIsPort),
									},
								},
								// TODO: set this long because startupProbe wasn't available before k8s 1.16, but this should be EOL by now, so we should see if we can revise this
								InitialDelaySeconds: 300,
								TimeoutSeconds:      3,
								PeriodSeconds:       10,
								FailureThreshold:    3,
							},
							// TODO: should have readiness probe too
							Resources: d.getResourceRequirements(),
							Ports:     d.getContainerPorts(),
						},
					},
					// TODO: Leader elector container should no longer be needed right?
					Tolerations: d.Spec.Agent.Pod.Tolerations,
					Affinity:    &d.Spec.Agent.Pod.Affinity,
				},
			},
			UpdateStrategy: d.InstanaAgent.Spec.Agent.UpdateStrategy,
		},
	}
	return optional.Of[client.Object](ds)
}

// TODO: Test this function and include multizone test case for Build()

func NewDaemonSetBuilder(
	agent *instanav1.InstanaAgent,
	isOpenshift bool,
	zone optional.Optional[instanav1.Zone],
) builder.ObjectBuilder {
	zone.IfPresent(
		func(zone instanav1.Zone) {
			agent = agent.DeepCopy()

			agent.Name = fmt.Sprintf("%s-%s", agent.Name, zone.Name.Name)
			agent.Spec.Zone.Name = zone.Name.Name
			agent.Spec.Agent.Pod.Tolerations = zone.Tolerations
			agent.Spec.Agent.Pod.Affinity = zone.Affinity
			agent.Spec.Agent.Mode = zone.Mode

		},
	)

	return &daemonSetBuilder{
		InstanaAgent:              agent,
		isOpenShift:               isOpenshift,
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
		JsonHasher:                hash.NewJsonHasher(),
		Helpers:                   helpers.NewHelpers(agent),
	}
}
