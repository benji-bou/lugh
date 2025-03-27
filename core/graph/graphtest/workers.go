package graphtest

import (
	"context"
	"fmt"
	"log/slog"
	"testing"

	"github.com/benji-bou/lugh/core/graph"
	"golang.org/x/exp/constraints"
)

func SliceProducer[K any](elems ...K) graph.IOWorker[K] {
	return graph.NewIOWorkerFromProducer(
		graph.ProducerFunc[K](
			func(_ context.Context, yield func(elem K) error) error {
				for _, elem := range elems {
					if err := yield(elem); err != nil {
						return fmt.Errorf("slice producer yield failed: %w", err)
					}
				}
				return nil
			},
		),
	)
}

// ForwardWorker is a helper function that returns a Worker that just directly yield the input
func ForwardWorker[K any]() graph.IOWorker[K] {
	return graph.NewIOWorkerFromWorker(graph.WorkerFunc[K](func(_ context.Context, input K, yield func(elem K) error) error {
		slog.Debug("ForwardWorker reveived data", "data", input)
		return yield(input)
	}))
}

// MultWorker is a helper function that returns a Worker that multiplies the input by a given multiplier
func MultWorker[K constraints.Integer](multiplcator K) graph.IOWorker[K] {
	return graph.NewIOWorkerFromWorker(graph.WorkerFunc[K](func(_ context.Context, input K, yield func(elem K) error) error {
		slog.Debug("MultWorker reveived data", "data", input)
		return yield(input * multiplcator)
	}))
}

// EvenWorker is a helper function that returns a Worker that yields only even numbers
func EvenWorker[K constraints.Integer]() graph.IOWorker[K] {
	return graph.NewIOWorkerFromWorker(graph.WorkerFunc[K](func(_ context.Context, input K, yield func(elem K) error) error {
		slog.Debug("EvenWorker reveived data", "data", input)
		if input%2 == 0 {
			return yield(input)
		}
		return nil
	}))
}

// OddWorker is a helper function that returns a Worker that yields only odd numbers
func OddWorker[K constraints.Integer]() graph.IOWorker[K] {
	return graph.NewIOWorkerFromWorker(graph.WorkerFunc[K](func(_ context.Context, input K, yield func(elem K) error) error {
		slog.Debug("OddWorker reveived data", "data", input)
		if input%2 != 0 {
			return yield(input)
		}
		return nil
	}))
}

// SingleForwardWorker is an helper function that Run a `ForwardWorkerTestable`
// and returns the input channel, output channel, and error channel of the worker
// it synchromizes the worker before returning it.
func SingleForwardWorker() (inputC chan<- []byte, outputC <-chan []byte, errC <-chan error) {
	inputTestC := make(chan []byte)
	v := ForwardWorker[[]byte]()
	v.SetInput(inputTestC)
	outputC = v.Output()
	ctx := graph.NewContext(context.Background())
	errC = v.Run(ctx) // , NewWorkerSynchronization()
	ctx.Synchronize()
	return inputTestC, outputC, errC
}

// NonRunningSingleForwardWorker is an helper function that Instanciates a `ForwardWorkerTestable`
// BUT it does not run the worker.
// Useful to test the behavior of the initialisation of the worker. Without running the worker.
func NonRunningSingleForwardWorker() (inputC chan<- []byte, outputC <-chan []byte, errC <-chan error) {
	inputTestC := make(chan []byte)
	v := ForwardWorker[[]byte]()
	v.SetInput(inputTestC)
	outputC = v.Output()
	return inputTestC, outputC, nil
}

// GraphWorker is an helper function that Run a `graph as a IOWorker`  setting input and output of the graph`
func GraphWorker[K any](workers []graph.IOWorkerVertex[K]) IOFunc[K] {
	gr := graph.NewIO(graph.WithVertices(workers))
	return func() (chan<- K, <-chan K, <-chan error) {
		inputC := make(chan K)
		ctx := graph.NewContext(context.Background())
		gr.SetInput(inputC)
		errC := gr.Run(ctx)
		return inputC, gr.Output(), errC
	}
}

func GenerateWorkerProducer[K any](ctx graph.SyncContext, elements ...K) (worker graph.IOWorker[K], workerStarter func()) {
	worker = SliceProducer(elements...)
	return worker, func() {
		worker.Run(ctx)
	}
}

func GenerateWorkerProducers[K any, A ~[]K](ctx graph.SyncContext, elements ...A) (workers []graph.IOWorker[K], workersStarter func()) {
	workers = make([]graph.IOWorker[K], 0, len(elements))
	runs := make([]func(), 0, len(elements))
	for i := range elements {
		w, run := GenerateWorkerProducer[K](ctx, elements[i]...)
		workers = append(workers, w)
		runs = append(runs, run)
	}
	return workers, func() {
		for _, r := range runs {
			r()
		}
	}
}

var (
	SingleForward = []graph.IOWorkerVertex[int]{
		graph.NewIOWorkerVertex("forward", []string{},
			ForwardWorker[int]()),
	}
	LinearWorkerChainMult2 = []graph.IOWorkerVertex[int]{
		graph.NewIOWorkerVertex("forward", []string{},
			ForwardWorker[int]()),
		graph.NewIOWorkerVertex("mult11", []string{"forward"},
			MultWorker[int](11)), //nolint:mnd // test 11 is expected here
		graph.NewIOWorkerVertex("odd", []string{"mult11"},
			OddWorker[int]()),
	}
	SplitWorkerChainMult2_3 = []graph.IOWorkerVertex[int]{
		graph.NewIOWorkerVertex("forward", []string{},
			ForwardWorker[int]()),
		graph.NewIOWorkerVertex("mult2", []string{"forward"},
			MultWorker[int](2)),
		graph.NewIOWorkerVertex("mult3", []string{"forward"},
			MultWorker[int](3)),
		graph.NewIOWorkerVertex("even", []string{"mult3"},
			EvenWorker[int]()),
	}
)

// TestWorkerChain is an helper function that test the `workerConfigTest` provided
// Retrieve the input/output/error chan of the configurated worker(s)
// Use the dataTest provided in the config and send it to the input of the worker(s)
// send the dataTested the output and error chan of the worker(s) to the assert function of the config
// The assert function is used to test the behavior of the worker(s)
func TestWorkerChain[K any](t *testing.T, testConfig WorkerConfigTest[K]) {
	t.Helper()
	inputTestC, outputTestC, errC := testConfig.IO()
	dataTest := testConfig.DataTest
	t.Run(testConfig.Name, func(t *testing.T) {
		go testConfig.Assert(t, testConfig.ExpectedOutput, outputTestC, errC)
		go func() {
			for _, data := range dataTest {
				inputTestC <- data
			}
			close(inputTestC)
		}()
	})
}
