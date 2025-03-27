package regex

import (
	"context"
	"fmt"
	"regexp"

	"github.com/benji-bou/lugh/core/graph"
)

type Config struct {
	Pattern string `yaml:"pattern"`
	Select  int    `yaml:"select"`
}

func Worker(config Config) (graph.WorkerFunc[[]byte], error) {
	regex, err := regexp.Compile(config.Pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regexp pattern %s: %w", config.Pattern, err)
	}
	return func(ctx context.Context, input []byte, yield func(elem []byte) error) error {
		resAllArr := regex.FindAllSubmatch(input, -1)
		for _, match := range resAllArr {
			var toYield []byte
			switch l := len(match); {
			case l == 0:
				continue
			case config.Select >= l:
				toYield = match[0]
			default:
				toYield = match[config.Select]
			}
			if err := yield(toYield); err != nil {
				return err
			}
		}
		return nil
	}, nil
}
