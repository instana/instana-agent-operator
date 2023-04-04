package pointer

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestTo(t *testing.T) {
	assertions := require.New(t)

	actual := *To(5)

	assertions.Equal(5, actual)
}

func TestDerefOrEmpty(t *testing.T) {
	t.Run(
		"non_nil_pointer_given", func(t *testing.T) {
			assertions := require.New(t)

			actual := DerefOrEmpty(To(10))

			assertions.Equal(10, actual)
		},
	)
	t.Run(
		"nil_pointer_given", func(t *testing.T) {
			assertions := require.New(t)

			actual := DerefOrEmpty[int](nil)

			assertions.Equal(0, actual)
		},
	)
}

func TestDerefOrDefault(t *testing.T) {
	t.Run(
		"non_nil_pointer_given", func(t *testing.T) {
			assertions := require.New(t)

			actual := DerefOrDefault(To(5), 10)

			assertions.Equal(5, actual)
		},
	)
	t.Run(
		"nil_pointer_given", func(t *testing.T) {
			assertions := require.New(t)

			actual := DerefOrDefault(nil, 10)

			assertions.Equal(10, actual)
		},
	)
}
