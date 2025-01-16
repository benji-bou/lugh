package graph

import (
	"errors"
	"fmt"
	"iter"
	"log/slog"
	"maps"
	"os"
	"slices"

	"github.com/benji-bou/lugh/helper/collections/set"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
)

type Graph[K comparable, T any] struct {
	graph.Graph[K, T]
}

func New[K comparable, T any](hash graph.Hash[K, T], options ...func(*graph.Traits)) Graph[K, T] {
	return Graph[K, T]{Graph: graph.New(hash, options...)}
}

func (g Graph[K, T]) DrawGraph(filepath string) error {
	file, err := os.Create(filepath) // #nosec G304
	if err != nil {
		return fmt.Errorf("failed to draw graph: %w", err)
	}
	return draw.DOT(g, file)
}

func (g Graph[K, T]) CloneFromEdge(edge ...graph.Edge[K]) (Graph[K, T], error) {
	if len(edge) == 0 {
		g, err := g.Clone()
		if err != nil {
			return Graph[K, T]{}, fmt.Errorf("failed to cloned from edges: %w", err)
		}
		return Graph[K, T]{Graph: g}, nil
	}
	newG := Graph[K, T]{Graph: graph.NewLike(g.Graph)}
	for _, e := range edge {
		src, err := g.Vertex(e.Source)
		if err != nil {
			return Graph[K, T]{}, fmt.Errorf("failed to get vertex %v: %w", e.Source, err)
		}
		err = newG.AddVertex(src)
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			return Graph[K, T]{}, fmt.Errorf("failed to insert vertex %v:  %w", e.Source, err)
		}
		dst, err := g.Vertex(e.Target)
		if err != nil {
			return Graph[K, T]{}, fmt.Errorf("failed to get vertex %v: %w", e.Target, err)
		}
		err = newG.AddVertex(dst)
		if err != nil && !errors.Is(err, graph.ErrVertexAlreadyExists) {
			return Graph[K, T]{}, fmt.Errorf("failed to insert vertex %v:   %w", e.Target, err)
		}
		err = newG.AddEdge(e.Source, e.Target)
		if err != nil && !errors.Is(err, graph.ErrEdgeAlreadyExists) {
			return Graph[K, T]{}, fmt.Errorf("failed to insert edge %v-%v:  %w", e.Source, e.Target, err)
		}
	}
	return newG, nil
}

// Search Methods

func (g Graph[K, T]) Vertices() (map[K]T, error) {
	neighbors, err := g.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("vertices failed to get all: %w", err)
	}
	res := make(map[K]T)
	for k := range neighbors {
		vertex, err := g.Vertex(k)
		if err != nil {
			return nil, fmt.Errorf("vertices failed to get vertex %v: %w", k, err)
		}
		res[k] = vertex
	}
	return res, nil
}

func (g Graph[K, T]) VertexEdges(hash K) ([]graph.Edge[K], error) {
	allEdges, err := g.UndirectedNeighbors()
	if err != nil {
		return nil, fmt.Errorf("vertexedges failed to get all UndirectedNeighbors: %w", err)
	}
	neighbors, ok := allEdges[hash]
	if !ok {
		return nil, fmt.Errorf("vertexedges failed to find vertex %v: %w", hash, graph.ErrVertexNotFound)
	}
	return slices.Collect(maps.Values(neighbors)), nil
}

type NeighbourSearchFunc[K comparable] func() (map[K]map[K]graph.Edge[K], error)

func (g Graph[K, T]) UndirectedNeighbors() (map[K]map[K]graph.Edge[K], error) {
	predecessors, err := g.PredecessorMap()
	if err != nil {
		return nil, fmt.Errorf("undirectedneighbors failed to get predecessors: %w", err)
	}
	adjancies, err := g.AdjacencyMap()
	if err != nil {
		return nil, fmt.Errorf("undirectedneighbors failed to get adjacencies: %w", err)
	}
	for hash, neighbors := range adjancies {
		if predecessorsNeighbors, ok := predecessors[hash]; ok {
			maps.Insert(neighbors, maps.All(predecessorsNeighbors))
		}
	}
	return adjancies, nil
}

