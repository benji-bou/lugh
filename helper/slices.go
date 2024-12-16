package helper

import "iter"

func Map[T any, A iter.Seq[T], O any](input A, mapper func(elem T) O) iter.Seq[O] {
	return func(yield func(O) bool) {
		for e := range input {
			mapped := mapper(e)
			if !yield(mapped) {
				return
			}
		}
	}
}
