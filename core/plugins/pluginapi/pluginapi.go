package pluginapi

import "context"

type IOPluginable interface {
	Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error)
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}
