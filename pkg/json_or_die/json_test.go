package json_or_die

import (
	"testing"

	"github.com/stretchr/testify/require"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
)

func testJsonOrDie[T any](t *testing.T, name string, createFunction func() JsonOrDieMarshaler[T]) {
	t.Run(
		name, func(t *testing.T) {
			for _, test := range []struct {
				name    string
				execute func(assertions *require.Assertions, j JsonOrDieMarshaler[T])
			}{
				{
					name: "should_panic_on_bad_json",
					execute: func(assertions *require.Assertions, j JsonOrDieMarshaler[T]) {
						assertions.Panics(
							func() {
								j.UnMarshalOrDie([]byte("{"))
							},
						)
					},
				},
				{
					name: "round_trip",
					execute: func(assertions *require.Assertions, j JsonOrDieMarshaler[T]) {
						expected := j.(*jsonOrDie[T]).obj

						marshaled := j.MarshalOrDie(expected)
						actual := j.UnMarshalOrDie(marshaled)

						assertions.Equal(expected, actual)
					},
				},
			} {
				t.Run(
					test.name, func(t *testing.T) {
						assertions := require.New(t)
						j := createFunction()
						test.execute(assertions, j)
					},
				)
			}
		},
	)
}

func TestJsonOrDie(t *testing.T) {
	testJsonOrDie(t, "struct", NewJsonOrDie[instanav1.InstanaAgent])
	testJsonOrDie(t, "map", NewJsonOrDieMap[string, any])
	testJsonOrDie(t, "array", NewJsonOrDie[string])
}
