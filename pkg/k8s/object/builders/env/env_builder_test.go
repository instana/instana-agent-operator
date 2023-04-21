package env

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

func rangeUntil(n int) []int {
	res := make([]int, 0, n)

	for i := 0; i < n; i++ {
		res = append(res, i)
	}

	return res
}

func assertAllElementsUnique[T comparable](assertions *require.Assertions, list []T) {
	m := make(map[T]bool, len(list))

	for _, element := range list {
		m[element] = true
	}

	assertions.Equal(len(list), len(m))
}

func TestEnvBuilder_getBuilder(t *testing.T) {
	const numDefinedEnvVars = 19

	t.Run(
		"each_defined_var_has_unique_function", func(t *testing.T) {
			assertions := require.New(t)

			eb := &envBuilder{}

			allBuilders := list.NewListMapTo[int, uintptr]().MapTo(
				rangeUntil(numDefinedEnvVars),
				func(envVar int) uintptr {
					method := eb.getBuilder(EnvVar(envVar))

					return reflect.ValueOf(method).Pointer()
				},
			)

			assertions.Len(allBuilders, numDefinedEnvVars)
			assertAllElementsUnique(assertions, allBuilders)
		},
	)

	t.Run(
		"panics_above_defined_limit", func(t *testing.T) {
			assertions := require.New(t)

			eb := &envBuilder{}

			assertions.PanicsWithError(
				"unknown environment variable requested", func() {
					eb.getBuilder(numDefinedEnvVars)
				},
			)
		},
	)
}
