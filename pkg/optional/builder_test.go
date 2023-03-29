package optional

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type intBuilder struct {
	val int
}

func (i *intBuilder) Build() Optional[int] {
	return Of(i.val)
}

func WillBuild(val int) Builder[int] {
	return &intBuilder{
		val: val,
	}
}

func BuildersFor(vals []int) []Builder[int] {
	res := make([]Builder[int], 0, len(vals))
	for _, val := range vals {
		res = append(res, WillBuild(val))
	}
	return res
}

func TestBuilderProcessor_BuildAll(t *testing.T) {
	assertions := require.New(t)

	builders := BuildersFor([]int{0, 1, 2, 0, 3, 4, 0, 5, 0, 6, 7, 8, 9, 0})

	bp := NewBuilderProcessor(builders)
	actual := bp.BuildAll()

	assertions.Equal([]int{1, 2, 3, 4, 5, 6, 7, 8, 9}, actual)
}

func Test_fromLiteralVal(t *testing.T) {
	assertions := require.New(t)

	actual := BuilderFromLiteral("asdf").Build()

	assertions.Equal(Of("asdf"), actual)
}
