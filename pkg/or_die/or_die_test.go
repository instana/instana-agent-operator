package or_die

import (
	"testing"

	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestOrPanic(t *testing.T) {
	p := New[string]()
	t.Run("no error", func(t *testing.T) {
		const expected = "oiwegoihsdoi"
		assertions := require.New(t)
		actual := p.ResultOrDie(func() (string, error) {
			return expected, nil
		})
		assertions.Equal(expected, actual)
	})
	t.Run("with error", func(t *testing.T) {
		const expected = "woiegoisoishdg"
		assertions := require.New(t)

		assertions.PanicsWithError(expected, func() {
			p.ResultOrDie(func() (string, error) {
				return "oisjeoijsdoigj", errors.New(expected)
			})
		})
	})
}
