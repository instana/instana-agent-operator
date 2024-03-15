package or_die

type orDie[T any] struct{}

func (o *orDie[T]) ResultOrDie(resultAndError func() (T, error)) T {
	switch res, err := resultAndError(); err == nil {
	case true:
		return res
	default:
		panic(err)
	}
}

type OrDie[T any] interface {
	ResultOrDie(resultAndError func() (T, error)) T
}

func New[T any]() OrDie[T] {
	return &orDie[T]{}
}
