package template

import (
	"fmt"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/load"
)

type Stage struct {
	PluginPath string   `yaml:"pluginPath"`
	Plugin     string   `yaml:"plugin"`
	Config     any      `yaml:"config"`
	Parents    []string `yaml:"parents"`
}

func (st Stage) LoadPlugin(name string, templateConfig TemplateConfig) (graph.IOWorkerVertex[[]byte], error) {
	if st.PluginPath == "" {
		st.PluginPath = templateConfig.PluginPath
	}
	secplugin, err := load.Worker(st.Plugin, st.PluginPath, st.Config)
	if err != nil {
		return graph.IOWorkerVertex[[]byte]{}, fmt.Errorf("stage %s loading plugin %s: %w", name, st.Plugin, err)
	}
	return graph.NewIOWorkerVertex(name, st.Parents, secplugin), nil
}
