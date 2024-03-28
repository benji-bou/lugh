package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
	"github.com/dominikbraun/graph"
	"github.com/google/uuid"
	"gopkg.in/yaml.v3"
)

type EmptySecPlugin struct {
}

func (spp EmptySecPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (spp EmptySecPlugin) Config(config []byte) error {
	return nil
}

func (spp EmptySecPlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return nil, nil
}

type SecPipePlugin struct {
	pipe   Pipeable
	plugin pluginctl.SecPipelinePluginable
}

func NewSecPipePlugin(pipe Pipeable, plugin pluginctl.SecPipelinePluginable) pluginctl.SecPipelinePluginable {
	return SecPipePlugin{pipe: pipe, plugin: plugin}
}
func (spp SecPipePlugin) GetInputSchema() ([]byte, error) {
	return spp.plugin.GetInputSchema()
}

func (spp SecPipePlugin) Config(config []byte) error {
	return spp.plugin.Config(config)
}

func (spp SecPipePlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	pipeOutputC, _ := spp.pipe.Pipe(ctx, input)
	return spp.plugin.Run(ctx, pipeOutputC)
}

type SecVertex struct {
	Name   string
	plugin pluginctl.SecPipelinePluginable
}

type SecGraph = graph.Graph[string, SecVertex]

type Template struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Version     string           `yaml:"version" json:"version"`
	Author      string           `yaml:"author" json:"author"`
	Stages      map[string]Stage `yaml:"stages" json:"stages"`
}

func (t *Template) getTemplateGraph() (SecGraph, SecVertex, error) {

	g := graph.New(func(spp SecVertex) string {
		return spp.Name
	}, graph.Directed())
	rootVertex := SecVertex{Name: uuid.NewString(), plugin: EmptySecPlugin{}}
	g.AddVertex(rootVertex)
	for stageName, stage := range t.Stages {
		err := stage.AddSecNode(g, stageName, rootVertex.Name)
		if err != nil {
			return nil, SecVertex{}, fmt.Errorf("failed to get secGraph %w", err)
		}
	}
	return g, rootVertex, nil
}

type Stage struct {
	Parents    []string               `yaml:"parents"`
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     map[string]any         `yaml:"config"`
	Pipe       []map[string]yaml.Node `yaml:"pipe"`
}

func (st Stage) AddSecNode(g SecGraph, name string, rootVertex string) error {
	trait, err := st.GetSecVertex(name)
	if err != nil {
		return fmt.Errorf("failed to add trait to graph, %w", err)
	}
	g.AddVertex(trait)
	if len(st.Parents) == 0 {
		g.AddEdge(rootVertex, name)
	}
	for _, p := range st.Parents {
		g.AddEdge(p, name)
	}
	return nil
}

func (pl Stage) configPlugin(name string, plugin pluginctl.SecPipelinePluginable) error {
	if pl.Config != nil && len(pl.Plugin) > 0 {
		cJson, err := json.Marshal(pl.Config)
		if err != nil {
			return fmt.Errorf("failed to marshal configuration %s with label %s: %w", pl.Plugin, name, err)
		}
		err = plugin.Config(cJson)
		if err != nil {
			return fmt.Errorf("failed to configure %s with label %s: %w", pl.Plugin, name, err)
		}
	}
	return nil
}

func (pl Stage) createPlugin(name string, path string) (pluginctl.SecPipelinePluginable, error) {
	plugin, err := pluginctl.NewPlugin(pl.Plugin, pluginctl.WithPath(path)).Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to build plugin %s with label %s: %w", pl.Plugin, name, err)
	}
	pipeCount := len(pl.Pipe)
	if pl.Pipe == nil || pipeCount == 0 {
		return plugin, nil
	}
	pipes := make([]Pipeable, 0, pipeCount)
	for _, pipeMapConfig := range pl.Pipe {
		for pipeName, pipeConfig := range pipeMapConfig {
			pipeable, err := BuildPipe(pipeName, pipeConfig)
			if err != nil {

				slog.Error("failed to build pipe", "error", err)
				return nil, fmt.Errorf("failed to build pipe %s for %s with label %s: %w", pipeName, pl.Plugin, name, err)
			}
			pipes = append(pipes, pipeable)
		}
	}
	pipe := NewChainedPipe(pipes...)
	return NewSecPipePlugin(pipe, plugin), nil
}

func (pl Stage) GetSecVertex(name string) (SecVertex, error) {
	defaultPath := "/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins"
	if pl.PluginPath != "" {
		defaultPath = pl.PluginPath
	}
	plugin, err := pl.createPlugin(name, defaultPath)
	if err != nil {
		return SecVertex{}, err
	}
	if err := pl.configPlugin(name, plugin); err != nil {
		return SecVertex{}, err
	}
	return SecVertex{Name: name, plugin: plugin}, nil
}

