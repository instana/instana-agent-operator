package multierror

import (
	"testing"

	"errors"

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
	assertions := require.New(t)

	me := NewMultiError(
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
		UnwrapAll(me.Combine()),
	)

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
		UnwrapAll(me.Combine()),
	)
}
