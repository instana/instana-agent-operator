package map_defaulter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapDefaulter_SetIfEmpty(t *testing.T) {
	t.Run("sets_default", func(t *testing.T) {
		assertions := require.New(t)

		var m map[string]interface{} = nil

		NewMapDefaulter(&m).SetIfEmpty("hello", "world")
		assertions.Equal("world", m["hello"])
	})
	t.Run("already_set", func(t *testing.T) {
		assertions := require.New(t)

		m := map[string]interface{}{"hello": "goodbye"}

		NewMapDefaulter(&m).SetIfEmpty("hello", "world")
		assertions.Equal("goodbye", m["hello"])
	})
}
