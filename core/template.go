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
	"github.com/benji-bou/tree"
	"gopkg.in/yaml.v3"
)

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

type SecNode = tree.Nodable[pluginctl.SecPipelinePluginable, string]

type Template struct {
	Name        string              `yaml:"name" json:"name"`
	Description string              `yaml:"description" json:"description"`
	Version     string              `yaml:"version" json:"version"`
	Author      string              `yaml:"author" json:"author"`
	Pipeline    map[string]Pipeline `yaml:"pipeline" json:"pipeline"`
}

func (t Template) GetSecNodeTree() (SecNode, error) {
	if len(t.Pipeline) > 1 {
		return nil, fmt.Errorf("multiple root node in the pipeline detected")
	}
	for _, child := range t.Pipeline {
		return child.GetSecNodeTree()
	}
	return nil, fmt.Errorf("No pipeline defined")
}

type Pipeline struct {
	PluginPath string                 `yaml:"pluginPath"`
	Plugin     string                 `yaml:"plugin"`
	Config     map[string]any         `yaml:"config"`
	Childs     map[string]Pipeline    `yaml:"childs"`
	Pipe       []map[string]yaml.Node `yaml:"pipe"`
}

func (pl Pipeline) GetSecNodeTree() (SecNode, error) {
	currentNode, err := pl.GetSecNode()
	if err != nil {
		return nil, fmt.Errorf("failed build secnode tree, %w", err)
	}
	childsNode := make([]SecNode, 0, len(pl.Childs))
	for _, child := range pl.Childs {
		childNode, err := child.GetSecNodeTree()
		if err != nil {
			return nil, fmt.Errorf("failed build secnode tree node: %s, %w", pl.Plugin, err)
		}
		childsNode = append(childsNode, childNode)
	}
	currentNode.(*tree.Node[pluginctl.SecPipelinePluginable, string]).AddNode(childsNode...)
	return currentNode, nil
}

func (pl Pipeline) GetSecNode() (SecNode, error) {
	defaultPath := "/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins"
	if pl.PluginPath != "" {
		defaultPath = pl.PluginPath
	}
	plugin, err := pluginctl.NewPlugin(pl.Plugin, pluginctl.WithPath(defaultPath)).Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to build plugin %s: %w", pl.Plugin, err)
	}
	if pl.Config != nil && len(pl.Plugin) > 0 {
		cJson, err := json.Marshal(pl.Config)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal configuration %s: %w", pl.Plugin, err)
		}
		err = plugin.Config(cJson)
		if err != nil {
			return nil, fmt.Errorf("failed to configure %s: %w", pl.Plugin, err)
		}
	}
	pipeCount := len(pl.Pipe)
	if pl.Pipe == nil || pipeCount == 0 {
		return tree.NewNode(pl.Plugin, plugin), nil
	}

	pipes := make([]Pipeable, 0, pipeCount)
	for _, pipeMapConfig := range pl.Pipe {
		for pipeName, pipeConfig := range pipeMapConfig {
			pipeable, err := BuildPipe(pipeName, pipeConfig)
			if err != nil {

				slog.Error("failed to build pipe", "error", err)
				return nil, fmt.Errorf("failed to build pipe %s for %s: %w", pipeName, pl.Plugin, err)
			}
			pipes = append(pipes, pipeable)
		}
	}
	pipe := NewChainedPipe(pipes...)
	return tree.NewNode(pl.Plugin, NewSecPipePlugin(pipe, plugin)), nil
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

	parentOutputCIndex := map[string][]<-chan *pluginctl.DataStream{}
	errorsOutputCIndex := map[string]<-chan error{}
	treeSecNode, err := t.GetSecNodeTree()
	if err != nil {
		return fmt.Errorf("failed to start template %w", err)
	}
	// walk accross the secNode tree in a levelordered way. (aka, every nodes from a level before visiting sublevels nodes)
	tree.Walker(treeSecNode, func(currentNode SecNode) error {
		currentPlugin := currentNode.GetValue()
		slog.Debug("current plugin", "plugin", currentPlugin)
		// Merging all parents outputs to feed the current node
		parentOutputC := chantools.Merge(parentOutputCIndex[currentNode.GetID()]...)

		childsNodes := currentNode.GetChilds()

		//Piping the parentStream output into the currentpluging
		currentOutputC, currentErrC := NewPluginPipe(currentPlugin).Pipe(ctx, parentOutputC)

		//Duplicate current Node output stream. Each duplicate will serve as an input for childNodes.
		//Skipping this, leads to each childs listening the same channels.. which mean only one child node will receive the stream.
		parentOutputCForChilds := chantools.Broadcast(currentOutputC, uint(len(childsNodes)))
		errorsOutputCIndex[currentNode.GetID()] = currentErrC

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

		return nil
	}, tree.LevelOrderSearch)
	return nil
}

func NewFileTemplate(path string) (Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Template{}, err
	}
	tpl := Template{}
	err = yaml.Unmarshal(content, &tpl)
	return tpl, err
}

func NewRawTemplate(raw interface{}) Template {
	return Template{}
}

// func NewMockTemplate() Template {
// 	return Template{
// 		Pipeline: MockTemplate,
// 	}
// }

func newPlugin(name string) pluginctl.SecPipelinePluginable {
	p, e := pluginctl.NewPlugin(name, pluginctl.WithPath("/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins")).Connect()
	if e != nil {
		slog.Error("failed to get new plugin", "function", "newPlugin", "error", e)
	}
	return p
}

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
