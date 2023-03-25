package env

import (
	"fmt"

	"github.com/instana/instana-agent-operator/pkg/optional"
	corev1 "k8s.io/api/core/v1"
)

type EnvBuilder interface {
	optional.Builder[corev1.EnvVar]
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

type fromLiteral struct {
	corev1.EnvVar
}

func fromLiteralVal(val corev1.EnvVar) EnvBuilder {
	return &fromLiteral{
		EnvVar: val,
	}
}

func (f *fromLiteral) Build() optional.Optional[corev1.EnvVar] {
	return optional.Of(f.EnvVar)
}
