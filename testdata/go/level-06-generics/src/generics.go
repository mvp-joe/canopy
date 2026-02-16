package generics

type Pair[T any] struct {
	First  T
	Second T
}

func NewPair[T any](a, b T) Pair[T] {
	return Pair[T]{First: a, Second: b}
}

func Map[T any, U any](items []T, fn func(T) U) []U {
	result := make([]U, len(items))
	for i, v := range items {
		result[i] = fn(v)
	}
	return result
}
