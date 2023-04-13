package result

import (
	"errors"
	"strconv"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOf(t *testing.T) {
	assertions := require.New(t)

	rslt := Of("hello", errors.New("world"))
	actualVal, actualErr := rslt.Get()

	assertions.Equal("hello", actualVal)
	assertions.Equal(errors.New("world"), actualErr)
}

func TestOfInline(t *testing.T) {
	assertions := require.New(t)

	rslt := OfInline(
		func() (res string, err error) {
			return "hello", errors.New("world")
		},
	)
	actualVal, actualErr := rslt.Get()

	assertions.Equal("hello", actualVal)
	assertions.Equal(errors.New("world"), actualErr)
}

func TestOfInlineCatchingPanic(t *testing.T) {
	expectedErr := errors.New("err")

	for _, test := range []struct {
		name        string
		do          func() (res string, err error)
		expectedVal string
		expectedErr error
	}{
		{
			name: "no_panic",
			do: func() (res string, err error) {
				return "val", expectedErr
			},
			expectedVal: "val",
			expectedErr: expectedErr,
		},
		{
			name: "with_panic",
			do: func() (res string, err error) {
				panic(expectedErr)
			},
			expectedVal: "",
			expectedErr: expectedErr,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				rslt := OfInlineCatchingPanic(test.do)
				actualVal, actualErr := rslt.Get()

				assertions.Equal(test.expectedVal, actualVal)
				assertions.ErrorIs(actualErr, test.expectedErr)
			},
		)
	}
}

func TestOfSuccess(t *testing.T) {
	assertions := require.New(t)

	rslt := OfSuccess("hello")
	actualVal, actualErr := rslt.Get()

	assertions.Equal("hello", actualVal)
	assertions.Nil(actualErr)
}

func TestOfFailure(t *testing.T) {
	assertions := require.New(t)

	rslt := OfFailure[string](errors.New("world"))
	actualVal, actualErr := rslt.Get()

	assertions.Empty(actualVal)
	assertions.Equal(errors.New("world"), actualErr)
}

func TestResult_IsSuccess(t *testing.T) {
	t.Run(
		"empty_val_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("", nil)
			assertions.True(rslt.IsSuccess())
		},
	)
	t.Run(
		"empty_val_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("", errors.New("asdf"))
			assertions.False(rslt.IsSuccess())
		},
	)
	t.Run(
		"non_empty_val_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("jljsd;lfjads", nil)
			assertions.True(rslt.IsSuccess())
		},
	)
	t.Run(
		"non_empty_val_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("kjasldjldsaf", errors.New(";ljkasldjf;lkdsf"))
			assertions.False(rslt.IsSuccess())
		},
	)
}

func TestResult_IsFailure(t *testing.T) {
	t.Run(
		"empty_val_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("", nil)
			assertions.False(rslt.IsFailure())
		},
	)
	t.Run(
		"empty_val_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("", errors.New("asdf"))
			assertions.True(rslt.IsFailure())
		},
	)
	t.Run(
		"non_empty_val_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("jljsd;lfjads", nil)
			assertions.False(rslt.IsFailure())
		},
	)
	t.Run(
		"non_empty_val_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("kjasldjldsaf", errors.New(";ljkasldjf;lkdsf"))
			assertions.True(rslt.IsFailure())
		},
	)
}

func TestResult_ToOptional(t *testing.T) {
	t.Run(
		"empty_val_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("", nil)
			assertions.Empty(rslt.ToOptional())
		},
	)
	t.Run(
		"empty_val_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("", errors.New("asdf"))
			assertions.Empty(rslt.ToOptional())
		},
	)
	t.Run(
		"non_empty_val_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("jljsd;lfjads", nil)
			assertions.NotEmpty(rslt.ToOptional())
		},
	)
	t.Run(
		"non_empty_val_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("kjasldjldsaf", errors.New(";ljkasldjf;lkdsf"))
			assertions.Empty(rslt.ToOptional())
		},
	)
}

