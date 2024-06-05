package list

import "reflect"

func toSet[T comparable](list []T) map[T]bool {
	res := make(map[T]bool, len(list))

	for _, item := range list {
		res[item] = true
	}

	return res
}

type ContainsElementChecker[T any] interface {
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

type deepContainsElementChecker[T any] struct {
	list []T
}

func (d *deepContainsElementChecker[T]) Contains(expected T) bool {
	for _, item := range d.list {
		if reflect.DeepEqual(expected, item) {
			return true
		}
	}

	return false
}

func NewDeepContainsElementChecker[T any](in []T) ContainsElementChecker[T] {
	return &deepContainsElementChecker[T]{
		list: in,
	}
}
