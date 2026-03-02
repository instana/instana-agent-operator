/*
 * (c) Copyright IBM Corp. 2025
 */
package _map

import "maps"

type Copier[K comparable, V any] interface {
	Copy() map[K]V
}

type copier[K comparable, V any] struct {
	m map[K]V
}

func (c *copier[K, V]) Copy() map[K]V {
	res := make(map[K]V, len(c.m))

	maps.Copy(res, c.m)

	return res
}

func NewCopier[K comparable, V any](m map[K]V) Copier[K, V] {
	return &copier[K, V]{
		m: m,
	}
}
