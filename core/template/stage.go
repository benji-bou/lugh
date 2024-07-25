package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugin"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"gopkg.in/yaml.v3"
)

type Stage struct {
	Parents    []string               `yaml:"parents"`
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     map[string]any         `yaml:"config"`
	Pipe       []map[string]yaml.Node `yaml:"pipe"`
}

func (st Stage) AddSecVertex(g graph.SecGraph, name string) error {
	vertex, err := st.GetSecVertex(name)
	if err != nil {
		return fmt.Errorf("failed to get SecVertex %s  because: %w", name, err)
	}
	err = g.AddVertex(vertex)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		return fmt.Errorf("couldn't add SecVertex %s to graph because: %w", name, err)
	}
	return nil
}

func (st Stage) AddEdges(g graph.SecGraph, name string, rootVertex graph.SecVertex) error {
	if len(st.Parents) == 0 {
		err := g.AddEdge(rootVertex.Name, name)
		if err != nil {
			return fmt.Errorf("failed to link rootVertex to  %s  because: %w", name, err)
		}
	}
	for _, p := range st.Parents {
		err := g.AddEdge(p, name)
		if err != nil && err != graph.ErrEdgeAlreadyExists {
			return fmt.Errorf("failed to link %s to  %s because: %w", p, name, err)
		}
	}
	return nil
}

func (st Stage) configPlugin(name string, plugin pluginctl.SecPipelinePluginable) error {
	if st.Config != nil && len(st.Config) > 0 {
		cJson, err := json.Marshal(st.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal config %s with label %s because: %w", st.Plugin, name, err)
		}
		err = plugin.Config(cJson)
		if err != nil {
			return fmt.Errorf("failed to configure %s with label %s because: %w", st.Plugin, name, err)
		}
	}
	return nil
}

func (st Stage) createPlugin(name string, path string) (pluginctl.SecPipelinePluginable, error) {
	plugin, err := pluginctl.NewPlugin(st.Plugin, pluginctl.WithPath(path)).Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to build plugin %s with label %s because: %w", st.Plugin, name, err)
	}
	pipeCount := len(st.Pipe)
	if st.Pipe == nil || pipeCount == 0 {
		return plugin, nil
	}
	pipe, err := st.buildPipes(name)
	if err != nil {
		return nil, err
	}
	return NewSecPipePlugin(pipe, plugin), nil
}

func (st Stage) GetSecVertex(name string) (SecVertex, error) {
	defaultPath := "/Users/benjamin/Private/Projects/SecPipeline/core/secpipeline/bin/plugins"
	if st.PluginPath != "" {
		defaultPath = st.PluginPath
	}
	plugin, err := st.createPlugin(name, defaultPath)
	if err != nil {
		return SecVertex{}, err
	}
	if err := st.configPlugin(name, plugin); err != nil {
		return SecVertex{}, err
	}
	return SecVertex{Name: name, plugin: plugin}, nil
}

func (st Stage) buildPipes(name string) (Pipeable, error) {
	pipeCount := len(st.Pipe)
	pipes := make([]Pipeable, 0, pipeCount)
	for _, pipeMapConfig := range st.Pipe {
		for pipeName, pipeConfig := range pipeMapConfig {
			pipeable, err := st.buildPipe(pipeName, pipeConfig)
			if err != nil {

				slog.Error("failed to build pipe", "error", err)
				return nil, fmt.Errorf("failed to build pipe %s for %s with label %s because: %w", pipeName, st.Plugin, name, err)
			}
			pipes = append(pipes, pipeable)
		}
	}
	return NewChainedPipe(pipes...), nil
}

func (st Stage) buildPipe(pipeName string, config yaml.Node) (plugin.Pipeable, error) {
	switch pipeName {
	case "goTemplate":
		configGoTpl := GoTemplateConfig{}
		err := config.Decode(&configGoTpl)
		if err != nil {
			return nil, fmt.Errorf("failed to build goTemplate pipe because: %w", err)
		}
		return NewGoTemplatePipeWithConfig(configGoTpl), nil
	case "base64":
		return NewBase64Decoder(), nil
	case "regex":
		configRegex := struct {
			Pattern string `yaml:"pattern"`
			Select  int    `yaml:"select"`
		}{}

		err := config.Decode(&configRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex pipe because: %w", err)
		}

		return NewRegexpPipe(configRegex.Pattern, configRegex.Select), nil
	case "insert":
		configInsert := struct {
			Content string `yaml:"content"`
		}{}
		err := config.Decode(&configInsert)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex pipe because: %w", err)
		}
		return NewInsertStringPipe(configInsert.Content), nil
	case "split":
		configSplit := struct {
			Content string `yaml:"sep"`
		}{}
		err := config.Decode(&configSplit)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex pipe because: %w", err)
		}
		return NewSplitPipe(configSplit.Content), nil
	default:
		return NewEmptyPipe(), errors.New("failed to build template. Template unknown")
	}
}
