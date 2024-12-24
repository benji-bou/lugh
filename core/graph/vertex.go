package graph

import (
	"context"

	"github.com/benji-bou/chantools"
)

type Worker[K any] interface {
	Work(ctx context.Context, input K) ([]K, error)
}

type Runner[K any] interface {
	Run(ctx context.Context, input <-chan K) (<-chan K, <-chan error)
}

type IOWorker[K any] interface {
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
	inputCListener chan (<-chan K)
	outputC        chan K
	errC           chan error
	ctxC           chan context.Context
	worker         Worker[K]
}

func NewIOWorkerFromWorker[K any](worker Worker[K]) IOWorker[K] {
	v := &syncWorker[K]{
		inputCListener: make(chan (<-chan K)),
		outputC:        make(chan K),
		errC:           make(chan error),
		ctxC:           make(chan context.Context),
		worker:         worker,
	}
	go v.startLoop()
	return v
}

func (v *syncWorker[K]) SetInput(input <-chan K) {
	v.inputCListener <- input
}

func (v *syncWorker[K]) Output() <-chan K {
	return v.outputC
}

func (v *syncWorker[K]) Run(ctx context.Context) <-chan error {
	v.ctxC <- ctx
	return v.errC

}

func (v *syncWorker[K]) startLoop() {
	currentInput := make(<-chan K)
	currentCtx := context.Background()
	defer close(v.errC)
	defer close(v.outputC)

	for {
		select {
		case ctx := <-v.ctxC:
			currentCtx = ctx
		case newInput := <-v.inputCListener:
			currentInput = newInput
		case data, ok := <-currentInput:
			if !ok {
				return
			}
			output, err := v.worker.Work(currentCtx, data)
			if err != nil {
				v.errC <- err
				break
			}
			for _, output := range output {
				v.outputC <- output
			}
		case <-currentCtx.Done():
			return
		}
	}
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

func (v *runWorker[K]) Run(ctx context.Context) <-chan error {
	outptutC, errC := v.runner.Run(ctx, v.input)
	go chantools.ForwardTo(ctx, outptutC, v.outputC)
	return errC

}

func NewIOWorkerFromRunner[K any](runner Runner[K]) IOWorker[K] {
	v := &runWorker[K]{
		runner:  runner,
		outputC: make(chan K),
	}
	return v
}
