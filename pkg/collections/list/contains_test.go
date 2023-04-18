package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

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

				cec := NewContainsElementChecker[int]()

				assertions.Equal(test.expected, cec.Contains(test.list, test.testElement))
			},
		)
	}
}
