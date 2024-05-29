package optional

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllNonEmpty(t *testing.T) {
	mpr := NewNonEmptyOptionalMapper[int]()

	in := []Optional[int]{
		Empty[int](),
		Of(1),
		Of(0),
		Of(2),
		Of(3),
		Empty[int](),
		Empty[int](),
		Empty[int](),
		Of(4),
		Of(5),
	}
	out := mpr.AllNonEmpty(in)

	assertions := require.New(t)

	assertions.Equal([]int{1, 2, 3, 4, 5}, out)
}
