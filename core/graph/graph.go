package graph

import (
	"context"
	"fmt"
	"os"

	"github.com/benji-bou/SecPipeline/core/plugin"
	"github.com/benji-bou/SecPipeline/core/template"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/dominikbraun/graph"
	"github.com/dominikbraun/graph/draw"
	"github.com/google/uuid"
)

type SecGraph struct {
	graph graph.Graph[string, SecVertex]
}

type SecGraphOption = helper.Option[SecGraph]

func WithTemplate(template template.Template) SecGraphOption {
	return func(configure *SecGraph) {

	}
}

func NewGraph(opt ...SecGraphOption) *SecGraph {
	return helper.ConfigurePtr(&SecGraph{}, opt...)
	// if err := t.addStagesVertex(g); err != nil {
	// 	return g, rootVertex, err
	// }
	// if err := t.addStagesEdges(g, rootVertex); err != nil {
	// 	return g, rootVertex, err
	// }
	// return g, rootVertex, nil

	// return
}

func (sg *SecGraph) DrawGraph(filepath string) error {
	file, _ := os.Create(filepath)
	return draw.DOT(sg.graph, file)
}

func (sg *SecGraph) generateGraph() {

	sg.graph = graph.New(func(spp SecVertex) string {
		return spp.Name
	}, graph.Directed())
	rootVertex := SecVertex{Name: uuid.NewString(), plugin: plugin.EmptySecPlugin{}}
	sg.graph.AddVertex(rootVertex)
}

func (sg *SecGraph) addStagesVertex(t template.Template) error {
	for stageName, stage := range t.Stages {
		err := stage.AddSecVertex(sg.graph, stageName)
		if err != nil {
			return fmt.Errorf("failed to get graph.secGraph because: %w", err)
		}
	}
	return nil
}

func (sg *SecGraph) addStagesEdges(rootVertex SecVertex, t template.Template) error {
	for stageName, stage := range t.Stages {
		stage.AddEdges(sg.graph, stageName, rootVertex)
	}
	return nil
}

func (sg *SecGraph) Start(ctx context.Context, t template.Template) (error, <-chan error) {

	//parentOutputCIndex: store the outputs of parents plugins where key is the child plugin
	// parentOutputCIndex := map[string][]<-chan *pluginctl.DataStream{}
	// errorsOutputCIndex := map[string]<-chan error{}
	// graphSec, rootVertex, err := t.getTemplateGraph()
	// if err != nil {
	// 	return fmt.Errorf("failed to build graph because: %w", err), nil
	// }
	// graph.BFS(graphSec, rootVertex.Name, func(s string) bool {

	// 	currentVertex, err := graphSec.Vertex(s)
	// 	if err != nil {
	// 		slog.Error("strange error: current visited vertex not found in the graph", "error", err)
	// 		return true
	// 	}
	// 	currentPlugin := currentVertex.plugin
	// 	slog.Debug("current plugin", "plugin", currentPlugin, "vertex", currentVertex.Name)
	// 	//Merge and Pipes all parents output into current plugins
	// 	parentOutputC := chantools.Merge(parentOutputCIndex[s]...)
	// 	currentOutputC, currentErrC := NewPluginPipe(currentPlugin).Pipe(ctx, parentOutputC)
	// 	errorsOutputCIndex[s] = currentErrC

	// 	// Get all childs of current Vertex
	// 	// Broadcast: create an output chan for each childs that will receive a copy of current plugin output)
	// 	// save for each child its own chan copy of current plugin output

	// 	childsMap, err := graphSec.AdjacencyMap()
	// 	if err != nil {
	// 		slog.Error(fmt.Sprintf("failed to start template %v", err))
	// 		return true
	// 	}
	// 	childsNodes := childsMap[s]
	// 	parentOutputCForChilds := chantools.Broadcast(currentOutputC, uint(len(childsNodes)))

	// 	// For each childNodes (of the current node) we register the currentNode output as an input for the childs node
	// 	// (save it to be able to retrieve it when the child node will be visited)
	// 	i := 0
	// 	for n := range childsNodes {
	// 		childInputC, exist := parentOutputCIndex[n]
	// 		if !exist {
	// 			childInputC = make([]<-chan *pluginctl.DataStream, 0, 1)
	// 		}
	// 		childInputC = append(childInputC, parentOutputCForChilds[i])
	// 		parentOutputCIndex[n] = childInputC
	// 		i++
	// 	}
	// 	return false
	// })
	// return nil, chantools.Merge(maps.Values(errorsOutputCIndex)...)
}

func (sg *SecGraph) AddSecVertex(g graph.SecGraph, name string) error {
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

func (sg *SecGraph) AddEdges(g graph.SecGraph, name string, rootVertex graph.SecVertex) error {
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
