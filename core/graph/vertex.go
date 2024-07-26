package graph

import (
	"github.com/benji-bou/SecPipeline/core/plugin"
	"github.com/benji-bou/SecPipeline/core/template"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
)

type SecVertex struct {
	Name   string
	plugin pluginctl.SecPluginable
}

type SecVertexOption = helper.OptionError[SecVertex]

func VertexFromStage(st template.Stage) SecVertexOption {
	return func(configure *SecVertex) error {
		defaultPath := "/Users/benjamin/Private/Projects/SecPipeline/core/secpipeline/bin/plugins"
		if st.PluginPath == "" {
			st.PluginPath = defaultPath
		}
		plugin, err := plugin.NewSecPipePlugin(plugin.WithStage(st))
		if err != nil {
			return err
		}
		configure.plugin = plugin
		return nil
	}
}

func VertexWithPlugin(plugin pluginctl.SecPluginable) SecVertexOption {
	return func(configure *SecVertex) error {
		configure.plugin = plugin
		return nil
	}
}

func NewSecVertex(name string, opt ...SecVertexOption) (SecVertex, error) {
	return helper.ConfigureWithError(SecVertex{Name: name}, opt...)
}
