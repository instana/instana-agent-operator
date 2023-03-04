package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestDiff_Diff(t *testing.T) {
	a := []int{0, 1, 2, 3, 4, 5, 6}
	b := []int{3, 4, 5, 6, 7, 8, 9}

	df := NewDiff[int]()
	ddf := NewDeepDiff[int]()

	for _, test := range []struct {
		name string

		old []int
		new []int

		expected []int
	}{
		{
			name:     "a-b",
			old:      a,
			new:      b,
			expected: []int{0, 1, 2},
		},
		{
			name:     "b-a",
			old:      b,
			new:      a,
			expected: []int{7, 8, 9},
		},
		{
			name:     "a-empty",
			old:      a,
			new:      nil,
			expected: a,
		},
		{
			name:     "empty-a",
			old:      nil,
			new:      a,
			expected: []int{},
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := df.Diff(test.old, test.new)
				assertions.ElementsMatch(test.expected, actual)

				deepActual := ddf.Diff(test.old, test.new)
				assertions.ElementsMatch(test.expected, deepActual)
			},
		)
	}
}
