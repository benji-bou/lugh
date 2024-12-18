package template

import (
	"log"
	"os"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"gopkg.in/yaml.v3"
)

type NamedStage struct {
	name string
}

type Stage struct {
	NamedStage
	graph.DefaultIOWorkerVertex[[]byte]
	Parents    []string               `yaml:"parents"`
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     map[string]any         `yaml:"config"`
	Pipe       []map[string]yaml.Node `yaml:"pipe"`
}

func (nt NamedStage) GetName() string {
	return nt.name

}
func (st Stage) GetParents() []string {
	return st.Parents
}

func (st Stage) LoadPlugin(name string) graph.IOWorkerVertex[[]byte] {
	defaultPath := os.Getenv("SP_PLUGIN_DEFAULT_PLUGIN_PATH")
	if st.PluginPath == "" {
		st.PluginPath = defaultPath
	}
	secplugin, err := grpc.NewPlugin(st.Plugin, grpc.WithPath(st.PluginPath)).Connect()
	if err != nil {
		log.Fatal("Failed to load plugin: ", err)
		return nil
	}
	return graph.NewDefaultIOWorkerVertex[[]byte](name, st.Parents, plugins.NewIOWorkerPlugin(secplugin))
}
