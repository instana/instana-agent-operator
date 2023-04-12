package optional

import "github.com/instana/instana-agent-operator/pkg/collections/list"

type NonEmptyOptionalMapper[T any] interface {
	AllNonEmpty(in []Optional[T]) []T
}

type nonEmptyOptionalMapper[T any] struct {
	list.ListFilter[Optional[T]]
	list.ListMapTo[Optional[T], T]
}

func NewNonEmptyOptionalMapper[T any]() NonEmptyOptionalMapper[T] {
	return &nonEmptyOptionalMapper[T]{
		ListFilter: list.NewListFilter[Optional[T]](),
		ListMapTo:  list.NewListMapTo[Optional[T], T](),
	}
}

func (o *nonEmptyOptionalMapper[T]) AllNonEmpty(in []Optional[T]) []T {
	withoutEmpties := o.Filter(
		in, func(val Optional[T]) bool {
			return !val.IsEmpty()
		},
	)

	return o.MapTo(
		withoutEmpties, func(val Optional[T]) T {
			return val.GetPodSelectorLabels()
		},
	)
}
