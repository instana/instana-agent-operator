package optional

import (
	"testing"

	"github.com/instana/instana-agent-operator/pkg/pointer"

	"github.com/stretchr/testify/require"
)

func assertIsEmpty[T any](t *testing.T, opt Optional[T]) {
	assertions := require.New(t)

	assertions.True(opt.IsEmpty())
	assertions.Zero(opt.Get())
}

func assertIsNotEmpty[T any](t *testing.T, opt Optional[T]) {
	assertions := require.New(t)

	assertions.False(opt.IsEmpty())
	assertions.NotZero(opt.Get())
}

func TestIsEmpty(t *testing.T) {
	t.Run("empty_created", func(t *testing.T) {
		assertIsEmpty(t, Empty[any]())
	})
	t.Run("nil_provided", func(t *testing.T) {
		assertIsEmpty(t, Of[any](nil))
	})
	t.Run("non_zero_pointer_to_zero_val", func(t *testing.T) {
		assertIsNotEmpty[*bool](t, Of[*bool](pointer.To(false)))
	})
	t.Run("non_zero_explicit", func(t *testing.T) {
		assertIsNotEmpty[bool](t, Of[bool](true))
	})
	t.Run("zero_explicit", func(t *testing.T) {
		assertIsEmpty[bool](t, Of[bool](false))
	})
}

func TestGetOrElseDo(t *testing.T) {
	t.Run("nil_given", func(t *testing.T) {
		assertions := require.New(t)

		opt := Empty[string]()
		expected := "apoiwejgoisag"

		actual := opt.GetOrElseDo(func() string {
			return expected
		})

		assertions.Equal(expected, actual)

	})
	t.Run("non_nil_given", func(t *testing.T) {
		assertions := require.New(t)

		expected := "opasegoihsegoihsg"

		opt := Of(expected)
		actual := opt.GetOrElseDo(func() string {
			return "proijrognasoieojsg"
		})

		assertions.Equal(expected, actual)
	})
}

func TestGetOrElse(t *testing.T) {
	t.Run("nil_given", func(t *testing.T) {
		assertions := require.New(t)

		opt := Empty[string]()
		expected := "apoiwejgoisag"

		actual := opt.GetOrElse(expected)

		assertions.Equal(expected, actual)

	})
	t.Run("non_nil_given", func(t *testing.T) {
		assertions := require.New(t)

		expected := "opasegoihsegoihsg"

		opt := Of(expected)
		actual := opt.GetOrElse("proijrognasoieojsg")

		assertions.Equal(expected, actual)
	})
}

func TestMap(t *testing.T) {
	type myType string

	t.Run("when_empty", func(t *testing.T) {
		assertions := require.New(t)

		in := Empty[string]()

		actual := Map[string, myType](in, func(in string) myType {
			return myType(in)
		})

		assertions.Equal(Empty[myType](), actual)
	})

	t.Run("when_not_empty", func(t *testing.T) {
		assertions := require.New(t)

		in := Of[string]("oiw4eoijsoidjdsgf")

		actual := Map[string, myType](in, func(in string) myType {
			return myType(in)
		})

		assertions.Equal(Of[myType]("oiw4eoijsoidjdsgf"), actual)
	})
}
