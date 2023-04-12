package transformations

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type PodSelectorLabelGenerator interface {
	GetPodSelectorLabels() map[string]string
	GetPodLabels(userLabels map[string]string) map[string]string
}

type podSelectorLabelGenerator struct {
	*instanav1.InstanaAgent
	component string
}

// TODO: Test

func (p *podSelectorLabelGenerator) GetPodLabels(userLabels map[string]string) map[string]string {
	podLabels := optional.Of(userLabels).GetOrDefault(make(map[string]string, 7))

	podLabels[NameLabel] = name
	podLabels[InstanceLabel] = p.Name
	podLabels[ComponentLabel] = p.component
	podLabels[PartOfLabel] = partOf
	podLabels[ManagedByLabel] = managedBy

	return podLabels
}

func (p *podSelectorLabelGenerator) GetPodSelectorLabels() map[string]string {
	return map[string]string{
		NameLabel:      name,
		InstanceLabel:  p.Name,
		ComponentLabel: p.component,
	}
}

func PodSelectorLabels(agent *instanav1.InstanaAgent, component string) PodSelectorLabelGenerator {
	return &podSelectorLabelGenerator{
		InstanaAgent: agent,
		component:    component,
	}
}
