package graph

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"slices"

	"github.com/benji-bou/diwo"
	"github.com/benji-bou/lugh/helper"
	"github.com/dominikbraun/graph"
)

var ErrIsRunning = errors.New("operation not permited while graph is running")

type IOGraph[K any] struct {
	GraphSelfDescribe[string, IOWorkerVertex[K]]
	isRunning     bool
	innerCancelFn context.CancelFunc
	hasInputSet   bool
}

func WithVertices[K any](it iter.Seq[IOWorkerVertex[K]]) IOGraphOption[K] {
	return func(configure *IOGraph[K]) {
		err := configure.AddVertices(it)
		if err != nil {
			slog.Warn("failed to add IOWorkerVertex iterator", "error", err)
		}
	}
}

type IOGraphOption[K any] func(*IOGraph[K])

func NewIO[K any](opt ...IOGraphOption[K]) *IOGraph[K] {
	sg := &IOGraph[K]{}
	sg.GraphSelfDescribe = GraphSelfDescribe[string, IOWorkerVertex[K]]{New(func(spp IOWorkerVertex[K]) string {
		return spp.GetName()
	}, graph.Directed())}
	for _, o := range opt {
		if o != nil {
			o(sg)
		}
	}
	return sg
}

func (*IOGraph[K]) MergeVertexOutput(vertices iter.Seq[IOWorker[K]]) <-chan K {
	resC := helper.IterMap(vertices, func(vertex IOWorker[K]) <-chan K {
		return vertex.Output()
	})
	return diwo.Merge(slices.Collect(resC)...)
}

func (sg *IOGraph[K]) initialize() error {
	slog.Debug("Initializing graph")
	parentsMap, err := sg.PredecessorVertices()
	if err != nil {
		return fmt.Errorf("error while initializing graph: %w", err)
	}
	for currentVertexHash, parentVertices := range parentsMap {
		slog.Debug("Initializing vertex", "vertex", currentVertexHash)
		currentVertex, err := sg.Vertex(currentVertexHash)
		if err != nil {
			slog.Error("Error while initializing graph: ", "error", err, "vertexHash", currentVertexHash)
			return fmt.Errorf("error while initializing graph: %w", err)
		}
		if len(parentVertices) == 0 && sg.hasInputSet {
			continue
		}
		parentsMapValuesIterator := maps.Values(parentVertices)

		slog.Debug("Initializing vertex set input",
			"vertex", currentVertexHash,
			"parents", slices.Collect(
				helper.IterMap(parentsMapValuesIterator,
					func(elem IOWorkerVertex[K]) string {
						return elem.GetName()
					},
				),
			))
		currentVertex.SetInput(
			sg.MergeVertexOutput(
				helper.IterMap(parentsMapValuesIterator,
					func(elem IOWorkerVertex[K]) IOWorker[K] {
						return elem
					},
				),
			),
		)
	}
	slog.Debug("End of initializing graph")
	return nil
}

func (sg *IOGraph[K]) start(ctx context.Context) <-chan error {
	ctxSync := NewContext(ctx)
	vertexMap, _ := sg.AdjacencyMap()
	if len(vertexMap) == 0 {
		return diwo.Once(errors.New("empty graph. No stage loaded"))
	}
	errorsOutputC := make([]<-chan error, 0, len(vertexMap))
	for vertexHash := range vertexMap {
		vertex, err := sg.Vertex(vertexHash)
		if err != nil {
			slog.Error("Error while start running Io Graph: ", "error", err, "vertexHash", vertexHash)
			continue
		}
		slog.Debug("Starting run vertex", "vertex", vertexHash)
		errorsOutputC = append(errorsOutputC, vertex.Run(ctxSync))
	}
	slog.Debug("Wait for vertex worker synchronization")
	ctxSync.Synchronize()
	slog.Debug("End of starting graph")
	return diwo.Merge(errorsOutputC...)
}

// IOWorker Implementation

func (sg *IOGraph[K]) Run(ctx SyncContext) <-chan error {
	defer func() {
		sg.isRunning = true
	}()
	innerCtx, cancelFn := context.WithCancel(ctx)
	sg.innerCancelFn = cancelFn
	err := sg.initialize()
	if err != nil {
		return diwo.Once(err)
	}
	return sg.start(innerCtx)
}

func (sg *IOGraph[K]) SetInput(inputC <-chan K) {
	defer func() {
		sg.hasInputSet = true
	}()
	iterChildless := slices.Collect(sg.IterParentlessVertex())
	qty := len(iterChildless)
	inputBroadcast := diwo.Broadcast(inputC, qty)
	for i, vertex := range iterChildless {
		vertex.SetInput(inputBroadcast[i])
	}
}

func (sg *IOGraph[K]) Output() <-chan K {
	iterChildless := slices.Collect(
		helper.IterMap(
			sg.IterChildlessVertex(),
			func(vertexHash IOWorkerVertex[K]) <-chan K {
				return vertexHash.Output()
			},
		),
	)
	return diwo.Merge(iterChildless...)
}

// Closer Implementation

func (sg *IOGraph[K]) Close() error {
	sg.innerCancelFn()
	sg.isRunning = false
	return nil
}

func (sg *IOGraph[K]) CloneFromEdge(edge ...graph.Edge[string]) (*IOGraph[K], error) {
	mewG, err := sg.GraphSelfDescribe.CloneFromEdge(edge...)
	if err != nil {
		return nil, fmt.Errorf("error cloning inner graph: %w", err)
	}
	return &IOGraph[K]{GraphSelfDescribe: *mewG}, nil
}

func (sg *IOGraph[K]) Split() ([]*IOGraph[K], error) {
	splittedEdges, err := sg.SplitEdges()
	if err != nil {
		return nil, fmt.Errorf("failed to split graph %w", err)
	}
	res := make([]*IOGraph[K], 0, len(splittedEdges))
	for _, edges := range splittedEdges {
		newG, err := sg.CloneFromEdge(edges...)
		if err != nil {
			return nil, fmt.Errorf("failed to split graph %w", err)
		}
		res = append(res, newG)
	}
	return res, nil
}
