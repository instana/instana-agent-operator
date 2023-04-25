package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_toSet(t *testing.T) {
	assertions := require.New(t)

	expected := map[int]bool{
		1: true,
		2: true,
		3: true,
	}

	in := []int{1, 2, 2, 3, 3, 3}
	actual := toSet(in)

	assertions.Equal(expected, actual)
}

func TestDiff_Diff(t *testing.T) {
	a := []int{0, 1, 2, 3, 4, 5, 6}
	b := []int{3, 4, 5, 6, 7, 8, 9}

	df := NewDiff[int]()

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
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := df.Diff(test.old, test.new)

				assertions.ElementsMatch(test.expected, actual)
			},
		)
	}
}
