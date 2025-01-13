package graphtest

import (
	"testing"
)

type (
	IOFunc[K any]     func() (chan<- K, <-chan K, <-chan error)
	AssertFunc[K any] func(t *testing.T, dataTest []K, outputC <-chan K, errC <-chan error)
)

// WorkerConfigTest is a struct that contains the configuration for a worker test.
// `f` is the function which generate the input/output and error chan to test.
// the f function should generate workers chain and return the input/output and error chan of the chain.
// `name` of the test,
// `dataTest`the data to be tested and sent to the input chan of the worker chains io returned by `f`
// `asserF` the function which compare the output with expected data.
// This is the function deciding if the test failed or succeeded.
type WorkerConfigTest[K any] struct {
	Name           string
	DataTest       []K
	ExpectedOutput []K
	IO             IOFunc[K]
	Assert         AssertFunc[K]
}
