package list

type ContainsElementChecker[T comparable] interface {
	Contains(in []T, v T) bool
}

type containsElementChecker[T comparable] struct{}

func (c *containsElementChecker[T]) Contains(in []T, expected T) bool {
	set := make(map[T]bool, len(in))

	for _, val := range in {
		set[val] = true
	}

	_, res := set[expected]

	return res
}

func NewContainsElementChecker[T comparable]() ContainsElementChecker[T] {
	return &containsElementChecker[T]{}
}
