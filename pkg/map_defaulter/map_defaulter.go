package map_defaulter

import "github.com/instana/instana-agent-operator/pkg/optional"

type MapDefaulter[K comparable, V any] interface {
	SetIfEmpty(key K, def V)
}

type mapDefaulter[K comparable, V any] struct {
	ptrToMap *map[K]V
}

func NewMapDefaulter[K comparable, V any](ptrToMap *map[K]V) MapDefaulter[K, V] {
	*ptrToMap = optional.Of(*ptrToMap).GetOrDefault(make(map[K]V))

	return &mapDefaulter[K, V]{
		ptrToMap: ptrToMap,
	}
}

func (m *mapDefaulter[K, V]) SetIfEmpty(key K, def V) {
	(*m.ptrToMap)[key] = optional.Of((*m.ptrToMap)[key]).GetOrDefault(def)
}
