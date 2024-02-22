package recovery

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCatch(t *testing.T) {
	t.Run(
		"catch_error", func(t *testing.T) {
			assertions := require.New(t)

			expected := errors.New("hello")

			actual := func() (err error) {
				defer Catch(&err)

				panic(expected)
			}()

			assertions.ErrorAs(actual, &caughtPanic{})
			assertions.True(AsCaughtPanic(actual))
			assertions.ErrorIs(actual, expected)

			t.Log(actual.Error())
		},
	)
	t.Run(
		"catch_other", func(t *testing.T) {
			assertions := require.New(t)

			expected := "hello"

			actual := func() (err error) {
				defer Catch(&err)

				panic(expected)
			}()

			assertions.ErrorAs(actual, &caughtPanic{})
			assertions.True(AsCaughtPanic(actual))

			t.Log(actual.Error())
		},
	)
	t.Run(
		"catch_from_defer", func(t *testing.T) {
			assertions := require.New(t)

			expectedReturn := errors.New("I will be returned")
			expectedPanic := errors.New("I will be thrown in panic")

			actual := func() (err error) {
				defer Catch(&err)

				defer func() {
					panic(expectedPanic)
				}()

				return expectedReturn
			}()

			assertions.ErrorAs(actual, &caughtPanic{})
			assertions.True(AsCaughtPanic(actual))
			assertions.ErrorIs(actual, expectedReturn)
			assertions.ErrorIs(actual, expectedPanic)

			t.Log(actual.Error())
		},
	)
}
