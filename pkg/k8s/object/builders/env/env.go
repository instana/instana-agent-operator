package env

import (
	"fmt"

	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

type EnvBuilder interface {
	Build() optional.Optional[corev1.EnvVar]
}

type fromFieldIfSet[T any] struct {
	name          string
	providedValue optional.Optional[T]
}

func fromField[T any](name string, val T) EnvBuilder {
	return &fromFieldIfSet[T]{
		name:          name,
		providedValue: optional.Of(val),
	}
}

func (f *fromFieldIfSet[T]) Build() optional.Optional[corev1.EnvVar] {
	switch f.providedValue.IsEmpty() {
	case true:
		return optional.Empty[corev1.EnvVar]()
	default:
		return optional.Of(corev1.EnvVar{
			Name:  f.name,
			Value: fmt.Sprintf("%v", f.providedValue.Get()),
		})
	}
}

// TODO: Add optional "Map" function for more generality, also add execute when not empty, etc
// TODO: Test
