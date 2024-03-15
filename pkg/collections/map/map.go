package _map

type MapConverter[K comparable, V any, O any] interface {
	ToList(in map[K]V, mapItemTo func(key K, val V) O) []O
}

type mapConverter[K comparable, V any, O any] struct{}

func NewMapConverter[K comparable, V any, O any]() MapConverter[K, V, O] {
	return &mapConverter[K, V, O]{}
}

func (m *mapConverter[K, V, O]) ToList(in map[K]V, mapItemTo func(key K, val V) O) []O {
	res := make([]O, 0, len(in))
	for k, v := range in {
		res = append(res, mapItemTo(k, v))
	}
	return res
}
