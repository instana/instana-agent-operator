package list

type Conditions[T any] interface {
	All(condition func(item T) bool) bool
	Any(condition func(item T) bool) bool
}

type conditions[T any] struct {
	list []T
}

func (c *conditions[T]) All(condition func(item T) bool) bool {
	for _, item := range c.list {
		if !condition(item) {
			return false
		}
	}
	return true
}

func (c *conditions[T]) Any(condition func(item T) bool) bool {
	for _, item := range c.list {
		if condition(item) {
			return true
		}
	}
	return false
}

func NewConditions[T any](list []T) Conditions[T] {
	return &conditions[T]{
		list: list,
	}
}
