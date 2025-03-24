package forward

import (
	"context"
	"log/slog"

	"github.com/benji-bou/lugh/core/graph"
)

func ForwardWorker[K any]() graph.IOWorker[K] {
	return graph.NewIOWorkerFromWorker(graph.WorkerFunc[K](func(_ context.Context, input K, yield func(elem K) error) error {
		slog.Debug("ForwardWorker reveived data", "data", input)
		return yield(input)
	}))
}
