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
		return v.DevVolume
	case RunVolume:
		return v.RunVolume
	case VarRunVolume:
		return v.VarRunVolume
	case VarRunKuboVolume:
		return v.VarRunKuboVolume
	case VarRunContainerdVolume:
		return v.VarRunContainerdVolume
	case VarContainerdConfigVolume:
		return v.VarRunContainerdVolume
	case SysVolume:
		return v.SysVolume
	case VarLogVolume:
		return v.VarLogVolume
	case VarLibVolume:
		return v.VarLibVolume
	case VarDataVolume:
		return v.VarDataVolume
	case MachineIdVolume:
		return v.MachineIdVolume
	case TlsVolume:
		return v.TlsVolume
	case RepoVolume:
		return v.RepoVolume
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
