package set

import "iter"

type Set[T comparable] struct {
	index map[T]struct{}
	array []T
}

func New[T comparable](elem ...T) *Set[T] {
	s := &Set[T]{index: make(map[T]struct{}, len(elem)), array: make([]T, len(elem))}
	for i, e := range elem {
		s.index[e] = struct{}{}
		s.array[i] = e
	}
	return s
}

func (s *Set[T]) Add(elem ...T) {
	for _, e := range elem {
		if _, ok := s.index[e]; !ok {
			s.index[e] = struct{}{}
			s.array = append(s.array, e)
		}
	}
}

func (s *Set[T]) Remove(e T) {
	if _, ok := s.index[e]; ok {
		delete(s.index, e)
		for i, v := range s.array {
			if v == e {
				if i+1 == len(s.array) {
					s.array = s.array[:i:i]
				} else {
					s.array = append(s.array[:i], s.array[i+1:len(s.array)-1]...)
				}
			}
		}
	}
}

func (s *Set[T]) Contains(e T) bool {
	_, ok := s.index[e]
	return ok
}

func (s *Set[T]) Len() int {
	return len(s.array)
}

func (s *Set[T]) Values() []T {
	return s.array
}

func (s *Set[T]) Clear() {
	clear(s.index)
	s.array = s.array[:0]
}

func (s *Set[T]) Union(other *Set[T]) *Set[T] {
	result := New[T](s.Values()...)
	result.Add(other.Values()...)
	return result
}

func (s *Set[T]) Intersection(other *Set[T]) *Set[T] {
	result := New[T]()
	for _, v := range s.array {
		if other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

func (s *Set[T]) Difference(other *Set[T]) *Set[T] {
	result := New[T]()
	for _, v := range s.array {
		if !other.Contains(v) {
			result.Add(v)
		}
	}
	return result
}

func (s *Set[T]) IsEmpty() bool {
	return len(s.array) == 0
}

func (s *Set[T]) Next() iter.Seq[T] {
	return func(yield func(T) bool) {
		for i := 0; i < len(s.array); i++ { //nolint:intrange // do not use range because
			// we want to iterate over array that can growth while iterating
			if !yield(s.array[i]) {
				return
			}
		}
	}
}
