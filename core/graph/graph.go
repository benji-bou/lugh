package graph

import (
	"context"
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
		for secVertex := range it {
			configure.AddIOWorkerVertex(secVertex)
		}
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
	return sg.NeighborVertices(sg.AdjacencyMap)
}

func (sg *IOGraph[K]) NeighborVertices(orientedNeighborSearch func() (map[string]map[string]graph.Edge[string], error)) (map[string]map[string]IOWorkerVertex[K], error) {
	verticesMaps, err := orientedNeighborSearch()
	if err != nil {
		return nil, err
	}
	res := make(map[string]map[string]IOWorkerVertex[K], len(verticesMaps))
	for currentVertexHash, neighborMap := range verticesMaps {
		res[currentVertexHash] = make(map[string]IOWorkerVertex[K], len(neighborMap))
		for neighborName, _ := range neighborMap {
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

func (sg *IOGraph[K]) AddIOWorkerVertex(newVertex IOWorkerVertex[K]) error {
	err := sg.AddVertex(newVertex)
	if err != nil {
		return err
	}
	for _, p := range newVertex.GetParents() {
		err := sg.AddEdge(p, newVertex.GetName())
		if err != nil {
			return err
		}
	}
	return nil
}

func (sg *IOGraph[K]) MergeVertexOutput(vertices iter.Seq[IOWorkerVertex[K]]) <-chan K {
	resC := helper.Map(vertices, func(vertex IOWorkerVertex[K]) <-chan K {
		return vertex.Output()
	})
	return chantools.Merge(slices.Collect(resC)...)

}
func (sg *IOGraph[K]) Initialize(ctx context.Context) error {

	parentsMap, err := sg.PredecessorVertices()
	if err != nil {
		return err
	}
	for currentVertexHash, parentVertex := range parentsMap {
		currentVertex, err := sg.Vertex(currentVertexHash)
		if err != nil {
			return err
		}
		currentVertex.SetInput(sg.MergeVertexOutput(maps.Values(parentVertex)))
	}
	return nil
}

func (sg *IOGraph[K]) Run(ctx context.Context) <-chan error {
	vertexMap, _ := sg.AdjacencyMap()
	errorsOutputC := make([]<-chan error, 0, len(vertexMap))
	for vertexHash := range vertexMap {
		vertex, err := sg.Vertex(vertexHash)
		if err != nil {
			slog.Error("Error while start running Io Graph: ", "error", err, "vertexHash", vertexHash)
			continue
		}
		errorsOutputC = append(errorsOutputC, vertex.Run(ctx))
	}
	return chantools.Merge(errorsOutputC...)
}

//TODO: Implement in other way this is not the concern of the graph to add output. should be done outside
// func (sg *IOGraph[K]) ChanOutputGraph(output chan<- *pluginctl.DataStream) error {

// 	rawOutputPlugin := plugin.NewRawOutputPlugin(plugin.WithChannel(output))
// 	outputVertex := SecVertex{name: fmt.Sprintf("datastream-chan-output-%s", uuid.NewString()), plugin: rawOutputPlugin, parents: slices.Collect(helper.Map(sg.IterChildlessVertex(), func(elem SecVertexer) string { return elem.GetName() }))}
// 	err := sg.AddSecVertexer(outputVertex)
// 	if err != nil {
// 		return fmt.Errorf("set chan ouptgraph failed: %w", err)
// 	}
// 	return nil
// }
