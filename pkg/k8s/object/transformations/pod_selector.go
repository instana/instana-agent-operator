package transformations

import instanav1 "github.com/instana/instana-agent-operator/api/v1"

type PodSelectorLabelGenerator interface {
	GetPodSelectorLabels() map[string]string
}

type podSelectorLabelGenerator struct {
	*instanav1.InstanaAgent
	component string
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