func (g Graph[K, T]) iterOrientedNeighborlessVertex(orientedNeighborSearch NeighbourSearchFunc[K]) iter.Seq[T] {
	return func(yield func(T) bool) {
		neighborMap, err := orientedNeighborSearch()
		if err != nil {
			panic(err)
		}
		for vertexID, neighbors := range neighborMap {
			if len(neighbors) == 0 {
				vertex, err := g.Vertex(vertexID)
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

func (g Graph[K, T]) IterChildlessVertex() iter.Seq[T] {
	return g.iterOrientedNeighborlessVertex(g.AdjacencyMap)
}

func (g Graph[K, T]) IterParentlessVertex() iter.Seq[T] {
	return g.iterOrientedNeighborlessVertex(g.PredecessorMap)
}

func (g Graph[K, T]) NeighborVertices(orientedNeighborSearch NeighbourSearchFunc[K]) (map[K]map[K]T, error) {
	verticesMaps, err := orientedNeighborSearch()
	if err != nil {
		return nil, err
	}
	res := make(map[K]map[K]T, len(verticesMaps))
	for currentVertexHash, neighborMap := range verticesMaps {
		res[currentVertexHash] = make(map[K]T, len(neighborMap))
		for neighborName := range neighborMap {
			v, err := g.Graph.Vertex(neighborName)
			if err != nil {
				return nil, err
			}
			res[currentVertexHash][neighborName] = v
		}
	}
	return res, nil
}

func (g Graph[K, T]) AdjancyVertices() (map[K]map[K]T, error) {
	return g.NeighborVertices(g.AdjacencyMap)
}

func (g Graph[K, T]) PredecessorVertices() (map[K]map[K]T, error) {
	return g.NeighborVertices(g.PredecessorMap)
}

func (g Graph[K, T]) Split() ([]Graph[K, T], error) {
	splittedEdges := [][]graph.Edge[K]{}
	verticesNeighborsMap, err := g.UndirectedNeighbors()
	if err != nil {
		return nil, fmt.Errorf("failed to get adjencyMap %w", err)
	}
	for vertexHash := range verticesNeighborsMap {
		listEdges := make([]graph.Edge[K], 0)
		stack := set.New[K](vertexHash)

		for current := range stack.Next() {
			for neighborHash, neighborEdge := range maps.All(verticesNeighborsMap[current]) {
				if ok := stack.Contains(neighborHash); !ok {
					listEdges = append(listEdges, neighborEdge)
					stack.Add(neighborHash)
				}
			}
			delete(verticesNeighborsMap, current)
		}
		splittedEdges = append(splittedEdges, listEdges)
	}
	res := make([]Graph[K, T], 0, len(splittedEdges))
	for _, edges := range splittedEdges {
		newG, err := g.CloneFromEdge(edges...)
		if err != nil {
			return nil, fmt.Errorf("failed to split graph %w", err)
		}
		res = append(res, newG)
	}
	return res, nil
}

type GraphSelfDescribe[K comparable, T VertexSelfDescribe[K]] struct {
	Graph[K, T]
}

func NewSelfDescribed[K comparable, T VertexSelfDescribe[K]](
	hash graph.Hash[K, T],
	options ...func(*graph.Traits),
) *GraphSelfDescribe[K, T] {
	return &GraphSelfDescribe[K, T]{Graph: Graph[K, T]{Graph: graph.New(hash, options...)}}
}

// Inserts Methods

func (g *GraphSelfDescribe[K, T]) AddSelfDescribeVertex(newVertex T) error {
	err := g.AddVertex(newVertex)
	if err != nil {
		slog.Error("Error adding vertex", "vertex", newVertex.GetName(), "error", err)

		return err
	}
	for _, p := range newVertex.GetParents() {
		err := g.AddEdge(p, newVertex.GetName())
		if err != nil {
			slog.Error("Error adding edge", "from", p, "to", newVertex.GetName(), "error", err)
			return err
		}
	}
	return nil
}

func (g *GraphSelfDescribe[K, T]) AddVertices(vertices iter.Seq[T]) error {
	for newVertex := range vertices {
		err := g.AddVertex(newVertex)
		if err != nil {
			slog.Error("Error adding vertex", "vertex", newVertex.GetName(), "error", err)

			return err
		}
	}
	for newVertex := range vertices {
		for _, p := range newVertex.GetParents() {
			err := g.AddEdge(p, newVertex.GetName())
			if err != nil {
				slog.Error("Error adding edge", "from", p, "to", newVertex.GetName(), "error", err)
				return err
			}
		}
	}
	return nil
}
