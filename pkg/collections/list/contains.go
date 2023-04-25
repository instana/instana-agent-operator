package list

func toSet[T comparable](list []T) map[T]bool {
	res := make(map[T]bool, len(list))

	for _, item := range list {
		res[item] = true
	}

	return res
}

type ContainsElementChecker[T comparable] interface {
	Contains(v T) bool
}

type containsElementChecker[T comparable] struct {
	set map[T]bool
}

func (c *containsElementChecker[T]) Contains(expected T) bool {
	_, res := c.set[expected]
	return res
}

func NewContainsElementChecker[T comparable](in []T) ContainsElementChecker[T] {
	return &containsElementChecker[T]{
		set: toSet(in),
	}
}
