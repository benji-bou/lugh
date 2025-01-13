package graph

import (
	"context"
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"os"
	"slices"

	"github.com/benji-bou/diwo"
	"github.com/benji-bou/lugh/helper"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

var ErrIsRunning = errors.New("operation not permited while graph is running")

type NeighbourSearchFunc func() (map[string]map[string]graph.Edge[string], error)

type IOGraph[K any] struct {
	graph.Graph[string, IOWorkerVertex[K]]
	isRunning     bool
	innerCancelFn context.CancelFunc
	hasInputSet   bool
}

func WithIOWorkerVertexIterator[K any](it iter.Seq[IOWorkerVertex[K]]) IOGraphOption[K] {
	return func(configure *IOGraph[K]) {
		err := configure.AddIOWorkerVertices(it)
		if err != nil {
			slog.Warn("failed to add IOWorkerVertex iterator", "error", err)
		}
	}
}

type IOGraphOption[K any] func(*IOGraph[K])

func New[K any](opt ...IOGraphOption[K]) *IOGraph[K] {
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
	file, _ := os.Create(filepath) // #nosec G304
	return draw.DOT(sg, file)
}

func (sg *IOGraph[K]) AdjancyVertices() (map[string]map[string]IOWorkerVertex[K], error) {
	return sg.NeighborVertices(sg.AdjacencyMap)
}

func (sg *IOGraph[K]) PredecessorVertices() (map[string]map[string]IOWorkerVertex[K], error) {
	return sg.NeighborVertices(sg.PredecessorMap)
}

func (sg *IOGraph[K]) NeighborVertices(orientedNeighborSearch NeighbourSearchFunc) (map[string]map[string]IOWorkerVertex[K], error) {
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

func (sg *IOGraph[K]) IterChildlessVertex() iter.Seq[IOWorkerVertex[K]] {
	return sg.iterOrientedNeighborlessVertex(sg.AdjacencyMap)
}

func (sg *IOGraph[K]) IterParentlessVertex() iter.Seq[IOWorkerVertex[K]] {
	return sg.iterOrientedNeighborlessVertex(sg.PredecessorMap)
}

func (sg *IOGraph[K]) iterOrientedNeighborlessVertex(orientedNeighborSearch NeighbourSearchFunc) iter.Seq[IOWorkerVertex[K]] {
	return func(yield func(IOWorkerVertex[K]) bool) {
		neighborMap, err := orientedNeighborSearch()
		if err != nil {
			panic(err)
		}
		for vertexID, neighbors := range neighborMap {
			if len(neighbors) == 0 {
				vertex, err := sg.Vertex(vertexID)
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

func (sg *IOGraph[K]) AddIOWorkerVertex(newVertex IOWorkerVertex[K]) error {
	if sg.isRunning {
		return ErrIsRunning
	}
	err := sg.AddVertex(newVertex)
	if err != nil {
		slog.Error("Error adding vertex", "vertex", newVertex.GetName(), "error", err)

		return err
	}
	for _, p := range newVertex.GetParents() {
		err := sg.AddEdge(p, newVertex.GetName())
		if err != nil {
			slog.Error("Error adding edge", "from", p, "to", newVertex.GetName(), "error", err)
			return err
		}
	}
	return nil
}

func (sg *IOGraph[K]) AddIOWorkerVertices(vertices iter.Seq[IOWorkerVertex[K]]) error {
	if sg.isRunning {
		return ErrIsRunning
	}
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

func (sg *IOGraph[K]) Close() error {
	sg.innerCancelFn()
	sg.isRunning = false
	return nil
}
