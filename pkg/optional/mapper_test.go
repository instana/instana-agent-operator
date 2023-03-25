package optional

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestAllNonEmpty(t *testing.T) {
	mpr := NewNonEmptyOptionalMapper[bool]()

	in := []Optional[bool]{
		Empty[bool](),
		Of(true),
		Of(false),
		Of(true),
		Of(true),
		Empty[bool](),
		Empty[bool](),
		Empty[bool](),
		Of(false),
		Of(true),
	}
	out := mpr.AllNonEmpty(in)

	assertions := require.New(t)

	assertions.Equal([]bool{true, true, true, true}, out)
}
