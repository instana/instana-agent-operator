package optional

import (
	"github.com/stretchr/testify/require"
	"k8s.io/utils/pointer"
	"testing"
)

func assertIsEmpty(t *testing.T, opt Optional[any]) {
	assertions := require.New(t)

	assertions.True(opt.IsEmpty())
	assertions.Nil(opt.Get())
}

func assertIsNotEmpty[T any](t *testing.T, opt Optional[T]) {
	assertions := require.New(t)

	assertions.False(opt.IsEmpty())
	assertions.NotNil(opt.Get())
}

func TestIsEmpty(t *testing.T) {
	t.Run("empty_created", func(t *testing.T) {
		assertIsEmpty(t, Empty[any]())
	})
	t.Run("nil_provided", func(t *testing.T) {
		assertIsEmpty(t, OfNilable[any](nil))
	})
	t.Run("not_nil_implicit", func(t *testing.T) {
		assertIsNotEmpty[bool](t, OfNilable[bool](pointer.BoolPtr(true)))
	})
	t.Run("not_nil_explicit", func(t *testing.T) {
		assertIsNotEmpty[bool](t, Of[bool](true))
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