func BuildPipe(pipeName string, config yaml.Node) (Pipeable, error) {
	switch pipeName {
	case "goTemplate":
		configGoTpl := GoTemplateConfig{}
		err := config.Decode(&configGoTpl)
		if err != nil {
			return nil, fmt.Errorf("failed to build goTemplate pipe : %w", err)
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
			return nil, fmt.Errorf("failed to build regex pipe : %w", err)
		}
		return NewRegexpPipe(configRegex.Pattern, configRegex.Select), nil
	case "insert":
		configInsert := struct {
			Content string `yaml:"content"`
		}{}
		err := config.Decode(&configInsert)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex pipe : %w", err)
		}
		return NewInsertStringPipe(configInsert.Content), nil
	case "split":
		configSplit := struct {
			Content string `yaml:"sep"`
		}{}
		err := config.Decode(&configSplit)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex pipe : %w", err)
		}
		return NewSplitPipe(configSplit.Content), nil
	default:
		return NewEmptyPipe(), errors.New("failed to build template. Template unknown")
	}
}

func (t Template) Start(ctx context.Context) error {

	//parentOutputCIndex: store the outputs of parents plugins where key is the child plugin
	parentOutputCIndex := map[string][]<-chan *pluginctl.DataStream{}
	errorsOutputCIndex := map[string]<-chan error{}
	graphSec, rootVertex, err := t.getTemplateGraph()
	if err != nil {
		return fmt.Errorf("failed to start template %w", err)
	}
	graph.BFS(graphSec, rootVertex.Name, func(s string) bool {

		currentVertex, err := graphSec.Vertex(s)
		if err != nil {
			slog.Error("strange error: current visited vertex not found in the graph", "error", err)
			return true
		}
		currentPlugin := currentVertex.plugin
		slog.Debug("current plugin", "plugin", currentPlugin, "vertex", currentVertex.Name)
		parentOutputC := chantools.Merge(parentOutputCIndex[s]...)

		childsMap, err := graphSec.AdjacencyMap()
		if err != nil {
			slog.Error(fmt.Sprintf("failed to start template %v", err))
			return true
		}
		childsNodes := childsMap[s]
		currentOutputC, currentErrC := NewPluginPipe(currentPlugin).Pipe(ctx, parentOutputC)

		parentOutputCForChilds := chantools.Broadcast(currentOutputC, uint(len(childsNodes)))
		errorsOutputCIndex[s] = currentErrC

		//Duplicate current Node output stream. Each duplicate will serve as an input for childNodes.
		//Skipping this, leads to each childs listening the same channels.. which mean only one child node will receive the stream.

		// For each childNodes (of the current node) we register the currentNode output as an input for the childs node
		// (save it to be able to retrieve it when the child node will be visited)
		i := 0
		for n := range childsNodes {
			childInputC, exist := parentOutputCIndex[n]
			if !exist {
				childInputC = make([]<-chan *pluginctl.DataStream, 0, 1)
			}
			childInputC = append(childInputC, parentOutputCForChilds[i])
			parentOutputCIndex[n] = childInputC
			i++
		}
		return false
	})
	return nil
}

func NewFileTemplate(path string) (Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Template{}, err
	}
	return NewRawTemplate(content)
}

func NewRawTemplate(raw []byte) (Template, error) {
	tpl := Template{}
	err := yaml.Unmarshal(raw, &tpl)

	return tpl, err

}

func newPlugin(name string) pluginctl.SecPipelinePluginable {
	p, e := pluginctl.NewPlugin(name, pluginctl.WithPath("/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins")).Connect()
	if e != nil {
		slog.Error("failed to get new plugin", "function", "newPlugin", "error", e)
	}
	return p
}

// func NewMockTemplate() Template {
// 	return Template{
// 		Pipeline: MockTemplate,
// 	}
// }

// var MockTemplate = tree.NewNode[pluginctl.SecPipelinePluginable, string](
// 	"martian",
// 	newPlugin("martianProxy"),
// 	tree.NewNode(
// 		"leaks",
// 		NewSecPipePlugin(
// 			NewChainedPipe(
// 				NewGoTemplatePipe(
// 					WithJsonInput(),
// 					WithTemplatePattern("{{ .response.content.text }},\n"),
// 				),
// 				// NewBase64Decoder(),
// 			), newPlugin("leaks")),
// 		tree.NewNode("rawfile",
// 			NewSecPipePlugin(
// 				NewChainedPipe(
// 					NewGoTemplatePipe(
// 						WithJsonInput(),
// 						WithTemplatePattern("{{ .response.content.text }},\n"),
// 					),
// 					// NewBase64Decoder(),
// 				), newPlugin("rawfile")),
// 		),
// 	),
// 	// tree.NewNode(
// 	// 	"spider",
// 	// 	NewSecPipePlugin(NewGoTemplatePipe(WithJsonInput(), WithTemplatePattern("")))
// 	// 	newPlugin("spider"),
// 	// ),
// )
