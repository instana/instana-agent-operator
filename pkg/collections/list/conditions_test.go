package list

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestConditions(t *testing.T) {
	condition := func(item bool) bool {
		return item
	}

	type conditionTest struct {
		name        string
		list        []bool
		expectedAny bool
		expectedAll bool
	}

	tests := make([]conditionTest, 0, 8)

	for _, first := range []bool{true, false} {
		for _, second := range []bool{true, false} {
			for _, third := range []bool{true, false} {
				list := []bool{first, second, third}
				tests = append(
					tests, conditionTest{
						name:        fmt.Sprintf("%v", list),
						list:        list,
						expectedAny: first || second || third,
						expectedAll: first && second && third,
					},
				)
			}
		}
	}

	for _, test := range tests {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				c := NewConditions(test.list)

				actualAny := c.Any(condition)
				assertions.Equal(test.expectedAny, actualAny)

				actualAll := c.All(condition)
				assertions.Equal(test.expectedAll, actualAll)
			},
		)
	}
}
