package helper

type (
	Option[T any]      func(configure *T)
	OptionError[T any] func(configure *T) error
)

func Configure[T any, O Option[T]](input T, opt ...O) T {
	for _, o := range opt {
		if o != nil {
			o(&input)
		}
	}
	return input
}

func ConfigurePtr[T any, O Option[T]](input *T, opt ...O) *T {
	for _, o := range opt {
		if o != nil {
			o(input)
		}
	}
	return input
}

func ConfigureWithError[T any, O OptionError[T]](input T, opt ...O) (T, error) {
	for _, o := range opt {
		if o != nil {
			e := o(&input)
			if e != nil {
				return input, e
			}
		}
	}
	return input, nil
}

func ConfigurePtrWithError[T any, O OptionError[T]](input *T, opt ...O) (*T, error) {
	for _, o := range opt {
		if o != nil {
			e := o(input)
			if e != nil {
				return input, e
			}
		}
	}
	return input, nil
}
