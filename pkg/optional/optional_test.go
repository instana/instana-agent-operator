package optional

import (
	"testing"

	"github.com/instana/instana-agent-operator/pkg/pointer"

	"github.com/stretchr/testify/require"
)

func assertIsEmpty[T any](t *testing.T, opt Optional[T]) {
	assertions := require.New(t)

	assertions.True(opt.IsEmpty())
	assertions.False(opt.IsNotEmpty())
	assertions.Zero(opt.Get())
}

func assertIsNotEmpty[T any](t *testing.T, opt Optional[T]) {
	assertions := require.New(t)

	assertions.False(opt.IsEmpty())
	assertions.True(opt.IsNotEmpty())
	assertions.NotZero(opt.Get())
}

func TestOptional_IsEmpty(t *testing.T) {
	for _, test := range []struct {
		name string
		f    func(t *testing.T)
	}{
		{
			name: "empty_created",
			f: func(t *testing.T) {
				assertIsEmpty(t, Empty[any]())
			},
		},
		{
			name: "nil_provided",
			f: func(t *testing.T) {
				assertIsEmpty(t, Of[any](nil))
			},
		},
		{
			name: "non_zero_pointer_to_zero_val",
			f: func(t *testing.T) {
				assertIsNotEmpty[*bool](t, Of[*bool](pointer.To(false)))
			},
		},
		{
			name: "non_zero_explicit",
			f: func(t *testing.T) {
				assertIsNotEmpty[bool](t, Of[bool](true))
			},
		},
		{
			name: "zero_explicit",
			f: func(t *testing.T) {
				assertIsEmpty[bool](t, Of[bool](false))
			},
		},
	} {
		t.Run(test.name, test.f)
	}
}

func TestOptional_GetOrElse(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    Optional[string]
		expected string
	}{
		{
			name:     "nil_given",
			input:    Empty[string](),
			expected: "proijrognasoieojsg",
		},
		{
			name:     "non_nil_given",
			input:    Of("opasegoihsegoihsg"),
			expected: "opasegoihsegoihsg",
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := tc.input.GetOrElse(
					func() string {
						return "proijrognasoieojsg"
					},
				)

				assertions.Equal(tc.expected, actual)
			},
		)
	}
}

func TestOptional_GetOrDefault(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    Optional[string]
		expected string
	}{
		{
			name:     "nil_given",
			input:    Empty[string](),
			expected: "proijrognasoieojsg",
		},
		{
			name:     "non_nil_given",
			input:    Of("opasegoihsegoihsg"),
			expected: "opasegoihsegoihsg",
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := tc.input.GetOrDefault("proijrognasoieojsg")

				assertions.Equal(tc.expected, actual)
			},
		)
	}
}

func TestMap(t *testing.T) {
	for _, tc := range []struct {
		name string
		in   Optional[string]
		want Optional[*string]
	}{
		{
			name: "when_empty",
			in:   Empty[string](),
			want: Empty[*string](),
		},
		{
			name: "when_not_empty",
			in:   Of[string]("oiw4eoijsoidjdsgf"),
			want: Of[*string](pointer.To("oiw4eoijsoidjdsgf")),
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := Map[string, *string](
					tc.in, func(in string) *string {
						return &in
					},
				)

				assertions.Equal(tc.want, actual)
			},
		)
	}
}

func TestIfPresent(t *testing.T) {
	t.Run(
		"not_present", func(t *testing.T) {
			assertions := require.New(t)

			o := Of("")
			o.IfPresent(
				func(_ string) {
					assertions.Fail("this function should not run if optional is empty")
				},
			)
		},
	)
	t.Run(
		"is_present", func(t *testing.T) {
			assertions := require.New(t)

			actual := 0

			o := Of(5)
			o.IfPresent(
				func(i int) {
					actual = i
				},
			)
			assertions.Equal(5, actual)
		},
	)
}
