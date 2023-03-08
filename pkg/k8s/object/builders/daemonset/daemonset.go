package daemonset

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
	v1 "k8s.io/api/apps/v1"
	coreV1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// TODO: Multiple zones

// TODO: Test and finish

type DaemonSetBuilder interface {
	Build() optional.Optional[client.Object]
}

type daemonSetBuilder struct {
	*instanav1.InstanaAgent
	instanaConfigurationHash string
}

func NewDaemonSetBuilder(agent *instanav1.InstanaAgent, instanaConfigurationHash string) DaemonSetBuilder {
	return &daemonSetBuilder{
		InstanaAgent:             agent,
		instanaConfigurationHash: instanaConfigurationHash,
	}
}

func (d *daemonSetBuilder) getSelectorMatchLabels() map[string]string {
	return transformations.AddCommonLabels(map[string]string{}, d.Name, true)
}

func (d *daemonSetBuilder) getPodTemplateLabels() map[string]string {
	podLabels := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Labels).GetOrElse(map[string]string{})
	podLabels["instana/agent-mode"] = string(optional.Of(d.InstanaAgent.Spec.Agent.Mode).GetOrElse(instanav1.APM))
	return transformations.AddCommonLabels(podLabels, d.Name, false)
}

func (d *daemonSetBuilder) getPodTemplateAnnotations() map[string]string {
	podAnnotations := optional.Of(d.InstanaAgent.Spec.Agent.Pod.Annotations).GetOrElse(map[string]string{})
	podAnnotations["instana-configuration-hash"] = d.instanaConfigurationHash
	return podAnnotations
}

func (d *daemonSetBuilder) Build() optional.Optional[client.Object] {
	if d.Spec.Agent.Key == "" && d.Spec.Agent.KeysSecret == "" {
		return optional.Empty[client.Object]()
	}
	ds := &v1.DaemonSet{
		TypeMeta: metav1.TypeMeta{
			Kind:       "DaemonSet",
			APIVersion: "apps/v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      d.Name,
			Namespace: d.Namespace,
		},
		Spec: v1.DaemonSetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: d.getSelectorMatchLabels(),
			},
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels:      d.getPodTemplateLabels(),
					Annotations: d.getPodTemplateAnnotations(),
				},
				Spec: coreV1.PodSpec{},
			},
			UpdateStrategy: d.InstanaAgent.Spec.Agent.UpdateStrategy,
		},
	}

}
