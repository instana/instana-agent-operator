package json_or_die

import (
	"encoding/json"

	"github.com/instana/instana-agent-operator/pkg/or_die"
)

type JsonOrDieMarshaler[T any] interface {
	marshalOrDie(obj T) []byte
	unMarshalOrDie(raw []byte) T
}

type jsonOrDie[T any] struct {
	or_die.OrDie[[]byte]
	obj T
}

func (j *jsonOrDie[T]) marshalOrDie(obj T) []byte {
	return j.ResultOrDie(
		func() ([]byte, error) {
			return json.Marshal(obj)
		},
	)
}

func (j *jsonOrDie[T]) unMarshalOrDie(raw []byte) T {
	j.ResultOrDie(
		func() ([]byte, error) {
			return nil, json.Unmarshal(raw, &j.obj)
		},
	)

	return j.obj
}

func NewJsonOrDie[T any]() JsonOrDieMarshaler[*T] {
	var obj T

	return &jsonOrDie[*T]{
		OrDie: or_die.New[[]byte](),
		obj:   &obj,
	}
}

func NewJsonOrDieMap[K comparable, V any]() JsonOrDieMarshaler[map[K]V] {
	obj := make(map[K]V, 0)

	return &jsonOrDie[map[K]V]{
		OrDie: or_die.New[[]byte](),
		obj:   obj,
	}
}

func NewJsonOrDieArray[T any]() JsonOrDieMarshaler[[]T] {
	obj := make([]T, 0)

	return &jsonOrDie[[]T]{
		OrDie: or_die.New[[]byte](),
		obj:   obj,
	}
}
