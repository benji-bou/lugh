package static

import "context"

type EmptyPlugin struct {
}

func (spp EmptyPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (spp EmptyPlugin) Config(config []byte) error {
	return nil
}

func (spp EmptyPlugin) Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	return nil, nil
}
