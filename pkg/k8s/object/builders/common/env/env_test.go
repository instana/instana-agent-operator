package env

import (
	"testing"

	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
)

func Test_fromCRField(t *testing.T) {
	t.Run(
		"when_empty", func(t *testing.T) {
			assertions := require.New(t)
			actual := fromCRField("MY_ENV_FIELD_1", "")

			assertions.Empty(actual)
		},
	)
	t.Run(
		"with_value", func(t *testing.T) {
			assertions := require.New(t)
			actual := fromCRField("MY_ENV_FIELD_1", "ewoihsdoighds")

			assertions.Equal(
				optional.Of(
					corev1.EnvVar{
						Name:  "MY_ENV_FIELD_1",
						Value: "ewoihsdoighds",
					},
				),
				actual,
			)
		},
	)
}
