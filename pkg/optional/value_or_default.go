package optional

func ValueOrDefault[T any](val *T, def T) {
	*val = Of(*val).GetOrDefault(def)
}
