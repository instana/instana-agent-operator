package optional

import (
	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

type Builder[T any] interface {
	Build() Optional[T]
}

type BuilderProcessor[T any] interface {
	BuildAll() []T
}

func NewBuilderProcessor[T any](builders []Builder[T]) BuilderProcessor[T] {
	return &builderProcessor[T]{
		builders:               builders,
		NonEmptyOptionalMapper: NewNonEmptyOptionalMapper[T](),
		ListMapTo:              list.NewListMapTo[Builder[T], Optional[T]](),
	}
}

type builderProcessor[T any] struct {
	builders []Builder[T]
	NonEmptyOptionalMapper[T]
	list.ListMapTo[Builder[T], Optional[T]]
}

func (b *builderProcessor[T]) BuildAll() []T {
	asOptionals := b.MapTo(
		b.builders, func(builder Builder[T]) Optional[T] {
			return builder.Build()
		},
	)
	return b.AllNonEmpty(asOptionals)
}

type fromLiteral[T any] struct {
	val T
}

func BuilderFromLiteral[T any](val T) Builder[T] {
	return &fromLiteral[T]{
		val: val,
	}
}

func (f *fromLiteral[T]) Build() Optional[T] {
	return Of(f.val)
}

// TODO: ForAllPresent here (separate out asOptionals bit OR use BuildAll result and apply to that), then applyAll / dry-run applyAll in controller util
