package template

import (
	"log"
	"log/slog"
	"maps"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins"
)

type Stage struct {
	Parents    []string               `yaml:"parents"`
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     any                    `yaml:"config"`
	Include    string                 `yaml:"include"`
	Variables  map[string]interface{} `yaml:"variables"`
}

func (st Stage) LoadPlugin(name string, defaultPluginsPath string, variables map[string]interface{}) graph.IOWorkerVertex[[]byte] {
	if st.PluginPath == "" {
		st.PluginPath = defaultPluginsPath
	}
	if graphWorker := st.IncludeGraph(name, variables); graphWorker != nil {
		return graph.NewDefaultIOWorkerVertex(name, st.Parents, graphWorker)
	}

	secplugin, err := plugins.LoadPlugin(st.Plugin, st.PluginPath, st.Config)
	if err != nil {
		log.Fatalf("load plugin: %s, %v", name, err)
		return nil
	}
	return graph.NewDefaultIOWorkerVertex(name, st.Parents, secplugin)
}

func (st Stage) IncludeGraph(parent string, variables map[string]interface{}) graph.IOWorker[[]byte] {
	if st.Include == "" {
		return nil
	}
	localVariables := maps.Clone(variables)

	localVariables["is_included"] = true
	localVariables["parent"] = parent
	maps.Copy(st.Variables, localVariables)
	tpl, err := NewFile(st.Include, localVariables)
	if err != nil {
		slog.Error("failed to start template", "error", err)
		return nil
	}
	g := graph.NewIO(graph.WithVertices(tpl.WorkerVertexIterator(st.PluginPath)))
	return g
}
