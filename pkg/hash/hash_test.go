package hash

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type TestStruct struct {
	Field1 string `json:"field1"`
	Field2 int    `json:"field2"`
}

func TestHasher(t *testing.T) {

	h := NewJsonHasher()

	t.Run(
		"ValidInput", func(t *testing.T) {
			assertions := require.New(t)
			testStruct := &TestStruct{
				Field1: "test1",
				Field2: 42,
			}
			const expectedHash = "+wF/Y9d+dlvLIQSi8zR0IA=="
			resultHash := h.HashJsonOrDie(testStruct)
			assertions.Equal(expectedHash, resultHash)
		},
	)

	t.Run(
		"InvalidInput", func(t *testing.T) {
			assertions := require.New(t)
			assertions.PanicsWithError(
				"json: unsupported type: func()", func() {
					h.HashJsonOrDie(func() {})
				},
			)
		},
	)
}