func TestResult_OnSuccess(t *testing.T) {
	t.Run(
		"with_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			actual := 0

			rslt := Of(5, nil)
			rsltCopy := rslt.OnSuccess(
				func(i int) {
					actual = i
				},
			)
			assertions.Same(rslt, rsltCopy)
			assertions.Equal(5, actual)
		},
	)
	t.Run(
		"with_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			actual := 0

			rslt := Of(5, errors.New("asdfds"))
			rsltCopy := rslt.OnSuccess(
				func(i int) {
					actual = i
				},
			)
			assertions.Same(rslt, rsltCopy)
			assertions.Equal(0, actual)
		},
	)
}

func TestResult_OnFailure(t *testing.T) {
	t.Run(
		"with_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			actual := 0

			rslt := Of(5, nil)
			rsltCopy := rslt.OnFailure(
				func(err error) {
					i, _ := strconv.Atoi(err.Error())
					actual = i
				},
			)
			assertions.Same(rslt, rsltCopy)
			assertions.Equal(0, actual)
		},
	)
	t.Run(
		"with_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			actual := 0

			rslt := Of(5, errors.New("7"))
			rsltCopy := rslt.OnFailure(
				func(err error) {
					i, _ := strconv.Atoi(err.Error())
					actual = i
				},
			)
			assertions.Same(rslt, rsltCopy)
			assertions.Equal(7, actual)
		},
	)
}

func TestResult_Recover(t *testing.T) {
	t.Run(
		"with_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("hello", nil)
			rsltCopy := rslt.Recover(
				func(err error) (string, error) {
					return err.Error(), nil
				},
			)
			assertions.Same(rslt, rsltCopy)
			actualVal, actualErr := rsltCopy.Get()
			assertions.Equal("hello", actualVal)
			assertions.Nil(actualErr)
		},
	)
	t.Run(
		"with_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("hello", errors.New("world"))
			rsltCopy := rslt.Recover(
				func(err error) (string, error) {
					return err.Error(), nil
				},
			)
			assertions.NotSame(rslt, rsltCopy)
			actualVal, actualErr := rsltCopy.Get()
			assertions.Equal("world", actualVal)
			assertions.Nil(actualErr)
		},
	)
}

func TestResult_RecoverCatching(t *testing.T) {
	t.Run(
		"with_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("hello", nil)
			rsltCopy := rslt.RecoverCatching(
				func(err error) (string, error) {
					return err.Error(), nil
				},
			)
			assertions.Same(rslt, rsltCopy)
			actualVal, actualErr := rsltCopy.Get()
			assertions.Equal("hello", actualVal)
			assertions.Nil(actualErr)
		},
	)
	t.Run(
		"with_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("hello", errors.New("world"))
			rsltCopy := rslt.RecoverCatching(
				func(err error) (string, error) {
					return err.Error(), nil
				},
			)
			assertions.NotSame(rslt, rsltCopy)
			actualVal, actualErr := rsltCopy.Get()
			assertions.Equal("world", actualVal)
			assertions.Nil(actualErr)
		},
	)
	t.Run(
		"with_non_nil_error_then_throw", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("hello", errors.New("world"))
			rsltCopy := rslt.
				RecoverCatching(
					func(err error) (string, error) {
						panic(errors.New(err.Error() + "_goodbye"))
					},
				).
				Recover(
					func(err error) (string, error) {
						return errors.Unwrap(errors.Unwrap(err)).Error(), nil
					},
				)
			assertions.NotSame(rslt, rsltCopy)
			actualVal, actualErr := rsltCopy.Get()
			assertions.Equal("world_goodbye", actualVal)
			assertions.Nil(actualErr)
		},
	)
}

func TestMap(t *testing.T) {
	t.Run(
		"with_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("9", nil)
			mappedResult := Map[string, int](
				rslt, func(str string) Result[int] {
					return Of(strconv.Atoi(str))
				},
			)
			actualVal, actualErr := mappedResult.Get()
			assertions.Equal(9, actualVal)
			assertions.Nil(actualErr)
		},
	)
	t.Run(
		"with_non_nil_error", func(t *testing.T) {
			assertions := require.New(t)

			rslt := Of("9", errors.New("qwerty"))
			mappedResult := Map[string, int](
				rslt, func(str string) Result[int] {
					return Of(strconv.Atoi(str))
				},
			)
			actualVal, actualErr := mappedResult.Get()
			assertions.Equal(0, actualVal)
			assertions.Equal(actualErr, errors.New("qwerty"))
		},
	)
}
