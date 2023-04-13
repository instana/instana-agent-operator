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

func TestIsEmpty(t *testing.T) {
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

func TestGetOrElseDo(t *testing.T) {
	t.Run(
		"nil_given", func(t *testing.T) {
			assertions := require.New(t)

			opt := Empty[string]()
			expected := "apoiwejgoisag"

			actual := opt.GetOrElse(
				func() string {
					return expected
				},
			)

			assertions.Equal(expected, actual)

		},
	)
	t.Run(
		"non_nil_given", func(t *testing.T) {
			assertions := require.New(t)

			expected := "opasegoihsegoihsg"

			opt := Of(expected)
			actual := opt.GetOrElse(
				func() string {
					return "proijrognasoieojsg"
				},
			)

			assertions.Equal(expected, actual)
		},
	)
}

func TestGetOrElse(t *testing.T) {
	t.Run(
		"nil_given", func(t *testing.T) {
			assertions := require.New(t)

			opt := Empty[string]()
			expected := "apoiwejgoisag"

			actual := opt.GetOrDefault(expected)

			assertions.Equal(expected, actual)

		},
	)
	t.Run(
		"non_nil_given", func(t *testing.T) {
			assertions := require.New(t)

			expected := "opasegoihsegoihsg"

			opt := Of(expected)
			actual := opt.GetOrDefault("proijrognasoieojsg")

			assertions.Equal(expected, actual)
		},
	)
}

func TestMap(t *testing.T) {
	type myType string

	t.Run(
		"when_empty", func(t *testing.T) {
			assertions := require.New(t)

			in := Empty[string]()

			actual := Map[string, myType](
				in, func(in string) myType {
					return myType(in)
				},
			)

			assertions.Equal(Empty[myType](), actual)
		},
	)

	t.Run(
		"when_not_empty", func(t *testing.T) {
			assertions := require.New(t)

			in := Of[string]("oiw4eoijsoidjdsgf")

			actual := Map[string, myType](
				in, func(in string) myType {
					return myType(in)
				},
			)

			assertions.Equal(Of[myType]("oiw4eoijsoidjdsgf"), actual)
		},
	)
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
