package poddisruptionbudget

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/builder"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/common/constants"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
)

const componentName = constants.ComponentK8Sensor

type podDisruptionBudgetBuilder struct {
	*instanav1.InstanaAgent
	transformations.PodSelectorLabelGenerator
}

func (p *podDisruptionBudgetBuilder) Build() builder.OptionalObject {
	// TODO implement me
	panic("implement me")
}

func (p *podDisruptionBudgetBuilder) ComponentName() string {
	// TODO implement me
	panic("implement me")
}

func (p *podDisruptionBudgetBuilder) IsNamespaced() bool {
	// TODO implement me
	panic("implement me")
}

func NewPodDisruptionBudgetBuilder(agent *instanav1.InstanaAgent) builder.ObjectBuilder {
	return &podDisruptionBudgetBuilder{
		InstanaAgent:              agent,
		PodSelectorLabelGenerator: transformations.PodSelectorLabels(agent, componentName),
	}
}
