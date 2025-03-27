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

type IO[K any] struct {
	GraphSelfDescribe[string, IOWorkerVertex[K]]
	innerCancelFn context.CancelFunc
	inputC        *diwo.Broker[K]
	outputC       <-chan K
}

func WithVertices[K any](it []IOWorkerVertex[K]) IOGraphOption[K] {
	return func(configure *IO[K]) {
		err := configure.AddVertices(it)
		if err != nil {
			slog.Warn("failed to add IOWorkerVertex iterator", "error", err)
		}
	}
}

type IOGraphOption[K any] func(*IO[K])

func NewIO[K any](opt ...IOGraphOption[K]) *IO[K] {
	sg := &IO[K]{}
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

func (*IO[K]) MergeVertexOutput(vertices iter.Seq[IOWorker[K]]) <-chan K {
	resC := helper.IterMap(vertices, func(vertex IOWorker[K]) <-chan K {
		return vertex.Output()
	})
	return diwo.Merge(slices.Collect(resC)...)
}

func (sg *IO[K]) piping() error {
	slog.Debug("piping: graph")
	parentsMap, err := sg.PredecessorVertices()
	if err != nil {
		return fmt.Errorf("error while piping graph: %w", err)
	}
	for currentVertexHash, parentVertices := range parentsMap {
		slog.Debug("piping: vertex", "vertex", currentVertexHash)
		currentVertex, err := sg.Vertex(currentVertexHash)
		if err != nil {
			slog.Error("Error while piping graph: ", "error", err, "vertexHash", currentVertexHash)
			return fmt.Errorf("error while piping graph: %w", err)
		}
		if len(parentVertices) == 0 && sg.inputC != nil {
			slog.Debug("piping: vertex set input", "vertex", currentVertexHash, "input", "inputC")
			currentVertex.SetInput(sg.inputC.Subscribe())

		} else {
			parentsMapValuesIterator := maps.Values(parentVertices)

			slog.Debug("piping: vertex set input",
				"vertex", currentVertexHash,
				"parentsOutput", slices.Collect(
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
	}
	slog.Debug("piping: End of piping graph")
	return nil
}

func (sg *IO[K]) start(ctx SyncContext) <-chan error {
	slog.Debug("start: Io Graph")

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
		slog.Debug("start: Starting run vertex", "vertex", vertexHash)
		errorsOutputC = append(errorsOutputC, vertex.Run(ctx))
	}
	slog.Debug("start: Wait for  worker synchronization")

	slog.Debug("start: End of worker synchronization.")
	return diwo.Merge(errorsOutputC...)
}

func (sg *IO[K]) Run(ctx SyncContext) <-chan error {
	slog.Debug("run: Io Graph")
	err := sg.piping()
	if err != nil {
		return diwo.Once(err)
	}
	return sg.start(ctx)
}

// IOWorker Implementation
func (sg *IO[K]) SetInput(inputC <-chan K) {
	slog.Debug("IOGraphWorker start SetInput")

	if sg.inputC == nil && inputC != nil {
		sg.inputC = diwo.NewBroker(inputC)
	}
	slog.Debug("IOGraphWorker end SetInput")
}

func (sg *IO[K]) Output() <-chan K {
	if sg.outputC == nil {
		iterChildless := slices.Collect(
			helper.IterMap(
				sg.IterChildlessVertex(),
				func(vertexHash IOWorkerVertex[K]) <-chan K {
					return vertexHash.Output()
				},
			),
		)
		sg.outputC = diwo.Merge(iterChildless...)
	}
	return sg.outputC
}

func (sg *IO[K]) CloneFromEdge(edge ...graph.Edge[string]) (*IO[K], error) {
	mewG, err := sg.GraphSelfDescribe.CloneFromEdge(edge...)
	if err != nil {
		return nil, fmt.Errorf("error cloning inner graph: %w", err)
	}
	return &IO[K]{GraphSelfDescribe: *mewG}, nil
}

func (sg *IO[K]) Split() ([]*IO[K], error) {
	splittedEdges, err := sg.SplitEdges()
	if err != nil {
		return nil, fmt.Errorf("failed to split graph %w", err)
	}
	res := make([]*IO[K], 0, len(splittedEdges))
	for _, edges := range splittedEdges {
		newG, err := sg.CloneFromEdge(edges...)
		if err != nil {
			return nil, fmt.Errorf("failed to split graph %w", err)
		}
		res = append(res, newG)
	}
	return res, nil
}
