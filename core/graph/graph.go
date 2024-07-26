package graph

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"golang.org/x/exp/maps"

	"github.com/benji-bou/SecPipeline/core/plugin"
	"github.com/benji-bou/SecPipeline/core/template"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/google/uuid"
)

type SecGraph struct {
	graph      graph.Graph[string, SecVertex]
	rootVertex SecVertex
}

type SecGraphOption = helper.OptionError[SecGraph]

func WithTemplate(template template.Template) SecGraphOption {
	return func(configure *SecGraph) error {
		if err := configure.addStagesVertex(template); err != nil {
			return err
		}
		if err := configure.addStagesEdges(template); err != nil {
			return err
		}
		return nil
	}
}

func WithRawTemplate(rawTemplate []byte) SecGraphOption {
	return func(configure *SecGraph) error {
		tpl, err := template.NewRawTemplate(rawTemplate)
		if err != nil {
			return err
		}
		return WithTemplate(tpl)(configure)
	}
}

func WithTemplatePath(tplPath string) SecGraphOption {
	return func(configure *SecGraph) error {
		tpl, err := template.NewFileTemplate(tplPath)

		if err != nil {
			return err
		}
		return WithTemplate(tpl)(configure)
	}
}

func NewGraph(opt ...SecGraphOption) (*SecGraph, error) {
	sg := &SecGraph{}
	sg.graph = graph.New(func(spp SecVertex) string {
		return spp.Name
	}, graph.Directed())
	sg.rootVertex = SecVertex{Name: uuid.NewString(), plugin: plugin.EmptySecPlugin{}}
	sg.graph.AddVertex(sg.rootVertex)
	return helper.ConfigurePtrWithError(sg, opt...)
}

func (sg *SecGraph) DrawGraph(filepath string) error {
	file, _ := os.Create(filepath)
	return draw.DOT(sg.graph, file)
}

func (sg *SecGraph) GetChildlessVertex() (map[string]SecVertex, error) {
	res := make(map[string]SecVertex)
	childsMap, err := sg.graph.AdjacencyMap()
	if err != nil {
		panic(err)
	}
	for vertexId, childs := range childsMap {
		if len(childs) == 0 {
			res[vertexId], err = sg.graph.Vertex(vertexId)
			if err != nil {
				return nil, err
			}
		}
	}
	return res, nil
}

func (sg *SecGraph) AddSecVertex(newVertex SecVertex, parentVertex ...SecVertex) error {
	err := sg.graph.AddVertex(newVertex)
	if err != nil {
		return err
	}
	for _, p := range parentVertex {
		err := sg.graph.AddEdge(p.Name, newVertex.Name)
		if err != nil {
			return err
		}
	}
	return nil
}

func (sg *SecGraph) Start(ctx context.Context) (error, <-chan error) {

	// parentOutputCIndex: store the outputs of parents plugins where key is the child plugin
	parentOutputCIndex := map[string][]<-chan *pluginctl.DataStream{}
	errorsOutputCIndex := map[string]<-chan error{}

	graph.BFS(sg.graph, sg.rootVertex.Name, func(s string) bool {

		currentVertex, err := sg.graph.Vertex(s)
		if err != nil {
			slog.Error("strange error: current visited vertex not found in the graph", "error", err)
			return true
		}
		currentPlugin := currentVertex.plugin
		slog.Debug("current plugin", "plugin", currentPlugin, "vertex", currentVertex.Name)
		//Merge and Pipes all parents output into current plugins
		parentOutputC := chantools.Merge(parentOutputCIndex[s]...)
		currentOutputC, currentErrC := plugin.NewPipeToPlugin(currentPlugin).Pipe(ctx, parentOutputC)
		errorsOutputCIndex[s] = currentErrC

		// Get all childs of current Vertex
		// Broadcast: create an output chan for each childs that will receive a copy of current plugin output)
		// save for each child its own chan copy of current plugin output

		childsMap, err := sg.graph.AdjacencyMap()
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
	slog.Debug("Workflow started")
	return nil, chantools.Merge(maps.Values(errorsOutputCIndex)...)
}

func (sg *SecGraph) addStagesVertex(t template.Template) error {
	for stageName, stage := range t.Stages {
		err := sg.addStagedSecVertex(stageName, stage)
		if err != nil {
			return fmt.Errorf("failed to get graph.secGraph because: %w", err)
		}
	}
	return nil
}

func (sg *SecGraph) addStagedSecVertex(name string, stage template.Stage) error {
	vertex, err := NewSecVertex(name, VertexFromStage(stage))
	if err != nil {
		return fmt.Errorf("failed to get SecVertex %s  because: %w", name, err)
	}
	err = sg.graph.AddVertex(vertex)
	if err != nil && err != graph.ErrVertexAlreadyExists {
		return fmt.Errorf("couldn't add SecVertex %s to graph because: %w", name, err)
	}
	return nil
}

func (sg *SecGraph) addStagesEdges(t template.Template) error {
	for stageName, stage := range t.Stages {
		sg.addStagedEdges(stageName, stage)
	}
	return nil
}

func (sg *SecGraph) addStagedEdges(name string, st template.Stage) error {
	if len(st.Parents) == 0 {
		err := sg.graph.AddEdge(sg.rootVertex.Name, name)
		if err != nil {
			return fmt.Errorf("failed to link rootVertex to  %s  because: %w", name, err)
		}
	}
	for _, p := range st.Parents {
		err := sg.graph.AddEdge(p, name)
		if err != nil && err != graph.ErrEdgeAlreadyExists {
			return fmt.Errorf("failed to link %s to  %s because: %w", p, name, err)
		}
	}
	return nil
}
