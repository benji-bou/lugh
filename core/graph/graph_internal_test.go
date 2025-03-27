package graph

import (
	"context"
	"fmt"
	"testing"
)

func ForwardWorker[K any]() IOWorker[K] {
	return NewIOWorkerFromWorker(WorkerFunc[K](func(_ context.Context, input K, yield func(elem K) error) error {
		return yield(input)
	}))
}

func TestInitialize(t *testing.T) {
	forwardWorker := ForwardWorker[int]()
	g := NewIO[int](WithVertices(
		[]IOWorkerVertex[int]{
			NewIOWorkerVertex(
				"forward",
				[]string{},
				ForwardWorker[int](),
			),
		},
	),
	)
	t.Run("SetInput on worker", func(t *testing.T) {
		inputWorker := make(chan int)
		forwardWorker.SetInput(inputWorker)
		err := g.piping()
		if err != nil {
			t.Errorf("Initialize graph failed %s", err.Error())
		}
		select {
		case _, ok := <-inputWorker:
			if !ok {
				t.Errorf("Input channel is closed after initialization")
			}
		default:
		}
	})
	t.Run("SetInput on graph", func(t *testing.T) {
		inputgraph := make(chan int)
		g.SetInput(inputgraph)
		err := g.piping()
		if err != nil {
			t.Errorf("Initialize graph failed %s", err.Error())
		}
		forward, err := g.Vertex("forward")
		if err != nil {
			t.Errorf("Get vertex failed %s", err.Error())
			return
		}
		workerInputC := forward.IOWorker.(*BroadcasterIOWorker[int]).IOWorker.(*syncWorker[int]).inputC

		select {
		case _, ok := <-inputgraph:
			if !ok {
				t.Errorf("Graph Input channel is closed after initialization")
			}
		case _, ok := <-workerInputC:
			if !ok {
				t.Errorf("parentless Worker Input channel is closed after initialization")
			}
		default:
			fmt.Printf("ok")
		}
	})
}
