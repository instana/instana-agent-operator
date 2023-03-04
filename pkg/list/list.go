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
		return *val.Get()
	})
}

// TODO: maybe: ignore anything not named instana-agent?
// TODO: other todo, owned resources, exponential backoff config, resource renderer interface?, general transformers interface + implement (common labels + owner refs), apply all function, basic controller tasks, then status later on, suite test, new ci build with all tests running + golangci lint, fix golangci settings
// TODO: extra: yamlified config.yaml, etc.
