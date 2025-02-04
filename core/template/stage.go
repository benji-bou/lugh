package template

import (
	"log"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins"
)

type Stage struct {
	Parents    []string `yaml:"parents"`
	PluginPath string   `yaml:"pluginPath"`
	Plugin     string   `yaml:"plugin"`
	Config     any      `yaml:"config"`
}

func (st Stage) LoadPlugin(name string, defaultPluginsPath string) graph.IOWorkerVertex[[]byte] {
	if st.PluginPath == "" {
		st.PluginPath = defaultPluginsPath
	}
	secplugin, err := plugins.LoadPlugin(st.Plugin, st.PluginPath, st.Config)
	if err != nil {
		log.Fatalf("load plugin: %s, %v", name, err)
		return nil
	}
	return graph.NewDefaultIOWorkerVertex[[]byte](name, st.Parents, secplugin)
}
