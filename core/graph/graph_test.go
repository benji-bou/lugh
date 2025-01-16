package graph_test

import (
	"slices"
	"testing"

	"github.com/benji-bou/lugh/core/graph/graphtest"
	gr "github.com/dominikbraun/graph"
	"github.com/google/go-cmp/cmp"
)

func TestCloneFromEdges(t *testing.T) {
	// todo
	t.Run("CloneFromEdges", func(_ *testing.T) {})
}

type AddNeighborFunc func(res map[string]map[string]gr.Edge[string], edge gr.Edge[string])

func addNeighbors(res map[string]map[string]gr.Edge[string], edge gr.Edge[string]) {
	addAjancy(res, edge)
	addPredecessor(res, edge)
}

func addAjancy(res map[string]map[string]gr.Edge[string], edge gr.Edge[string]) {
	if _, ok := res[edge.Target]; !ok {
		res[edge.Target] = make(map[string]gr.Edge[string])
	}
	res[edge.Target][edge.Source] = edge
}

func addPredecessor(res map[string]map[string]gr.Edge[string], edge gr.Edge[string]) {
	if _, ok := res[edge.Source]; !ok {
		res[edge.Source] = make(map[string]gr.Edge[string])
	}
	res[edge.Source][edge.Target] = edge
}

func neighborsFromTestVertices(test [][]graphtest.TestVertex, add AddNeighborFunc) map[string]map[string]gr.Edge[string] {
	res := make(map[string]map[string]gr.Edge[string])
	for _, vertices := range test {
		for _, vertex := range vertices {
			for _, parent := range vertex.Parents {
				add(res, gr.Edge[string]{Target: vertex.Name, Source: parent, Properties: gr.EdgeProperties{Attributes: map[string]string{}}})
			}
		}
	}
	return res
}

func TestUndirectedNeighboors(t *testing.T) {
	testCases := graphtest.GenerateGraphTestCases()

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			expected := neighborsFromTestVertices(tc.SubVertices, addNeighbors)
			g := graphtest.NewGraphTest()
			err := g.AddVertices(slices.Values(slices.Concat(tc.SubVertices...)))
			if err != nil {
				t.Errorf("Error adding vertices: %v", err)
				return
			}
			res, err := g.UndirectedNeighbors()
			if err != nil {
				t.Errorf("Error splitting graph: %v", err)
				return
			}
			resultCmp := cmp.Diff(expected, res)
			if resultCmp != "" {
				t.Errorf("Expected neighbors did not match:\n%s", resultCmp)
			}
		})
	}
}

func TestSplit(t *testing.T) {
	testCases := graphtest.GenerateGraphTestCases()
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			g := graphtest.NewGraphTest()
			err := g.AddVertices(slices.Values(slices.Concat(tc.SubVertices...)))
			if err != nil {
				t.Errorf("Error adding vertices: %v", err)
				return
			}
			splittedGraph, err := g.Split()
			if err != nil {
				t.Errorf("Error splitting graph: %v", err)
				return
			}
			if len(splittedGraph) != tc.NumSubgraph {
				t.Errorf("Split Failed. Wrong number of subgraphs Want %d Got %d", tc.NumSubgraph, len(splittedGraph))
				return
			}
		})
	}
}
