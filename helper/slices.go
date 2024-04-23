package helper

func Filter[T any, A ~[]T](input A, filter func(elem T) bool) A {
	output := make(A, 0, len(input))
	for _, e := range input {
		if filter(e) {
			output = append(output, e)
		}
	}
	return output
}
