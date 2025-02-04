package graph_test

import (
	"context"
	"slices"
	"testing"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/graph/graphtest"
	"github.com/benji-bou/lugh/helper"
)

type testConfig struct {
	name  string
	input [][]string
}

func TestMergeVertexOutput(t *testing.T) {
	testCases := []testConfig{
		{"empty input", [][]string{}},
		{"single element input", [][]string{{"element1"}}},
		{"1D multiple elements input", [][]string{{"element1"}, {"element2"}, {"element3"}}},
		{"2D multiple elements input", [][]string{{"element1a", "element1b", "element1c"}, {"element2a", "element2b", "element2c"}, {"element3a", "element3b", "element3c"}}},
	}

	g := graph.NewIO[string]()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := graph.NewContext(context.Background())
			workers, run := graphtest.GenerateWorkerProducers(ctx, tc.input...)
			result := g.MergeVertexOutput(slices.Values(workers))
			run()
			resA := make([]string, 0, len(tc.input))
			ctx.Synchronize()
			for r := range result {
				resA = append(resA, r)
			}
			expected := slices.Sorted(helper.Flatten(tc.input))
			slices.Sort(resA)
			if !slices.Equal(resA, expected) {
				t.Errorf("MergeVertexOutput = %s; want %s", resA, tc.input)
			}
		})
	}
}

func TestGraphAsWorker(t *testing.T) {
	input := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	tableTestCases := []graphtest.WorkerConfigTest[int]{
		{
			Name:           "linear worker chain",
			DataTest:       input,
			ExpectedOutput: input,
			IO:             graphtest.GraphWorker[int](graphtest.SingleForward), Assert: graphtest.AssertDefaultEqual[int],
		},
		{
			Name:           "linear worker chain mult11 and odd",
			DataTest:       input,
			ExpectedOutput: []int{11, 33, 55, 77, 99},
			IO:             graphtest.GraphWorker(graphtest.LinearWorkerChainMult2),
			Assert:         graphtest.AssertDefaultEqual[int],
		},
	}

	for _, ttc := range tableTestCases {
		graphtest.TestWorkerChain(t, ttc)
	}
}
