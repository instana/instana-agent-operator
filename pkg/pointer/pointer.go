package pointer

func To[T any](in T) *T {
	return &in
}

func DerefOrEmpty[T any](in *T) T {
	return DerefOrElse(
		in, func() T {
			var zero T
			return zero
		},
	)
}

func DerefOrDefault[T any](in *T, def T) T {
	return DerefOrElse(
		in, func() T {
			return def
		},
	)
}

func DerefOrElse[T any](in *T, do func() T) T {
	switch in {
	case nil:
		return do()
	default:
		return *in
	}
}
