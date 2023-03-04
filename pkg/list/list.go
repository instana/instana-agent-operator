package list

import "github.com/instana/instana-agent-operator/pkg/optional"

func Filter[T any](in []T, shouldBeIncluded func(val T) bool) []T {
	res := make([]T, 0, len(in))
	for _, v := range in {
		v := v
		if shouldBeIncluded(v) {
			res = append(res, v)
		}
	}
	return res
}

func MapTo[X any, Y any](in []X, mapItemTo func(val X) Y) []Y {
	res := make([]Y, 0, len(in))
	for _, v := range in {
		v := v
		res = append(res, mapItemTo(v))
	}
	return res
}

func AllNonEmpty[T any](in []optional.Optional[T]) []T {
	withoutEmpties := Filter[optional.Optional[T]](in, func(val optional.Optional[T]) bool {
		return !val.IsEmpty()
	})

	return MapTo[optional.Optional[T], T](withoutEmpties, func(val optional.Optional[T]) T {
		return val.Get()
	})
}

// TODO: error if not expected name and namespace
// TODO: owned resources in controller watch
// TODO: exponential backoff config
// TODO: resource renderer interface?
// TODO: general transformers interface + implement (common labels + owner refs)
// TODO: apply all function, basic controller tasks
// TODO: then status later on, suite test
// TODO: new ci build with all tests running + golangci lint, fix golangci settings
// TODO: extra: yamlified config.yaml, etc.
// TODO: fix "controller-manager" naming convention
