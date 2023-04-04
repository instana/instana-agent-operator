package pointer

import "github.com/instana/instana-agent-operator/pkg/optional"

func To[T any](in T) *T {
	return &in
}

func DerefOrEmpty[T any](in *T) T {
	return optional.Map[*T, T](
		optional.Of(in), func(in *T) T {
			return *in
		},
	).Get()
}

func DerefOrDefault[T any](in *T, def T) T {
	return *optional.Of(in).GetOrDefault(&def)
}
