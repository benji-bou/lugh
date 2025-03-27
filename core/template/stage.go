package template

import (
	"errors"
	"fmt"
	"log/slog"
	"maps"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/load"
)

var ErrEmptyIncludePath = errors.New("include path is empty")

type Stage struct {
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     any                    `yaml:"config"`
	Parents    []string               `yaml:"parents"`
	Include    string                 `yaml:"include"`
	Variables  map[string]interface{} `yaml:"variables"`
}

func (st Stage) LoadPlugin(name string, defaultPluginsPath string, variables map[string]interface{}) (graph.IOWorkerVertex[[]byte], error) {
	if st.PluginPath == "" {
		st.PluginPath = defaultPluginsPath
	}
	graphWorker, err := st.IncludeGraph(name, variables)
	if graphWorker != nil && err == nil {
		return graph.NewIOWorkerVertex(name, st.Parents, graphWorker), nil
	} else if !errors.Is(err, ErrEmptyIncludePath) {
		return graph.IOWorkerVertex[[]byte]{}, err
	}

	secplugin, err := load.Worker(st.Plugin, st.PluginPath, st.Config)
	if err != nil {
		return graph.IOWorkerVertex[[]byte]{}, fmt.Errorf("stage %s failed to load plugin %s: %w", name, st.Plugin, err)
	}
	return graph.NewIOWorkerVertex(name, st.Parents, secplugin), nil
}

func (st Stage) IncludeGraph(parent string, variables map[string]interface{}) (graph.IOWorker[[]byte], error) {
	if st.Include == "" {
		return nil, ErrEmptyIncludePath
	}
	localVariables := maps.Clone(variables)

	localVariables["is_included"] = true
	localVariables["parent"] = parent
	if st.Variables != nil {
		maps.Copy(localVariables, st.Variables)
	}
	tpl, err := NewFile(st.Include, localVariables)
	if err != nil {
		slog.Error("failed to start template", "error", err)
		return nil, fmt.Errorf("include template %s failed: %w", st.Include, err)
	}
	vertices, err := tpl.WorkerVertexIterator(st.PluginPath)
	if err != nil {
		return nil, fmt.Errorf("include template %s failed to load vertices: %w", st.Include, err)
	}
	g := graph.NewIO(graph.WithVertices(vertices))
	return g, nil
}
