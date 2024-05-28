package optional

import "reflect"

type optional[T any] struct {
	val T
}

func (o *optional[T]) IsNotPresent() bool {
	valueOf := reflect.ValueOf(o.val)
	return !valueOf.IsValid() || valueOf.IsZero()
}

func (o *optional[T]) IsPresent() bool {
	return !o.IsNotPresent()
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
	switch o.IsNotPresent() {
	case true:
		return f()
	default:
		return o.val
	}
}

func (o *optional[T]) IfPresent(do func(val T)) {
	if o.IsPresent() {
		do(o.Get())
	}
}

type Optional[T any] interface {
	IsNotPresent() bool
	IsPresent() bool
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
	switch in.IsNotPresent() {
	case true:
		return Empty[U]()
	default:
		return Of(transform(in.Get()))
	}
}
