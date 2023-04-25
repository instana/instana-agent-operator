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

func TestContainsElementChecker_Contains(t *testing.T) {
	for _, test := range []struct {
		name        string
		list        []int
		testElement int
		expected    bool
	}{
		{
			name:        "contains",
			list:        []int{1, 2, 3, 4},
			testElement: 2,
			expected:    true,
		},
		{
			name:        "not_contains",
			list:        []int{4, 5, 6, 7},
			testElement: 8,
			expected:    false,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				cec := NewContainsElementChecker[int](test.list)

				assertions.Equal(test.expected, cec.Contains(test.testElement))

				dec := NewDeepContainsElementChecker(test.list)

				assertions.Equal(test.expected, dec.Contains(test.testElement))
			},
		)
	}
}
