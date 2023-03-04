package list

func filter[T any](in []T, shouldBeIncluded func(val T) bool) []T {
	res := make([]T, 0, len(in))
	for _, v := range in {
		v := v
		if shouldBeIncluded(v) {
			res = append(res, v)
		}
	}
	return res
}

func mapTo[X any, Y any](in []X, mapItemTo func(v X) Y) []Y {
	res := make([]Y, 0, len(in))
	for _, v := range in {
		v := v
		res = append(res, mapItemTo(v))
	}
	return res
}

// TODO: filter and mapto for optional

// TODO: other todo, owned resources, exponential backoff config, resource renderer interface?, general transformers interface + implement (common labels + owner refs), apply all function, basic controller tasks, then status later on
