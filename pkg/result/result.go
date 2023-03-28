package result

import (
	"errors"
	"github.com/instana/instana-agent-operator/pkg/optional"
)

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

func (r *result[T]) IsSuccess() bool {
	return errors.Is(r.err, nil)
}

func (r *result[T]) IsFailure() bool {
	return !r.IsSuccess()
}

func (r *result[T]) Get() (T, error) {
	return r.res, r.err
}

func (r *result[T]) ToOptional() optional.Optional[T] {
	switch r.IsSuccess() {
	case true:
		return optional.Of(r.res)
	default:
		return optional.Empty[T]()
	}
}

func (r *result[T]) OnSuccess(do func(res T)) Result[T] {
	if r.IsSuccess() {
		do(r.res)
	}
	return r
}

func (r *result[T]) OnFailure(do func(err error)) Result[T] {
	if r.IsFailure() {
		do(r.err)
	}
	return r
}

func (r *result[T]) Recover(do func(err error) Result[T]) Result[T] {
	switch r.IsFailure() {
	case true:
		return do(r.err)
	default:
		return r
	}
}

// TODO: Tests

// TODO: Of, OfFunction, ofResult, ofError, Map
