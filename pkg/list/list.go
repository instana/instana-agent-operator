package list

import "github.com/instana/instana-agent-operator/pkg/optional"

type ListFilter[T any] interface {
	Filter(in []T, shouldBeIncluded func(val T) bool) []T
}

type ListMapTo[T any, S any] interface {
	MapTo(in []T, mapItemTo func(val T) S) []S
}

type NonEmptyOptionalMapper[T any] interface {
	AllNonEmpty(in []optional.Optional[T]) []T
}

type transformer[T any, S any] struct{}

type nonEmptyOptionalMapper[T any] struct {
	transformer[optional.Optional[T], T]
}

func NewListFilter[T any]() ListFilter[T] {
	return &transformer[T, any]{}
}

func NewListMapTo[T any, S any]() ListMapTo[T, S] {
	return &transformer[T, S]{}
}

func NewNonEmptyOptionalMapper[T any]() NonEmptyOptionalMapper[T] {
	return &nonEmptyOptionalMapper[T]{}
}

func (t *transformer[T, S]) Filter(in []T, shouldBeIncluded func(val T) bool) []T {
	res := make([]T, 0, len(in))
	for _, v := range in {
		v := v
		if shouldBeIncluded(v) {
			res = append(res, v)
		}
	}
	return res
}

func (t *transformer[T, S]) MapTo(in []T, mapItemTo func(val T) S) []S {
	res := make([]S, 0, len(in))
	for _, v := range in {
		v := v
		res = append(res, mapItemTo(v))
	}
	return res
}

func (o *nonEmptyOptionalMapper[T]) AllNonEmpty(in []optional.Optional[T]) []T {
	withoutEmpties := o.transformer.Filter(in, func(val optional.Optional[T]) bool {
		return !val.IsEmpty()
	})

	return o.transformer.MapTo(withoutEmpties, func(val optional.Optional[T]) T {
		return val.Get()
	})
}

// TODO: warning if not expected name and namespace (and status/event?)
// TODO: owned resources in controller watch
// TODO: exponential backoff config
// TODO: resource renderer interface?
// TODO: general transformers interface + implement (common labels + owner refs)
// TODO: apply all function, basic controller tasks
// TODO: then status later on, suite test
// TODO: new ci build with all tests running + golangci lint, fix golangci settings
// TODO: extra: yamlified config.yaml, etc.
// TODO: fix "controller-manager" naming convention
// TODO: status and events (+conditions?)
