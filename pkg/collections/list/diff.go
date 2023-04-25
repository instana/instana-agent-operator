package list

func toSet[T comparable](list []T) map[T]bool {
	res := make(map[T]bool, len(list))

	for _, item := range list {
		res[item] = true
	}

	return res
}

type Diff[T comparable] interface {
	Diff(old []T, new []T) []T
}

type diff[T comparable] struct{}

func (d *diff[T]) Diff(old []T, new []T) []T {
	oldSet := toSet(old)
	newSet := toSet(new)

	res := make([]T, 0, len(oldSet))

	for item := range oldSet {
		if _, inNewSet := newSet[item]; !inNewSet {
			res = append(res, item)
		}
	}

	return res
}

func NewDiff[T comparable]() Diff[T] {
	return &diff[T]{}
}
