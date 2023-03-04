package optional

type optional[T any] struct {
	val *T
}

func (o *optional[T]) IsEmpty() bool {
	return o.val == nil
}

func (o *optional[T]) Get() *T {
	return o.val
}

func (o *optional[T]) GetOrElse(val T) T {
	return o.GetOrElseDo(func() T {
		return val
	})
}

func (o *optional[T]) GetOrElseDo(f func() T) T {
	switch o.IsEmpty() {
	case true:
		return f()
	default:
		return *o.val
	}
}

type Optional[T any] interface {
	IsEmpty() bool
	Get() *T
	GetOrElse(val T) T
	GetOrElseDo(func() T) T
}

func Empty[T any]() Optional[T] {
	return &optional[T]{}
}

func Of[T any](val T) Optional[T] {
	return &optional[T]{
		val: &val,
	}
}

func OfNilable[T any](val *T) Optional[T] {
	return &optional[T]{
		val: val,
	}
}