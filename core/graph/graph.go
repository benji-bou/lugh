package graph

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"os"
	"slices"

	"maps"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/chantools"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

type IOGraph[K any] struct {
	graph.Graph[string, IOWorkerVertex[K]]
}

func WithIOWorkerVertexIterator[K any](it iter.Seq[IOWorkerVertex[K]]) IOGraphOption[K] {
	return func(configure *IOGraph[K]) {
		configure.AddIOWorkerVertex(it)
	}
}

type IOGraphOption[K any] func(*IOGraph[K])

func NewGraph[K any](opt ...IOGraphOption[K]) *IOGraph[K] {
	sg := &IOGraph[K]{}
	sg.Graph = graph.New(func(spp IOWorkerVertex[K]) string {
		return spp.GetName()
	}, graph.Directed())
	for _, o := range opt {
		if o != nil {
			o(sg)
		}
	}
	return sg
}

func (sg *IOGraph[K]) DrawGraph(filepath string) error {
	file, _ := os.Create(filepath)
	return draw.DOT(sg, file)
}

func (sg *IOGraph[K]) AdjancyVertices() (map[string]map[string]IOWorkerVertex[K], error) {
	return sg.NeighborVertices(sg.AdjacencyMap)
}

func (sg *IOGraph[K]) PredecessorVertices() (map[string]map[string]IOWorkerVertex[K], error) {
	return sg.NeighborVertices(sg.PredecessorMap)
}

func (sg *IOGraph[K]) NeighborVertices(orientedNeighborSearch func() (map[string]map[string]graph.Edge[string], error)) (map[string]map[string]IOWorkerVertex[K], error) {
	verticesMaps, err := orientedNeighborSearch()
	if err != nil {
		return nil, err
	}
	res := make(map[string]map[string]IOWorkerVertex[K], len(verticesMaps))
	for currentVertexHash, neighborMap := range verticesMaps {
		res[currentVertexHash] = make(map[string]IOWorkerVertex[K], len(neighborMap))
		for neighborName := range neighborMap {
			v, err := sg.Graph.Vertex(neighborName)
			if err != nil {
				return nil, err
			}
			res[currentVertexHash][neighborName] = v
		}
	}
	return res, nil
}

func (sg *IOGraph[K]) IterChildlessVertex() iter.Seq[IOWorker[K]] {
	return sg.iterOrientedNeighborlessVertex(sg.AdjacencyMap)
}

func (sg *IOGraph[K]) IterParentlessVertex() iter.Seq[IOWorker[K]] {
	return sg.iterOrientedNeighborlessVertex(sg.PredecessorMap)
}

func (sg *IOGraph[K]) iterOrientedNeighborlessVertex(orientedNeighborSearch func() (map[string]map[string]graph.Edge[string], error)) iter.Seq[IOWorker[K]] {
	return func(yield func(IOWorker[K]) bool) {
		neighborMap, err := orientedNeighborSearch()
		if err != nil {
			panic(err)
		}
		for vertexId, neighbors := range neighborMap {
			if len(neighbors) == 0 {
				vertex, err := sg.Vertex(vertexId)
				if err != nil {
					continue
				}
				if !yield(vertex) {
					return
				}
			}
		}
	}
}

func (sg *IOGraph[K]) AddIOWorkerVertex(vertices iter.Seq[IOWorkerVertex[K]]) error {
	for newVertex := range vertices {
		err := sg.AddVertex(newVertex)
		if err != nil {
			slog.Error("Error adding vertex", "vertex", newVertex.GetName(), "error", err)

			return err
		}
	}
	for newVertex := range vertices {
		for _, p := range newVertex.GetParents() {
			err := sg.AddEdge(p, newVertex.GetName())
			if err != nil {
				slog.Error("Error adding edge", "from", p, "to", newVertex.GetName(), "error", err)
				return err
			}
		}
	}
	return nil
}

func (sg *IOGraph[K]) MergeVertexOutput(vertices iter.Seq[IOWorkerVertex[K]]) <-chan K {
	resC := helper.IterMap(vertices, func(vertex IOWorkerVertex[K]) <-chan K {
		return vertex.Output()
	})
	return chantools.Merge(slices.Collect(resC)...)

}
func (sg *IOGraph[K]) initialize() error {
	slog.Debug("Initializing graph")
	parentsMap, err := sg.PredecessorVertices()
	if err != nil {
		return fmt.Errorf("error while initializing graph: %w", err)
	}
	for currentVertexHash, parentVertex := range parentsMap {
		slog.Debug("Initializing vertex", "vertex", currentVertexHash)
		currentVertex, err := sg.Vertex(currentVertexHash)
		if err != nil {
			slog.Error("Error while initializing graph: ", "error", err, "vertexHash", currentVertexHash)
			return fmt.Errorf("error while initializing graph: %w", err)
		}

		parentsMapValuesIterator := maps.Values(parentVertex)
		slog.Debug("Initializing vertex set input", "vertex", currentVertexHash, "parents", slices.Collect(helper.IterMap(parentsMapValuesIterator, func(elem IOWorkerVertex[K]) string { return elem.GetName() })))

		currentVertex.SetInput(sg.MergeVertexOutput(parentsMapValuesIterator))
	}
	slog.Debug("End of initializing graph")
	return nil
}

func (sg *IOGraph[K]) start(ctx context.Context) <-chan error {
	ctxSync := NewContext(ctx)
	vertexMap, _ := sg.AdjacencyMap()
	if len(vertexMap) == 0 {
		return chantools.Once(errors.New("empty graph. No stage loaded"))
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
	return chantools.Merge(errorsOutputC...)
}

func (sg *IOGraph[K]) Run(ctx context.Context) <-chan error {
	err := sg.initialize()
	if err != nil {
		return chantools.Once(err)
	}
	return sg.start(ctx)
}
