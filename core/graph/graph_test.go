package graph

import (
	"context"
	"slices"
	"testing"

	"github.com/benji-bou/diwo"
	"github.com/benji-bou/lugh/helper"
)

type testConfig struct {
	name  string
	input [][]string
}

func generateWorkerProducer[K any](ctx SyncContext, elements ...K) (IOWorker[K], func()) {
	worker := NewIOWorkerFromWorker(ForwardWorkerTestable[K]{})
	inputC := diwo.FromSlice(elements)
	worker.SetInput(inputC)
	return worker, func() {
		worker.Run(ctx)
	}
}

func runWorkerConfig(tc testConfig, ctx SyncContext) ([]IOWorker[string], func()) {

	workers := make([]IOWorker[string], 0, len(tc.input))
	runs := make([]func(), 0, len(tc.input))
	for i := 0; i < len(tc.input); i++ {
		w, run := generateWorkerProducer(ctx, tc.input[i]...)
		workers = append(workers, w)
		runs = append(runs, run)
	}
	return workers, func() {
		for _, r := range runs {
			r()
		}
	}
}
func TestMergeVertexOutput(t *testing.T) {

	testCases := []testConfig{
		{"empty input", [][]string{}},
		{"single element input", [][]string{{"element1"}}},
		{"1D multiple elements input", [][]string{{"element1"}, {"element2"}, {"element3"}}},
		{"2D multiple elements input", [][]string{{"element1a", "element1b", "element1c"}, {"element2a", "element2b", "element2c"}, {"element3a", "element3b", "element3c"}}}}

	g := New[string]()
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := NewContext(context.Background())
			workers, run := runWorkerConfig(tc, ctx)
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
