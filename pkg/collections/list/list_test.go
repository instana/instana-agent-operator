package list

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFilter(t *testing.T) {
	mpr := NewListFilter[bool]()

	in := []bool{true, true, false, true, false, false, true}
	out := mpr.Filter(in, func(val bool) bool {
		return val
	})

	assertions := require.New(t)

	assertions.Len(out, 4)

	for _, v := range out {
		assertions.True(v)
	}
}

func TestMapTo(t *testing.T) {
	mpr := NewListMapTo[bool, int]()

	in := []bool{true, true, false, true, false, false, true}
	out := mpr.MapTo(in, func(v bool) int {
		switch v {
		case true:
			return 1
		default:
			return 0
		}
	})

	assertions := require.New(t)

	assertions.Equal([]int{1, 1, 0, 1, 0, 0, 1}, out)
}
