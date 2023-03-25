package daemonset

import (
	"strings"

	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/env"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/hash"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Multiple zones

// TODO: Test and finish

type daemonSetBuilder struct {
	*instanav1.InstanaAgent

	transformations.Transformations
	hash.Hasher
	helpers.Helpers
}

func NewDaemonSetBuilder(agent *instanav1.InstanaAgent) builders.ResourceBuilder {
	return &daemonSetBuilder{
		InstanaAgent:    agent,
		Transformations: transformations.NewTransformations(agent),
		Hasher:          hash.NewHasher(),
		Helpers:         helpers.NewHelpers(agent),
	}
}

func (d *daemonSetBuilder) getSelectorMatchLabels() map[string]string {
	return d.AddCommonLabelsToMap(map[string]string{}, d.Name, true)
}

func (d *daemonSetBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Labels).GetOrElse(map[string]string{})
	podLabels["instana/agent-mode"] = string(optional.Of(d.InstanaAgent.Spec.Agent.Mode).GetOrElse(instanav1.APM))
	return d.AddCommonLabelsToMap(podLabels, d.Name, false)
}

func (d *daemonSetBuilder) getPodTemplateAnnotations() map[string]string {
	podAnnotations := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Annotations).GetOrElse(map[string]string{})
	podAnnotations["instana-configuration-hash"] = d.HashOrDie(&d.Spec)
	return podAnnotations
}

func (d *daemonSetBuilder) getImagePullSecrets() []corev1.LocalObjectReference {
	res := d.Spec.Agent.ImageSpec.PullSecrets

	if strings.HasPrefix(d.Spec.Agent.ImageSpec.Name, builders.ContainersInstanaIoRegistry) {
		res = append(res, corev1.LocalObjectReference{
			Name: builders.ContainersInstanaIoSecretName,
		})
	}

	return res
}

func (d *daemonSetBuilder) getEnvBuilders() []env.EnvBuilder {
	return append(
		[]env.EnvBuilder{
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
		},
		env.UserProvidedEnv(d.InstanaAgent)...,
	)
}

func (d *daemonSetBuilder) Build() optional.Optional[client.Object] {
	if d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "" {
		return optional.Empty[client.Object]()
	}
	ds := &appsv1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/appsv1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
		},
		Spec: appsv1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: d.getSelectorMatchLabels(),
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.getPodTemplateAnnotations(),
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: d.ServiceAccountName(),
					NodeSelector:       d.Spec.Agent.Pod.NodeSelector,
					HostNetwork:        true,
					HostPID:            true,
					PriorityClassName:  d.Spec.Agent.Pod.PriorityClassName,
					DNSPolicy:          corev1.DNSClusterFirstWithHostNet,
					ImagePullSecrets:   d.getImagePullSecrets(),
					Containers: []corev1.Container{
						{
							Name:            "instana-agent",
							Image:           d.Spec.Agent.Image(),
							ImagePullPolicy: d.Spec.Agent.PullPolicy,
							//Env:             d.getEnvVars(),
						},
					},
				},
			},
			UpdateStrategy: d.InstanaAgent.Spec.Agent.UpdateStrategy,
		},
	}
	return optional.Of[client.Object](ds)
}
