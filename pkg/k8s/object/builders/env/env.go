package env

import (
	"fmt"

	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

func fromCRField[T any](name string, val T) optional.Optional[corev1.EnvVar] {
	providedVal := optional.Of(val)
	return optional.Map(providedVal, func(in T) corev1.EnvVar {
		return corev1.EnvVar{
			Name:  name,
			Value: fmt.Sprintf("%v", providedVal.Get()),
		}
	})
}
