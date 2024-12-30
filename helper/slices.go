package helper

import "iter"

func IterMap[T any, A iter.Seq[T], O any](input A, mapper func(elem T) O) iter.Seq[O] {
	return func(yield func(O) bool) {
		for e := range input {
			mapped := mapper(e)
			if !yield(mapped) {
				return
			}
		}
	}
}

func Map[T any, A ~[]T, O any](input A, mapper func(elem T) O) []O {
	res := make([]O, len(input))
	for i, e := range input {
		res[i] = mapper(e)
	}
	return res
}

func Flatten[T any, A ~[]T, F ~[]A](input F) iter.Seq[T] {
	return func(yield func(T) bool) {
		for _, subInput := range input {
			for _, e := range subInput {
				if !yield(e) {
					return
				}
			}
		}
	}
}
