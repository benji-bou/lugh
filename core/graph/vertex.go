package graph

import "github.com/benji-bou/SecPipeline/pluginctl"

type SecVertex struct {
	Name   string
	plugin pluginctl.SecPipelinePluginable
}
