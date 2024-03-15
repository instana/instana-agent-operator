package _map

type Copier[K comparable, V any] interface {
	Copy() map[K]V
}

type copier[K comparable, V any] struct {
	m map[K]V
}

func (c *copier[K, V]) Copy() map[K]V {
	res := make(map[K]V, len(c.m))

	for k, v := range c.m {
		res[k] = v
	}

	return res
}

func NewCopier[K comparable, V any](m map[K]V) Copier[K, V] {
	return &copier[K, V]{
		m: m,
	}
}
