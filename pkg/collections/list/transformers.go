/*
 * (c) Copyright IBM Corp. 2024, 2026
 * (c) Copyright Instana Inc. 2024, 2026
 */

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
		if shouldBeIncluded(v) {
			res = append(res, v)
		}
	}
	return res
}

func (t *transformer[T, S]) MapTo(in []T, mapItemTo func(val T) S) []S {
	res := make([]S, 0, len(in))
	for _, v := range in {
		res = append(res, mapItemTo(v))
	}
	return res
}
