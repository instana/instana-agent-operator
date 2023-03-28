package result

import "github.com/instana/instana-agent-operator/pkg/optional"

type Result[T any] interface {
	IsSuccess() bool
	IsFailure() bool
	Get() (T, error)
	ToOptional() optional.Optional[T]
	OnSuccess(func(T)) Result[T]
	OnFailure(func(error)) Result[T]
	Recover(func(error) Result[T]) Result[T]
}

type result[T any] struct {
	res T
	err error
}

// TODO: Of, OfFunction, ofResult, ofError, Map
