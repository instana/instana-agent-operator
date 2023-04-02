package defaulter

import "github.com/instana/instana-agent-operator/pkg/optional"

type Defaulter[T any] interface {
	SetIfEmpty(def T)
}

type defaulter[T any] struct {
	val *T
}

func NewDefaulter[T any](val *T) Defaulter[T] {
	return &defaulter[T]{
		val: val,
	}
}

func (d *defaulter[T]) SetIfEmpty(def T) {
	*d.val = optional.Of(*d.val).GetOrDefault(def)
}

type mapDefaulter[K comparable, V any] struct {
	key      K
	ptrToMap *map[K]V
}

func NewMapDefaulter[K comparable, V any](key K, ptrToMap *map[K]V) Defaulter[V] {
	return &mapDefaulter[K, V]{
		key:      key,
		ptrToMap: ptrToMap,
	}
}

func (m *mapDefaulter[K, V]) SetIfEmpty(def V) {
	(*m.ptrToMap)[m.key] = optional.Of((*m.ptrToMap)[m.key]).GetOrDefault(def)
}
