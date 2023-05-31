package json_or_die

import (
	"encoding/json"

	"github.com/instana/instana-agent-operator/pkg/or_die"
)

type JsonOrDieMarshaler[T any] interface {
	MarshalOrDie(obj T) []byte
	UnMarshalOrDie(raw []byte) T
}

type jsonOrDie[T any] struct {
	or_die.OrDie[[]byte]
	newEmptyObject func() T
}

func (j *jsonOrDie[T]) MarshalOrDie(obj T) []byte {
	return j.ResultOrDie(
		func() ([]byte, error) {
			return json.Marshal(obj)
		},
	)
}

func (j *jsonOrDie[T]) UnMarshalOrDie(raw []byte) T {
	obj := j.newEmptyObject()

	j.ResultOrDie(
		func() ([]byte, error) {
			return nil, json.Unmarshal(raw, &obj)
		},
	)

	return obj
}

func NewJsonOrDie[T any]() JsonOrDieMarshaler[*T] {
	return &jsonOrDie[*T]{
		OrDie: or_die.New[[]byte](),
		newEmptyObject: func() *T {
			var obj T
			return &obj
		},
	}
}

func NewJsonOrDieMap[K comparable, V any]() JsonOrDieMarshaler[map[K]V] {
	return &jsonOrDie[map[K]V]{
		OrDie: or_die.New[[]byte](),
		newEmptyObject: func() map[K]V {
			return make(map[K]V, 0)
		},
	}
}

func NewJsonOrDieArray[T any]() JsonOrDieMarshaler[[]T] {
	return &jsonOrDie[[]T]{
		OrDie: or_die.New[[]byte](),
		newEmptyObject: func() []T {
			return make([]T, 0)
		},
	}
}
