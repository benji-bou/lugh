package static

import "github.com/benji-bou/lugh/core/graph"

type EmptyPlugin struct {
}

func (spp EmptyPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (spp EmptyPlugin) Config(config []byte) error {
	return nil
}

func (spp EmptyPlugin) Run(context graph.Context, input <-chan []byte) <-chan []byte {
	return nil, nil
}
