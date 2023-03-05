package pointer

func ToPointer[T any](in T) *T {
	return &in
}
