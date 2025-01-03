package template

import (
	"log"
	"os"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins"
	"gopkg.in/yaml.v3"
)

type NamedStage struct {
	name string
}

type Stage struct {
	NamedStage
	Parents    []string               `yaml:"parents"`
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     any                    `yaml:"config"`
	Pipe       []map[string]yaml.Node `yaml:"pipe"`
}

func (st Stage) LoadPlugin(name string) graph.IOWorkerVertex[[]byte] {
	defaultPath := os.Getenv("SP_PLUGIN_DEFAULT_PLUGIN_PATH")
	if st.PluginPath == "" {
		st.PluginPath = defaultPath
	}
	secplugin, err := plugins.LoadPlugin(st.Plugin, st.PluginPath, st.Config)
	if err != nil {
		log.Fatal("Failed to load plugin: ", err)
		return nil
	}
	return graph.NewDefaultIOWorkerVertex[[]byte](name, st.Parents, secplugin)
}
