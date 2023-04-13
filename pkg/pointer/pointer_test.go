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
	for _, tc := range []struct {
		name   string
		input  *int
		output int
	}{
		{
			name:   "non_nil_pointer_given",
			input:  To(10),
			output: 10,
		},
		{
			name:   "nil_pointer_given",
			input:  nil,
			output: 0,
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := DerefOrEmpty(tc.input)

				assertions.Equal(tc.output, actual)
			},
		)
	}
}

func TestDerefOrDefault(t *testing.T) {
	for _, tt := range []struct {
		name   string
		input  *int
		defVal int
		want   int
	}{
		{
			name:   "non_nil_pointer_given",
			input:  To(5),
			defVal: 10,
			want:   5,
		},
		{
			name:   "nil_pointer_given",
			input:  nil,
			defVal: 10,
			want:   10,
		},
	} {
		t.Run(
			tt.name, func(t *testing.T) {
				assertions := require.New(t)
				actual := DerefOrDefault(tt.input, tt.defVal)
				assertions.Equal(tt.want, actual)
			},
		)
	}
}

func TestDerefOrElse(t *testing.T) {
	t.Run(
		"non_nil_pointer_given", func(t *testing.T) {
			assertions := require.New(t)

			actual := DerefOrElse(
				To(5), func() int {
					return 10
				},
			)

			assertions.Equal(5, actual)
		},
	)
	t.Run(
		"nil_pointer_given", func(t *testing.T) {
			assertions := require.New(t)

			actual := DerefOrElse(
				nil, func() int {
					return 10
				},
			)

			assertions.Equal(10, actual)
		},
	)
}
