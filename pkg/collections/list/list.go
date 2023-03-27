package list

type ListFilter[T any] interface {
	Filter(in []T, shouldBeIncluded func(val T) bool) []T
}

type ListMapTo[T any, S any] interface {
	MapTo(in []T, mapItemTo func(val T) S) []S
}

type transformer[T any, S any] struct{}

func NewListFilter[T any]() ListFilter[T] {
	return &transformer[T, any]{}
}

func NewListMapTo[T any, S any]() ListMapTo[T, S] {
	return &transformer[T, S]{}
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

// TODO: ~~warning (error) if not expected name and namespace (and status/event?)~~ -> shouldn't be needed with helm uninstall logic below
// TODO: Keep Helm uninstall step for migration -> Do this step iff (old) finalizer is present as this indicates upgrade, use a different finalizer from now on (if one is still needed)
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
// TODO: extra: runtime status from agents?
// TODO: extra auto detect OpenShift, auto set tolerations, etc.
// TODO: finalizers to delete cluster-scoped resource types via deletecollection on labels? Or just do it the "wrong" way?
// TODO: CRD validation flags?

// TODO: Mockgens in Makefile
// TODO: Result type?
