package core

import (
	"context"

	"github.com/benji-bou/SecPipeline/pluginctl"
)

type Pipeable interface {
	Pipe(ctx context.Context) (<-chan *pluginctl.DataStream, <-chan error)
}

type DefaultPipe struct {
	from <-chan *pluginctl.DataStream
	to   pluginctl.SecPipelinePluginable
}

func NewPipe(from <-chan *pluginctl.DataStream, to pluginctl.SecPipelinePluginable) Pipeable {
	return DefaultPipe{from: from, to: to}
}

func (dc DefaultPipe) Pipe(ctx context.Context) (<-chan *pluginctl.DataStream, <-chan error) {
	return dc.to.Run(ctx, dc.from)
}
