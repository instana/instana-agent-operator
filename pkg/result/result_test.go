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

func TestResult_IsSuccess_IsFailure(t *testing.T) {
	type testCase struct {
		name  string
		input string
		err   error
		want  bool
	}

	testCases := []testCase{
		{
			name:  "empty_val_nil_error",
			input: "",
			err:   nil,
			want:  true,
		},
		{
			name:  "empty_val_non_nil_error",
			input: "",
			err:   errors.New("asdf"),
			want:  false,
		},
		{
			name:  "non_empty_val_nil_error",
			input: "jljsd;lfjads",
			err:   nil,
			want:  true,
		},
		{
			name:  "non_empty_val_non_nil_error",
			input: "kjasldjldsaf",
			err:   errors.New(";ljkasldjf;lkdsf"),
			want:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				rslt := Of(tc.input, tc.err)
				assertions.Equal(tc.want, rslt.IsSuccess())
				assertions.Equal(tc.want, !rslt.IsFailure())
			},
		)
	}
}

func TestResult_ToOptional(t *testing.T) {
	for _, test := range []struct {
		name        string
		inputVal    string
		inputErr    error
		expectEmpty bool
	}{
		{
			name:        "empty_val_nil_error",
			inputVal:    "",
			inputErr:    nil,
			expectEmpty: true,
		},
		{
			name:        "empty_val_non_nil_error",
			inputVal:    "",
			inputErr:    errors.New(""),
			expectEmpty: true,
		},
		{
			name:        "non_empty_val_nil_error",
			inputVal:    "abcd",
			inputErr:    nil,
			expectEmpty: false,
		},
		{
			name:        "non_empty_val_non_nil_error",
			inputVal:    "abcd",
			inputErr:    errors.New(""),
			expectEmpty: true,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				rslt := Of(test.inputVal, test.inputErr)
				assertions.Equal(test.expectEmpty, rslt.ToOptional().IsEmpty())
			},
		)
	}
}

func TestResult_OnSuccess(t *testing.T) {
	for _, tc := range []struct {
		name   string
		result Result[int]
		expect int
	}{
		{
			name:   "with_nil_error",
			result: Of(5, nil),
			expect: 5,
		},
		{
			name:   "with_non_nil_error",
			result: Of(5, errors.New("asdfds")),
			expect: 0,
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := 0

				rsltCopy := tc.result.OnSuccess(
					func(i int) {
						actual = i
					},
				)

				assertions.Same(tc.result, rsltCopy)
				assertions.Equal(tc.expect, actual)
			},
		)
	}
}

func TestResult_OnFailure(t *testing.T) {
	for _, tc := range []struct {
		name     string
		input    Result[int]
		expected int
	}{
		{
			name:     "with_nil_error",
			input:    Of(5, nil),
			expected: 0,
		},
		{
			name:     "with_non_nil_error",
			input:    Of(5, errors.New("7")),
			expected: 7,
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				actual := 0

				rsltCopy := tc.input.OnFailure(
					func(err error) {
						i, _ := strconv.Atoi(err.Error())
						actual = i
					},
				)
				assertions.Same(tc.input, rsltCopy)
				assertions.Equal(tc.expected, actual)
			},
		)
	}
}

func TestResult_Recover(t *testing.T) {
	for _, tc := range []struct {
		name          string
		inputValue    string
		inputError    error
		expectedValue string
		expectedError error
		recoverFunc   func(error) (string, error)
		expectedSame  bool
	}{
		{
			name:          "with_nil_error",
			inputValue:    "hello",
			inputError:    nil,
			expectedValue: "hello",
			expectedError: nil,
			recoverFunc: func(err error) (string, error) {
				return err.Error(), nil
			},
			expectedSame: true,
		},
		{
			name:          "with_non_nil_error",
			inputValue:    "hello",
			inputError:    errors.New("world"),
			expectedValue: "world",
			expectedError: nil,
			recoverFunc: func(err error) (string, error) {
				return err.Error(), nil
			},
			expectedSame: false,
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				rslt := Of(tc.inputValue, tc.inputError)
				rsltCopy := rslt.Recover(tc.recoverFunc)

				if tc.expectedSame {
					assertions.Same(rslt, rsltCopy)
				} else {
					assertions.NotSame(rslt, rsltCopy)
				}

				actualVal, actualErr := rsltCopy.Get()
				assertions.Equal(tc.expectedValue, actualVal)
				assertions.Equal(tc.expectedError, actualErr)
			},
		)
	}
}

func TestResult_RecoverCatching(t *testing.T) {
	for _, tc := range []struct {
		name          string
		inputValue    string
		inputError    error
		expectedValue string
		expectedError error
		getRsltCopy   func(rslt Result[string]) Result[string]
		expectedSame  bool
	}{
		{
			name:          "with_nil_error",
			inputValue:    "hello",
			inputError:    nil,
			expectedValue: "hello",
			expectedError: nil,
			getRsltCopy: func(rslt Result[string]) Result[string] {
				return rslt.RecoverCatching(
					func(err error) (string, error) {
						return err.Error(), nil
					},
				)
			},
			expectedSame: true,
		},
		{
			name:          "with_non_nil_error",
			inputValue:    "hello",
			inputError:    errors.New("world"),
			expectedValue: "world",
			expectedError: nil,
			getRsltCopy: func(rslt Result[string]) Result[string] {
				return rslt.RecoverCatching(
					func(err error) (string, error) {
						return err.Error(), nil
					},
				)
			},
			expectedSame: false,
		},
		{
			name:          "with_non_nil_error_then_throw",
			inputValue:    "hello",
			inputError:    errors.New("world"),
			expectedValue: "world_goodbye",
			expectedError: nil,
			getRsltCopy: func(rslt Result[string]) Result[string] {
				return rslt.
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
			},
			expectedSame: false,
		},
	} {
		t.Run(
			tc.name, func(t *testing.T) {
				assertions := require.New(t)

				rslt := Of(tc.inputValue, tc.inputError)
				rsltCopy := tc.getRsltCopy(rslt)

				if tc.expectedSame {
					assertions.Same(rslt, rsltCopy)
				} else {
					assertions.NotSame(rslt, rsltCopy)
				}

				actualVal, actualErr := rsltCopy.Get()
				assertions.Equal(tc.expectedValue, actualVal)
				assertions.Equal(tc.expectedError, actualErr)
			},
		)
	}
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
