package volume

import (
	"errors"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/builders/helpers"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: test

type Volume int

const (
	DevVolume Volume = iota
	RunVolume
	VarRunVolume
	VarRunKuboVolume
	VarRunContainerdVolume
	VarContainerdConfigVolume
	SysVolume
	VarLogVolume
	VarLibVolume
	VarDataVolume
	MachineIdVolume
	ConfigVolume
	TlsVolume
	RepoVolume
)

type VolumeBuilder interface {
	Build(volumes ...Volume) []VolumeWithMount
}

type volumeBuilder struct {
	*instanav1.InstanaAgent
	helpers.Helpers
	isNotOpenShift bool
}

func (v *volumeBuilder) getBuilder(volume Volume) func() optional.Optional[VolumeWithMount] {
	switch volume {
	case DevVolume:
		return v.devVolume
	case RunVolume:
		return v.runVolume
	case VarRunVolume:
		return v.varRunVolume
	case VarRunKuboVolume:
		return v.varRunKuboVolume
	case VarRunContainerdVolume:
		return v.varRunContainerdVolume
	case VarContainerdConfigVolume:
		return v.varContainerdConfigVolume
	case SysVolume:
		return v.sysVolume
	case VarLogVolume:
		return v.varLogVolume
	case VarLibVolume:
		return v.varLibVolume
	case VarDataVolume:
		return v.varDataVolume
	case MachineIdVolume:
		return v.machineIdVolume
	case ConfigVolume:
		return v.configVolume
	case TlsVolume:
		return v.tlsVolume
	case RepoVolume:
		return v.repoVolume
	default:
		panic(errors.New("unknown volume requested"))
	}
}

func (v *volumeBuilder) Build(volumes ...Volume) []VolumeWithMount {
	volumeOptionals := list.NewListMapTo[Volume, optional.Optional[VolumeWithMount]]().MapTo(
		volumes,
		func(volume Volume) optional.Optional[VolumeWithMount] {
			return v.getBuilder(volume)()
		},
	)

	return optional.NewNonEmptyOptionalMapper[VolumeWithMount]().AllNonEmpty(volumeOptionals)
}

func NewVolumeBuilder(agent *instanav1.InstanaAgent, isOpenShift bool) VolumeBuilder {
	return &volumeBuilder{
		InstanaAgent:   agent,
		Helpers:        helpers.NewHelpers(agent),
		isNotOpenShift: !isOpenShift,
	}
}
