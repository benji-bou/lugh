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
	Run(ctx Context, input <-chan K) (<-chan K, <-chan error)
}

type IOWorker[K any] interface {
	Run(ctx SyncContext) <-chan error
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

func (v *syncWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	return chantools.New(func(errC chan<- error, params ...any) {
		slog.Debug("start", "object", "syncWorker", "function", "Run", "name", reflect.TypeOf(v.worker))
		defer func() {
			slog.Debug("end, output close", "object", "syncWorker", "function", "Run", "name", reflect.TypeOf(v.worker))
			close(v.outputC)
		}()
		ctx.Initialized()
		for data := range v.inputC {
			slog.Debug("received data start work", "data", data, "object", "syncWorker", "function", "Run", "name", reflect.TypeOf(v.worker))
			output, err := v.worker.Work(ctx, data)
			if err != nil {
				slog.Error("work failed", "error", err, "object", "syncWorker", "function", "Run", "name", reflect.TypeOf(v.worker))
				errC <- err
				continue
			}
			for _, output := range output {
				v.outputC <- output
				slog.Debug("work success, result  sent", "data", output, "error", err, "object", "syncWorker", "function", "Run", "name", reflect.TypeOf(v.worker))
			}
		}
	}, chantools.WithName[error](reflect.TypeOf(v.worker).String()))

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

func (v *runWorker[K]) Run(ctx SyncContext) <-chan error {
	ctx.Initializing()
	errC := make(chan error)
	go func() {
		slog.Debug("start", "object", "runWorker", "function", "Run", "name", reflect.TypeOf(v.runner))
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
