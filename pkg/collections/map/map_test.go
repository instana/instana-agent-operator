package _map

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapConverter_ToList(t *testing.T) {
	assertions := require.New(t)

	in := map[string]string{
		"foo":       "bar",
		"hello":     "world",
		"something": "else",
	}

	actual := NewMapConverter[string, string, string]().ToList(in, func(key string, val string) string {
		return fmt.Sprintf("%s: %s", key, val)
	})

	assertions.ElementsMatch([]string{"foo: bar", "hello: world", "something: else"}, actual)
}
