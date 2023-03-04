package list

type list[T any] struct {
	raw []T
} // TODO: switch to collection use map as backer then add constructor for map and for list where key is the position

// TODO: filter, map, forEach, add(...)

// TODO: other todo, owned resources, exponential backoff config, general transformers interface + implement (common labels + owner refs), apply all function, basic controller tasks, then status later on
