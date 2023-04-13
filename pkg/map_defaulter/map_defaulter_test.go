package map_defaulter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapDefaulter_SetIfEmpty(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    map[string]interface{}
		key      string
		value    interface{}
		expected interface{}
	}{
		{
			name:     "sets_default",
			input:    nil,
			key:      "hello",
			value:    "world",
			expected: "world",
		},
		{
			name:     "already_set",
			input:    map[string]interface{}{"hello": "goodbye"},
			key:      "hello",
			value:    "world",
			expected: "goodbye",
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				m := tc.input

				NewMapDefaulter(&m).SetIfEmpty(tc.key, tc.value)
				assertions.Equal(tc.expected, m[tc.key])
			},
		)
	}
}
