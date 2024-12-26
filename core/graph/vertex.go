package graph

import (
	"context"
	"log/slog"
	"reflect"

	"github.com/benji-bou/chantools"
)

type Worker[K any] interface {
	Work(ctx context.Context, input K) ([]K, error)
}

type Runner[K any] interface {
	// Run(ctx context.Context, input <-chan K, outputC chan<- K, errC chan<- error)
	Run(ctx context.Context, input <-chan K) (<-chan K, <-chan error)
}

// type workerSynchronization struct {
// 	wg sync.WaitGroup
// }

// func NewWorkerSynchronization() *workerSynchronization {
// 	return &workerSynchronization{wg: sync.WaitGroup{}}
// }
// func (ws *workerSynchronization) Initializing() {
// 	ws.wg.Add(1)
// }

// func (ws *workerSynchronization) Initialized() {
// 	ws.wg.Done()
// }
// func (ws *workerSynchronization) Synchronize() {
// 	ws.wg.Wait()
// }

type IOWorker[K any] interface {
	// Run(ctx context.Context, syncWorker *workerSynchronization) <-chan error
	Run(ctx context.Context) <-chan error
	SetInput(input <-chan K)
	Output() <-chan K
}

type IOWorkerVertex[K any] interface {
	IOWorker[K]
	GetName() string
	GetParents() []string
}

type DefaultIOWorkerVertex[K any] struct {
	IOWorker[K]
	name    string
	parents []string
}

func NewDefaultIOWorkerVertex[K any](name string, parents []string, decorated IOWorker[K]) *DefaultIOWorkerVertex[K] {
	return &DefaultIOWorkerVertex[K]{
		IOWorker: decorated,
		name:     name,
		parents:  parents,
	}
}

func (dwv *DefaultIOWorkerVertex[K]) GetName() string {
	return dwv.name
}
func (dwv *DefaultIOWorkerVertex[K]) GetParents() []string {
	return dwv.parents
}

type syncWorker[K any] struct {
	inputC  <-chan K
	outputC chan K
	worker  Worker[K]
}

func NewIOWorkerFromWorker[K any](worker Worker[K]) IOWorker[K] {
	v := &syncWorker[K]{
		outputC: make(chan K),
		worker:  worker,
	}

	return v
}

func (v *syncWorker[K]) SetInput(input <-chan K) {
	v.inputC = input
}

func (v *syncWorker[K]) Output() <-chan K {
	return v.outputC
}

func (v *syncWorker[K]) Run(ctx context.Context) <-chan error { //, syncWorker *workerSynchronization
	// syncWorker.Initializing()
	return chantools.New(func(errC chan<- error, params ...any) {
		slog.Debug("start run in syncWorker", "name", reflect.TypeOf(v.worker))
		defer func() {
			slog.Debug("sync worker closing output and error channel", "name", reflect.TypeOf(v.worker))
			close(v.outputC)
		}()
		// syncWorker.Initialized()
		for data := range v.inputC {
			slog.Debug("sync worker work", "data", data, "name", reflect.TypeOf(v.worker))
			output, err := v.worker.Work(ctx, data)
			if err != nil {
				errC <- err
				continue
			}
			for _, output := range output {
				v.outputC <- output
			}
		}
	}, chantools.WithContext[error](ctx), chantools.WithName[error](reflect.TypeOf(v.worker).String()))

}

type runWorker[K any] struct {
	input   <-chan K
	outputC chan K
	runner  Runner[K]
}

func (v *runWorker[K]) SetInput(input <-chan K) {
	v.input = input
}

func (v *runWorker[K]) Output() <-chan K {
	return v.outputC
}

func (v *runWorker[K]) Run(ctx context.Context) <-chan error { //, syncWorker *workerSynchronization
	// syncWorker.Initializing()
	slog.Debug("start run in runWorker", "name", v.runner)
	errC := make(chan error)
	go func() {
		defer func() {
			close(errC)
			close(v.outputC)
		}()
		// syncWorker.Initialized()
		outputC, errCRun := v.runner.Run(ctx, v.input)
		chantools.ForwardTo(ctx, outputC, v.outputC)
		chantools.ForwardTo(ctx, errCRun, errC)
	}()
	return errC
}

func NewIOWorkerFromRunner[K any](runner Runner[K]) IOWorker[K] {
	v := &runWorker[K]{
		runner:  runner,
		outputC: make(chan K),
	}
	return v
}
