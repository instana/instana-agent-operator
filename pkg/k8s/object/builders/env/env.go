package env

import (
	"fmt"

	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

type EnvBuilder interface {
	Build() optional.Optional[corev1.EnvVar]
}

type fromCRFieldIfSet[T any] struct {
	name          string
	providedValue optional.Optional[T]
}

func fromCRField[T any](name string, val T) EnvBuilder {
	return &fromCRFieldIfSet[T]{
		name:          name,
		providedValue: optional.Of(val),
	}
}

func (f *fromCRFieldIfSet[T]) Build() optional.Optional[corev1.EnvVar] {
	return optional.Map(f.providedValue, func(in T) corev1.EnvVar {
		return corev1.EnvVar{
			Name:  f.name,
			Value: fmt.Sprintf("%v", f.providedValue.Get()),
		}
	})
}

// TODO: all common + function to return all common for multiple places?
