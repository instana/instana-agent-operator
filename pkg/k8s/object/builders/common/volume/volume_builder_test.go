package volume

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	instanav1 "github.com/instana/instana-agent-operator/api/v1"
	"github.com/instana/instana-agent-operator/pkg/collections/list"
)

const numDefinedVolumes = 15

func rangeUntil(n int) []Volume {
	res := make([]Volume, 0, n)

	for i := 0; i < n; i++ {
		res = append(res, Volume(i))
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

func TestVolumeBuilder_getBuilder(t *testing.T) {
	t.Run(
		"each_defined_var_has_unique_function", func(t *testing.T) {
			assertions := require.New(t)

			vb := &volumeBuilder{}

			allBuilders := list.NewListMapTo[Volume, uintptr]().MapTo(
				rangeUntil(numDefinedVolumes),
				func(volume Volume) uintptr {
					method := vb.getBuilder(volume)

					return reflect.ValueOf(method).Pointer()
				},
			)

			assertions.Len(allBuilders, numDefinedVolumes)
			assertAllElementsUnique(assertions, allBuilders)
		},
	)

	t.Run(
		"panics_above_defined_limit", func(t *testing.T) {
			assertions := require.New(t)

			vb := &volumeBuilder{}

			assertions.PanicsWithError(
				"unknown volume requested", func() {
					vb.getBuilder(numDefinedVolumes)
				},
			)
		},
	)
}

// TODO: Improvements here

func TestVolumeBuilder_Build(t *testing.T) {
	for _, test := range []struct {
		name               string
		isOpenShift        bool
		expectedNumVolumes int
	}{
		{
			name:               "isOpenShift",
			isOpenShift:        true,
			expectedNumVolumes: 10,
		},
		{
			name:               "isNotOpenShift",
			isOpenShift:        false,
			expectedNumVolumes: 13,
		},
	} {
		t.Run(
			test.name, func(t *testing.T) {
				assertions := require.New(t)

				vb := NewVolumeBuilder(&instanav1.InstanaAgent{}, test.isOpenShift)

				actualvolumes, actualVolumeMounts := vb.Build(rangeUntil(numDefinedVolumes)...)

				assertions.Len(actualvolumes, test.expectedNumVolumes)
				assertions.Len(actualVolumeMounts, test.expectedNumVolumes)
			},
		)
	}
}
