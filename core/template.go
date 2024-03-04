package core

import (
	"log/slog"

	"github.com/benji-bou/SecPipeline/pluginctl"
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
	// currentNode := t.Pipeline
	// var currentInputC <-chan []byte
	// childOutputC := map[string]struct {
	// 	outputC    <-chan []byte
	// 	outputErrC <-chan error
	// }{}
	tree.Walker(t.Pipeline, func(currentNode SecNode) error {
		// currentPlugin := currentNode.GetValue()

		// //TODO: Stoquer l'outputC de newPipe (du plugin enfant) dans une map pour l'envoyer dans ces propres enfants plus tard
		// //TODO: ICI nous NE devons PAS changer currentInputC car utilisé pour le prochain child du current Node
		// tmpOuputC, errC := NewPipe(currentInputC, childPlugin).Pipe()
		// childOutputC[childNode.GetID()] = struct {
		// 	outputC    <-chan []byte
		// 	outputErrC <-chan error
		// }{
		// 	outputC:    tmpOuputC,
		// 	outputErrC: errC,
		// }
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
	p, e := pluginctl.NewPlugin("martianProxy", pluginctl.WithPath("/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins")).Connect()
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
