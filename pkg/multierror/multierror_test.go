package multierror

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func UnwrapAll(err error) []error {
	u, ok := err.(interface {
		Unwrap() []error
	})
	if !ok {
		return nil
	}
	return u.Unwrap()
}

func TestMultiError(t *testing.T) {
	meTarget := MultiError{}
	seTarget := errors.New("")

	t.Run("empty_should_be_nil", func(t *testing.T) {
		assertions := require.New(t)

		me := NewMultiErrorBuilder()
		assertions.ErrorIs(me.Build(), nil)

		me.Add(errors.New(""))
		assertions.NotNil(me.Build())
	})
	t.Run("all_nil_should_be_nil", func(t *testing.T) {
		assertions := require.New(t)

		me := NewMultiErrorBuilder(nil, nil)
		assertions.ErrorIs(me.Build(), nil)

		me.Add(errors.New(""))
		assertions.NotNil(me.Build())
	})
	t.Run("combine_and_add", func(t *testing.T) {
		assertions := require.New(t)

		me := NewMultiErrorBuilder(
			errors.New("1"),
			nil,
			errors.New("2"),
			errors.New("3"),
		)
		assertions.Equal(
			[]error{
				errors.New("1"),
				nil,
				errors.New("2"),
				errors.New("3"),
			},
			me.All(),
		)
		assertions.Equal(
			[]error{
				errors.New("1"),
				errors.New("2"),
				errors.New("3"),
			},
			me.AllNonNil(),
		)
		assertions.Equal(
			[]error{
				errors.New("1"),
				errors.New("2"),
				errors.New("3"),
			},
			UnwrapAll(me.Build().(MultiError).Unwrap()),
		)
		assertions.NotNil(me.Build())
		assertions.ErrorAs(me.Build(), &meTarget)
		assertions.ErrorAs(me.Build(), &seTarget)

		me.Add(
			errors.New("4"),
			nil,
			errors.New("5"),
		)
		assertions.Equal(
			[]error{
				errors.New("1"),
				nil,
				errors.New("2"),
				errors.New("3"),
				errors.New("4"),
				nil,
				errors.New("5"),
			},
			me.All(),
		)
		assertions.Equal(
			[]error{
				errors.New("1"),
				errors.New("2"),
				errors.New("3"),
				errors.New("4"),
				errors.New("5"),
			},
			me.AllNonNil(),
		)
		assertions.Equal(
			[]error{
				errors.New("1"),
				errors.New("2"),
				errors.New("3"),
				errors.New("4"),
				errors.New("5"),
			},
			UnwrapAll(me.Build().(MultiError).Unwrap()),
		)
		assertions.NotNil(me.Build())
		assertions.ErrorAs(me.Build(), &meTarget)
		assertions.ErrorAs(me.Build(), &seTarget)
	})
}
