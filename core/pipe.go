package core

import "github.com/benji-bou/SecPipeline/pluginctl"

type Pipeable interface {
	Pipe() (<-chan []byte, <-chan error)
}

type DefaultPipe struct {
	from <-chan []byte
	to   pluginctl.SecPipelinePluginable
}

func NewPipe(from <-chan []byte, to pluginctl.SecPipelinePluginable) Pipeable {
	return DefaultPipe{from: from, to: to}
}

func (dc DefaultPipe) Pipe() (<-chan []byte, <-chan error) {
	return dc.to.Run(dc.from)
}
