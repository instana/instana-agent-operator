package optional

import "reflect"

type optional[T any] struct {
	val T
}

func (o *optional[T]) IsEmpty() bool {
	valueOf := reflect.ValueOf(o.val)
	return !valueOf.IsValid() || valueOf.IsZero()
}

func (o *optional[T]) IsNotEmpty() bool {
	return !o.IsEmpty()
}

func (o *optional[T]) Get() T {
	return o.val
}

func (o *optional[T]) GetOrDefault(val T) T {
	return o.GetOrElse(
		func() T {
			return val
		},
	)
}

func (o *optional[T]) GetOrElse(f func() T) T {
	switch o.IsEmpty() {
	case true:
		return f()
	default:
		return o.val
	}
}

func (o *optional[T]) IfPresent(do func(val T)) {
	if o.IsNotEmpty() {
		do(o.Get())
	}
}

type Optional[T any] interface {
	IsEmpty() bool
	IsNotEmpty() bool
	Get() T
	GetOrDefault(val T) T
	GetOrElse(func() T) T
	IfPresent(func(T))
}

func Empty[T any]() Optional[T] {
	return &optional[T]{}
}

func Of[T any](val T) Optional[T] {
	return &optional[T]{
		val: val,
	}
}

func Map[T any, U any](in Optional[T], transform func(in T) U) Optional[U] {
	switch in.IsEmpty() {
	case true:
		return Empty[U]()
	default:
		return Of(transform(in.Get()))
	}
}
