package result

import (
	"errors"

	"github.com/instana/instana-agent-operator/pkg/optional"
	"github.com/instana/instana-agent-operator/pkg/recovery"
)

type Result[T any] interface {
	IsSuccess() bool
	IsFailure() bool
	Get() (T, error)
	ToOptional() optional.Optional[T]
	OnSuccess(func(T)) Result[T]
	OnFailure(func(error)) Result[T]
	Recover(func(error) (T, error)) Result[T]
	RecoverCatching(func(error) (T, error)) Result[T]
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

func (r *result[T]) recover(getRecoveredResult func() Result[T]) Result[T] {
	switch r.IsFailure() {
	case true:
		return getRecoveredResult()
	default:
		return r
	}
}

func (r *result[T]) Recover(do func(err error) (T, error)) Result[T] {
	return r.recover(
		func() Result[T] {
			return Of(do(r.err))
		},
	)
}

func (r *result[T]) RecoverCatching(do func(err error) (T, error)) Result[T] {
	return r.recover(
		func() Result[T] {
			return OfInlineCatchingPanic(
				func() (res T, err error) {
					return do(r.err)
				},
			)
		},
	)
}

func Of[T any](res T, err error) Result[T] {
	return &result[T]{
		res: res,
		err: err,
	}
}

func OfInline[T any](do func() (res T, err error)) Result[T] {
	return Of(do())
}

func OfInlineCatchingPanic[T any](do func() (res T, err error)) Result[T] {
	return OfInline[T](
		func() (res T, err error) {
			defer recovery.Catch(&err)
			return do()
		},
	)
}

func OfSuccess[T any](res T) Result[T] {
	return Of(res, nil)
}

func OfFailure[T any](err error) Result[T] {
	return &result[T]{
		err: err,
	}
}

func Map[I any, O any](original Result[I], mapper func(res I) Result[O]) Result[O] {
	switch res, err := original.Get(); original.IsSuccess() {
	case true:
		return mapper(res)
	default:
		return OfFailure[O](err)
	}
}
