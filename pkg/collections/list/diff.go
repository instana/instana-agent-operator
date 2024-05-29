package list

type Diff[T any] interface {
	Diff(old []T, new []T) []T
}

type diff[T any] struct {
	newContainsElementChecker func(in []T) ContainsElementChecker[T]
}

func (d *diff[T]) Diff(old []T, new []T) []T {
	newSet := d.newContainsElementChecker(new)

	res := make([]T, 0, len(old))

	for _, item := range old {
		if !newSet.Contains(item) {
			res = append(res, item)
		}
	}

	return res
}

func NewDiff[T comparable]() Diff[T] {
	return &diff[T]{
		newContainsElementChecker: NewContainsElementChecker[T],
	}
}

func NewDeepDiff[T any]() Diff[T] {
	return &diff[T]{
		newContainsElementChecker: NewDeepContainsElementChecker[T],
	}
}
