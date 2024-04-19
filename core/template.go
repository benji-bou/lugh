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
	"github.com/dominikbraun/graph/draw"
	"github.com/google/uuid"
	"golang.org/x/exp/maps"
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

func (t *Template) DrawGraphOnly(filepath string) error {
	g, _, err := t.getTemplateGraph()
	if err != nil {
		return fmt.Errorf("failed to build graph template because: %w", err)
	}
	file, _ := os.Create(filepath)
	return draw.DOT(g, file)
}

func (t *Template) getTemplateGraph() (SecGraph, SecVertex, error) {

	g := graph.New(func(spp SecVertex) string {
		return spp.Name
	}, graph.Directed())
	rootVertex := SecVertex{Name: uuid.NewString(), plugin: EmptySecPlugin{}}
	g.AddVertex(rootVertex)
	if err := t.addStagesVertex(g); err != nil {
		return g, rootVertex, err
	}
	if err := t.addStagesEdges(g, rootVertex); err != nil {
		return g, rootVertex, err
	}
	return g, rootVertex, nil
}

func (t *Template) addStagesVertex(g SecGraph) error {
	for stageName, stage := range t.Stages {
		err := stage.AddSecVertex(g, stageName)
		if err != nil {
			return fmt.Errorf("failed to get secGraph because: %w", err)
		}
	}
	return nil
}

func (t *Template) addStagesEdges(g SecGraph, rootVertex SecVertex) error {
	for stageName, stage := range t.Stages {
		stage.AddEdges(g, stageName, rootVertex)
	}
	return nil
}

func (t Template) Start(ctx context.Context) (error, <-chan error) {

	//parentOutputCIndex: store the outputs of parents plugins where key is the child plugin
	parentOutputCIndex := map[string][]<-chan *pluginctl.DataStream{}
	errorsOutputCIndex := map[string]<-chan error{}
	graphSec, rootVertex, err := t.getTemplateGraph()
	if err != nil {
		return fmt.Errorf("failed to build graph because: %w", err), nil
	}
	graph.BFS(graphSec, rootVertex.Name, func(s string) bool {

		currentVertex, err := graphSec.Vertex(s)
		if err != nil {
			slog.Error("strange error: current visited vertex not found in the graph", "error", err)
			return true
		}
		currentPlugin := currentVertex.plugin
		slog.Debug("current plugin", "plugin", currentPlugin, "vertex", currentVertex.Name)
		//Merge and Pipes all parents output into current plugins
		parentOutputC := chantools.Merge(parentOutputCIndex[s]...)
		currentOutputC, currentErrC := NewPluginPipe(currentPlugin).Pipe(ctx, parentOutputC)
		errorsOutputCIndex[s] = currentErrC

		// Get all childs of current Vertex
		// Broadcast: create an output chan for each childs that will receive a copy of current plugin output)
		// save for each child its own chan copy of current plugin output

		childsMap, err := graphSec.AdjacencyMap()
		if err != nil {
			slog.Error(fmt.Sprintf("failed to start template %v", err))
			return true
		}
		childsNodes := childsMap[s]
		parentOutputCForChilds := chantools.Broadcast(currentOutputC, uint(len(childsNodes)))

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
	return nil, chantools.Merge(maps.Values(errorsOutputCIndex)...)
}

func (t Template) Raw() ([]byte, error) {
	tplBytes, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal template to yaml, %w", err)
	}
	return tplBytes, nil
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

type Stage struct {
	Parents    []string               `yaml:"parents"`
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     map[string]any         `yaml:"config"`
	Pipe       []map[string]yaml.Node `yaml:"pipe"`
}

func (st Stage) AddSecVertex(g SecGraph, name string) error {
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

func (st Stage) AddEdges(g SecGraph, name string, rootVertex SecVertex) error {
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
	defaultPath := "/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins"
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

func (st Stage) buildPipe(pipeName string, config yaml.Node) (Pipeable, error) {
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
