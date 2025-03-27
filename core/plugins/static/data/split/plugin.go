package split

import (
	"context"
	"strings"

	"github.com/benji-bou/lugh/core/graph"
)

func Worker(separator string) graph.WorkerFunc[[]byte] {
	return func(ctx context.Context, input []byte, yield func(elem []byte) error) error {
		for _, elem := range strings.Split(string(input), separator) {
			if err := yield([]byte(elem)); err != nil {
				return err
			}
		}
		return nil
	}
}
