package list

type ContainsElementChecker[T comparable] interface {
	Contains(in []T, v T) bool
}

type containsElementChecker[T comparable] struct{}

func (c *containsElementChecker[T]) Contains(in []T, expected T) bool {
	for _, val := range in {
		if val == expected {
			return true
		}
	}
	return false
}

func NewContainsElementChecker[T comparable]() ContainsElementChecker[T] {
	return &containsElementChecker[T]{}
}
