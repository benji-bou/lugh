package graphtest

import (
	"github.com/benji-bou/lugh/core/graph"
	gr "github.com/dominikbraun/graph"
)

type TestVertex struct {
	Name    string
	Parents []string
}

type GraphTestCases struct {
	SubVertices [][]TestVertex
	Name        string
	NumSubgraph int
}

func (tv TestVertex) GetName() string {
	return tv.Name
}

func (tv TestVertex) GetParents() []string {
	return tv.Parents
}

func NewGraphTest() *graph.GraphSelfDescribe[string, TestVertex] {
	return graph.NewSelfDescribed[string, TestVertex](func(elem TestVertex) string { return elem.Name }, gr.Directed())
}

func GenerateGraphTestCases() []GraphTestCases {
	return []GraphTestCases{
		{
			SubVertices: [][]TestVertex{
				{
					{"1_1-a", []string{}},
					{"1_2-a", []string{"1_1-a"}},
					{"1_2-b", []string{"1_1-a"}},
				},
				{
					{"2_1-a", []string{}},
					{"2_2-a", []string{"2_1-a"}},
					{"2_2-b", []string{"2_1-a"}},
				},
			},
			Name:        "2_graph_bin_leaf",
			NumSubgraph: 2,
		},
		{
			SubVertices: [][]TestVertex{
				{
					{"1_1-a", []string{}},
					{"1_2-a", []string{"1_1-a"}},
					{"1_2-b", []string{"1_1-a"}},
				},
				{
					{"2_1-a", []string{}},
					{"2_2-a", []string{"2_1-a"}},
					{"2_2-b", []string{"2_1-a"}},
				},
				{
					{"3_1-a", []string{}},
					{"3_2-a", []string{"3_1-a"}},
					{"3_2-b", []string{"3_1-a"}},
				},
			},
			Name:        "3_graph_bin_leaf",
			NumSubgraph: 3,
		},
		{
			SubVertices: [][]TestVertex{
				{
					{"1_1-a", []string{}},
					{"1_2-a", []string{"1_1-a"}},
					{"1_2-b", []string{"1_1-a"}},
				},
				{
					{"2_1-a", []string{}},
					{"2_2-a", []string{"2_1-a", "1_2-a"}},
					{"2_2-b", []string{"2_1-a"}},
				},
			},
			Name:        "1_graph_bin_leaf_merge_parents",
			NumSubgraph: 1,
		},
		{
			SubVertices: [][]TestVertex{
				{
					{"1_1-a", []string{}},
					{"1_2-a", []string{"1_1-a"}},
					{"1_2-b", []string{"1_1-a"}},
					{"2_1-a", []string{}},
					{"2_2-a", []string{"2_1-a", "1_2-a"}},
					{"2_2-b", []string{"2_1-a"}},
				},
				{
					{"3_1-a", []string{}},
					{"3_2-a", []string{"3_1-a"}},
					{"3_2-b", []string{"3_1-a"}},
				},
			},
			Name:        "2_graph_bin_leaf_merge_parents",
			NumSubgraph: 2,
		},
		{
			SubVertices: [][]TestVertex{
				{
					{"1_1-a", []string{}},
					{"1_2-a", []string{"1_1-a"}},
					{"1_2-b", []string{"1_1-a"}},
				},
				{
					{"2_1-a", []string{}},
					{"2_2-a", []string{"2_1-a"}},
					{"2_2-b", []string{"2_1-a"}},
					{"2_2-c", []string{"2_1-a"}},
				},
				{
					{"3_1-a", []string{}},
					{"3_2-a", []string{"3_1-a"}},
					{"3_2-b", []string{"3_1-a"}},
				},
			},
			Name:        "3_graph_mult_leaf",
			NumSubgraph: 3,
		},
	}
}
