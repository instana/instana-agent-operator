package defaulter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type myType struct {
	myField map[string]interface{}
}

func TestDefaulter_SetIfEmpty(t *testing.T) {
	t.Run("sets_default", func(t *testing.T) {
		assertions := require.New(t)

		mt := &myType{}
		assertions.Nil(mt.myField)

		NewDefaulter(&mt.myField).SetIfEmpty(make(map[string]interface{}))
		assertions.NotNil(mt.myField)
	})
	t.Run("already_set", func(t *testing.T) {
		assertions := require.New(t)

		m := map[string]interface{}{"hello": "world", "foo": "bar"}
		mt := &myType{
			myField: m,
		}

		NewDefaulter(&mt.myField).SetIfEmpty(make(map[string]interface{}))
		assertions.Equal(m, mt.myField)
	})
}

func TestMapDefaulter_SetIfEmpty(t *testing.T) {
	t.Run("sets_default", func(t *testing.T) {
		assertions := require.New(t)

		m := map[string]interface{}{}

		NewMapDefaulter("hello", &m).SetIfEmpty("world")
		assertions.Equal("world", m["hello"])
	})
	t.Run("already_set", func(t *testing.T) {
		assertions := require.New(t)

		m := map[string]interface{}{"hello": "goodbye"}

		NewMapDefaulter("hello", &m).SetIfEmpty("world")
		assertions.Equal("goodbye", m["hello"])
	})
}
