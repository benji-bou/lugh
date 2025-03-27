package base64

import (
	"context"
	"encoding/base64"
	"fmt"

	"github.com/benji-bou/lugh/core/graph"
)

func Base64Decode() graph.WorkerFunc[[]byte] {
	return func(ctx context.Context, input []byte, yield func(elem []byte) error) error {
		res, err := base64.RawStdEncoding.DecodeString(string(input))
		if err != nil {
			return fmt.Errorf("unable to decode from base64: %w", err)
		}
		return yield(res)
	}
}
