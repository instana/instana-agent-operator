package transformations

import (
	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	_map "github.com/instana/instana-agent-operator/pkg/collections/map"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

type PodSelectorLabelGeneratorRemote interface {
	GetPodSelectorLabels() map[string]string
	GetPodLabels(userLabels map[string]string) map[string]string
}

type podSelectorLabelGeneratorRemote struct {
	*instanav1.RemoteAgent
	zone      *instanav1.Zone
	component string
}

func (p *podSelectorLabelGeneratorRemote) GetPodLabels(userLabels map[string]string) map[string]string {
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

func (p *podSelectorLabelGeneratorRemote) GetPodSelectorLabels() map[string]string {
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

func PodSelectorLabelsRemote(agent *instanav1.RemoteAgent, component string) PodSelectorLabelGenerator {
	return PodSelectorLabelsWithZoneInfoRemote(agent, component, nil)
}

func PodSelectorLabelsWithZoneInfoRemote(
	agent *instanav1.RemoteAgent,
	component string,
	zone *instanav1.Zone,
) PodSelectorLabelGeneratorRemote {
	return &podSelectorLabelGeneratorRemote{
		RemoteAgent: agent,
		component:   component,
		zone:        zone,
	}
}
