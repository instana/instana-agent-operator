package builder

import (
	"sigs.k8s.io/controller-runtime/pkg/client"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/k8s/object/transformations"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

// TODO: Test

type OptionalObject = optional.Optional[client.Object]

type ObjectBuilder interface {
	Build() OptionalObject
	ComponentName() string
	IsNamespaced() bool
}

type BuilderTransformer interface {
	Apply(bldr ObjectBuilder) OptionalObject
}

type builderTransformer struct {
	transformations.Transformations
}

func (b *builderTransformer) Apply(builder ObjectBuilder) optional.Optional[client.Object] {
	switch opt := builder.Build(); opt.IsNotEmpty() {
	case true:
		obj := opt.Get()
		b.AddCommonLabels(obj, builder.ComponentName())
		if builder.IsNamespaced() {
			b.AddOwnerReference(obj)
		}
		return optional.Of(obj)
	default:
		return opt
	}

}

func NewBuilderTransformer(agent *instanav1.InstanaAgent) BuilderTransformer {
	return &builderTransformer{
		Transformations: transformations.NewTransformations(agent),
	}
}
