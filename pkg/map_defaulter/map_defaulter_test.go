/*
 * (c) Copyright IBM Corp. 2024, 2026
 * (c) Copyright Instana Inc. 2024, 2026
 */

package map_defaulter

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMapDefaulter_SetIfEmpty(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    map[string]any
		key      string
		value    any
		expected any
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
			input:    map[string]any{"hello": "goodbye"},
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
