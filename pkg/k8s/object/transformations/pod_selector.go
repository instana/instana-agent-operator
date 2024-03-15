package transformations

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	_map "github.com/instana/instana-agent-operator/pkg/collections/map"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

const (
	ZoneLabel = "io.instana/zone"
)

type PodSelectorLabelGenerator interface {
	GetPodSelectorLabels() map[string]string
	GetPodLabels(userLabels map[string]string) map[string]string
}

type podSelectorLabelGenerator struct {
	*instanav1.InstanaAgent
	zone      *instanav1.Zone
	component string
}

func (p *podSelectorLabelGenerator) GetPodLabels(userLabels map[string]string) map[string]string {
	podLabels := optional.Of(_map.NewCopier(userLabels).Copy()).GetOrDefault(make(map[string]string, 6))

	podLabels[NameLabel] = name
	podLabels[InstanceLabel] = p.Name
	podLabels[ComponentLabel] = p.component
	podLabels[PartOfLabel] = partOf
	podLabels[ManagedByLabel] = managedBy

	if p.zone != nil {
		podLabels[ZoneLabel] = p.zone.Name.Name
	}

	return podLabels
}

func (p *podSelectorLabelGenerator) GetPodSelectorLabels() map[string]string {
	labels := map[string]string{
		NameLabel:      name,
		InstanceLabel:  p.Name,
		ComponentLabel: p.component,
	}

	if p.zone != nil {
		labels[ZoneLabel] = p.zone.Name.Name
	}

	return labels
}

func PodSelectorLabels(agent *instanav1.InstanaAgent, component string) PodSelectorLabelGenerator {
	return PodSelectorLabelsWithZoneInfo(agent, component, nil)
}

func PodSelectorLabelsWithZoneInfo(
	agent *instanav1.InstanaAgent,
	component string,
	zone *instanav1.Zone,
) PodSelectorLabelGenerator {
	return &podSelectorLabelGenerator{
		InstanaAgent: agent,
		component:    component,
		zone:         zone,
	}
}
