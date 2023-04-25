package list

type Diff[T comparable] interface {
	Diff(old []T, new []T) []T
}

type diff[T comparable] struct{}

func (d *diff[T]) Diff(old []T, new []T) []T {
	newSet := NewContainsElementChecker(new)

	res := make([]T, 0, len(old))

	for _, item := range old {
		if !newSet.Contains(item) {
			res = append(res, item)
		}
	}

	return res
}

func NewDiff[T comparable]() Diff[T] {
	return &diff[T]{}
}
