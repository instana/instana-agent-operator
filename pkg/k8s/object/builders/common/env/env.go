package env

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"

	"github.com/instana/instana-agent-operator/pkg/optional"
)

func fromCRField[T any](name string, val T) optional.Optional[corev1.EnvVar] {
	return optional.Map(
		optional.Of(val), func(v T) corev1.EnvVar {
			return corev1.EnvVar{
				Name:  name,
				Value: fmt.Sprintf("%v", v),
			}
		},
	)
}
