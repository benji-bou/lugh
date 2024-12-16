package graph

import (
	"context"
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
