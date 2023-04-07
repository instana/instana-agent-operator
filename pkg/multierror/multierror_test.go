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
	meTarget := multiError{}
	seTarget := errors.New("")

	t.Run(
		"empty_should_be_nil", func(t *testing.T) {
			assertions := require.New(t)

			me := NewMultiErrorBuilder()
			assertions.ErrorIs(me.Build(), nil)

			me.Add(errors.New(""))
			assertions.NotNil(me.Build())
		},
	)
	t.Run(
		"all_nil_should_be_nil", func(t *testing.T) {
			assertions := require.New(t)

			me := NewMultiErrorBuilder(nil, nil)
			assertions.ErrorIs(me.Build(), nil)

			me.Add(errors.New(""))
			assertions.NotNil(me.Build())
		},
	)
	t.Run(
		"is_all_constituent_errors", func(t *testing.T) {
			assertions := require.New(t)

			expected1 := errors.New("1")
			expected2 := errors.New("2")

			me := NewMultiErrorBuilder(nil, expected1, nil)

			assertions.Equal([]error{nil, expected1, nil}, me.All())
			assertions.Equal([]error{expected1}, me.AllNonNil())
			assertions.ErrorIs(me.Build(), expected1)

			me.Add(expected2)

			assertions.Equal([]error{nil, expected1, nil, expected2}, me.All())
			assertions.Equal([]error{expected1, expected2}, me.AllNonNil())
			actual := me.Build()
			assertions.ErrorIs(actual, expected1)
			assertions.ErrorIs(actual, expected2)
		},
	)
	t.Run(
		"combine_and_add", func(t *testing.T) {
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
				UnwrapAll(me.Build().(multiError).Unwrap()),
			)
			assertions.NotNil(me.Build())
			assertions.ErrorAs(me.Build(), &meTarget)
			assertions.True(AsMultiError(me.Build()))
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
				UnwrapAll(me.Build().(multiError).Unwrap()),
			)
			assertions.NotNil(me.Build())
			assertions.ErrorAs(me.Build(), &meTarget)
			assertions.True(AsMultiError(me.Build()))
			assertions.ErrorAs(me.Build(), &seTarget)
		},
	)
}
