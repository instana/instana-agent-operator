package poddisruptionbudget

import (
	v1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/helpers"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/pointer"
)

const componentName = constants.ComponentK8Sensor

type podDisruptionBudgetBuilder struct {
	*instanav1.InstanaAgent

	helpers.Helpers
	transformations.PodSelectorLabelGenerator
}

func (p *podDisruptionBudgetBuilder) build() *v1.PodDisruptionBudget {
	return &v1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "policy/v1",
			Kind:       "PodDisruptionBudget",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      p.K8sSensorResourcesName(),
			Namespace: p.Namespace,
		},
		Spec: v1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: p.GetPodSelectorLabels(),
			},
			MinAvailable: pointer.To(intstr.FromInt(p.Spec.K8sSensor.DeploymentSpec.Replicas - 1)),
		},
	}
}

func (p *podDisruptionBudgetBuilder) Build() builder.OptionalObject {
	if pointer.DerefOrEmpty(p.Spec.K8sSensor.PodDisruptionBudget.Enabled) && p.Spec.K8sSensor.DeploymentSpec.Replicas > 1 {
		return optional.Of[client.Object](p.build())
	} else {
		return optional.Empty[client.Object]()
	}
}

func (p *podDisruptionBudgetBuilder) ComponentName() string {
	return componentName
}

func (p *podDisruptionBudgetBuilder) IsNamespaced() bool {
	return true
}

func NewPodDisruptionBudgetBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &podDisruptionBudgetBuilder{
		InstanaAgent:              agent,
		Helpers:                   helpers.NewHelpers(agent),
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
	}
}
