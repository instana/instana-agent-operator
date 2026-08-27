/*
 * (c) Copyright IBM Corp. 2024, 2026
 * (c) Copyright Instana Inc. 2024, 2026
 */

package list

import "slices"

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
	return slices.ContainsFunc(c.list, condition)
}

func NewConditions[T any](list []T) Conditions[T] {
	return &conditions[T]{
		list: list,
	}
}
