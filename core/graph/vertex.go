package graph

import (
	"context"
	"log"
	"sync"

	"github.com/benji-bou/chantools"
)

type RunnableWorker interface {
	Run(ctx context.Context) <-chan error
}

type IOWorker[K any] interface {
	RunnableWorker
	SetInput(input <-chan K)
	Output() <-chan K
}

type IOWorkerVertex[K any] interface {
	IOWorker[K]
	GetName() string
	GetParents() []string
}

type DefaultIOWorker[K any] struct {
	inputC         <-chan K
	outputC        <-chan K
	isRunning      bool
	isRunningMutex sync.Mutex
	decorate       RunnableWorker
}

func NewDefaultIOWorker[K any](decorated RunnableWorker) IOWorker[K] {
	return &DefaultIOWorker[K]{
		inputC:         nil,
		outputC:        nil,
		isRunning:      false,
		isRunningMutex: sync.Mutex{},
	}
}

func (dw *DefaultIOWorker[K]) SetInput(inputC <-chan K) {
	dw.isRunningMutex.Lock()
	defer dw.isRunningMutex.Unlock()
	if dw.isRunning {
		log.Fatalf("cannot add input after DefaultIOWorker is running")
	}
	dw.inputC = inputC
}

func (dw *DefaultIOWorker[K]) GetInput() <-chan K {
	return dw.inputC
}

func (dw *DefaultIOWorker[K]) Output() <-chan K {
	return dw.outputC
}

func (dw *DefaultIOWorker[K]) SetOutput(output <-chan K) {
	dw.isRunningMutex.Lock()
	defer dw.isRunningMutex.Unlock()
	if dw.isRunning {
		log.Fatalf("cannot add output after DefaultIOWorker is running")
	}
	dw.outputC = output
}

func (dw *DefaultIOWorker[K]) Run(ctx context.Context) <-chan error {
	dw.UpdateRun(true)
	return chantools.New(func(c chan<- error, params ...any) {
		chantools.ForwardTo(ctx, dw.decorate.Run(ctx), c)
	}, chantools.WithTearDown[error](func() {
		dw.UpdateRun(false)
	}))
}

func (dw *DefaultIOWorker[K]) UpdateRun(isRunning bool) {
	dw.isRunningMutex.Lock()
	defer dw.isRunningMutex.Unlock()
	dw.isRunning = isRunning
}

type DefaultIOWorkerVertex[K any] struct {
	IOWorker[K]
	name    string
	parents []string
}

func NewDefaultIOWorkerVertex[K any](name string, parents []string, decorated RunnableWorker) *DefaultIOWorkerVertex[K] {
	return &DefaultIOWorkerVertex[K]{
		IOWorker: NewDefaultIOWorker[K](decorated),
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
