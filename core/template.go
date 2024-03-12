package core

import (
	"context"
	"log/slog"

	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
	"github.com/benji-bou/tree"
)

type SecNode = tree.Nodable[pluginctl.SecPipelinePluginable, string]

type Template struct {
	Description string `yaml:"description" json:"description"`
	Version     string `yaml:"version" json:"version"`
	Author      string `yaml:"author" json:"author"`
	path        string `yaml:"path" json:"path"`
	Pipeline    SecNode
}

func (t Template) Start() {

	parentOutputCIndex := map[string][]<-chan *pluginctl.DataStream{}
	errorsOutputCIndex := map[string]<-chan error{}

	tree.Walker(t.Pipeline, func(currentNode SecNode) error {
		currentPlugin := currentNode.GetValue()
		slog.Debug("current plugin", "plugin", currentPlugin)
		parentOutputC := chantools.Merge(parentOutputCIndex[currentNode.GetID()]...)

		currentOutputC, currentErrC := NewPipe(parentOutputC, currentPlugin).Pipe(context.Background())
		errorsOutputCIndex[currentNode.GetID()] = currentErrC
		for n := range currentNode.GetChilds() {
			childInputC, exist := parentOutputCIndex[n]
			if !exist {
				childInputC = make([]<-chan *pluginctl.DataStream, 0, 1)
			}
			childInputC = append(childInputC, currentOutputC)
			parentOutputCIndex[n] = childInputC
		}

		return nil
	}, tree.LevelOrderSearch)

}

func NewFileTemplate(path string) Template {
	return Template{}
}

func NewRawTemplate(raw interface{}) Template {
	return Template{}
}

func NewMockTemplate() Template {
	return Template{
		Pipeline: MockTemplate,
	}
}

func newPlugin(name string) pluginctl.SecPipelinePluginable {
	p, e := pluginctl.NewPlugin(name, pluginctl.WithPath("/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins")).Connect()
	if e != nil {
		slog.Error("failed to get new plugin", "function", "newPlugin", "error", e)
	}
	return p
}

var MockTemplate = tree.NewNode(
	"martian",
	newPlugin("martianProxy"),
	tree.NewNode(
		"leaks",
		newPlugin("leaks"),
	),
)
