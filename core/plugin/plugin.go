package plugin

import (
	"context"

	"github.com/benji-bou/SecPipeline/pluginctl"
)

type EmptySecPlugin struct {
}

func (spp EmptySecPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (spp EmptySecPlugin) Config(config []byte) error {
	return nil
}

func (spp EmptySecPlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return nil, nil
}

type SecPipePlugin struct {
	pipe   Pipeable
	plugin pluginctl.SecPipelinePluginable
}

func NewSecPipePlugin(pipe Pipeable, plugin pluginctl.SecPipelinePluginable) pluginctl.SecPipelinePluginable {
	return SecPipePlugin{pipe: pipe, plugin: plugin}
}
func (spp SecPipePlugin) GetInputSchema() ([]byte, error) {
	return spp.plugin.GetInputSchema()
}

func (spp SecPipePlugin) Config(config []byte) error {
	return spp.plugin.Config(config)
}

func (spp SecPipePlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	pipeOutputC, _ := spp.pipe.Pipe(ctx, input)
	return spp.plugin.Run(ctx, pipeOutputC)
}
