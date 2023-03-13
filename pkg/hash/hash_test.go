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

	h := NewHasher()

	t.Run("ValidInput", func(t *testing.T) {
		assertions := require.New(t)
		testStruct := &TestStruct{
			Field1: "test1",
			Field2: 42,
		}
		const expectedHash = "\xfb\x01\x7fc\xd7~v[\xcb!\x04\xa2\xf34t "
		resultHash := h.HashOrDie(testStruct)
		assertions.Equal(expectedHash, resultHash)
	})

	t.Run("InvalidInput", func(t *testing.T) {
		assertions := require.New(t)
		assertions.PanicsWithError("json: unsupported type: func()", func() {
			h.HashOrDie(func() {})
		})
	})
}
