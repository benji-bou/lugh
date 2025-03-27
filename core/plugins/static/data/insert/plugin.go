package insert

import (
	"context"

	"github.com/benji-bou/lugh/core/graph"
)

func Worker(insert string) graph.WorkerFunc[[]byte] {
	return func(ctx context.Context, input []byte, yield func(elem []byte) error) error {
		input = append(input, []byte(insert)...)
		return yield(input)
	}
}
