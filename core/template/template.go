package template

import (
	"fmt"
	"iter"
	"log/slog"
	"os"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/helper"
	"gopkg.in/yaml.v3"
)

type PluginLoader interface {
	LoadPlugin(name string) graph.IOWorkerVertex[[]byte]
}

type Option = helper.Option[Template]

type StageLoader func(name string, yml *yaml.Node) (PluginLoader, error)

func WithStageLoader(loader StageLoader) Option {
	return func(tpl *Template) {
		tpl.loadStage = loader
	}
}

func SimpleStageLoader(getPluginLoader func() PluginLoader) StageLoader {
	stage := getPluginLoader()
	return func(name string, yml *yaml.Node) (PluginLoader, error) {
		err := yml.Decode(&stage)
		return stage, err
	}
}

func defaultStageLoader() StageLoader {
	return SimpleStageLoader(func() PluginLoader {
		return Stage{}
	})
}

type Template struct {
	Name        string                `yaml:"name" json:"name"`
	Description string                `yaml:"description" json:"description"`
	Version     string                `yaml:"version" json:"version"`
	Author      string                `yaml:"author" json:"author"`
	Stages      map[string]*yaml.Node `yaml:"stages" json:"stages"`
	loadStage   StageLoader
}

func (t Template) Raw() ([]byte, error) {
	tplBytes, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal template to yaml, %w", err)
	}
	return tplBytes, nil
}

func NewFile[S Stage](path string, opt ...Option) (Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Template{}, err
	}
	return New(content, opt...)
}

func New[S Stage](raw []byte, opt ...Option) (Template, error) {
	tpl := helper.Configure(Template{loadStage: defaultStageLoader()}, opt...)
	err := yaml.Unmarshal(raw, &tpl)
	return tpl, err

}

func (t Template) IterPluginLoader() iter.Seq2[string, PluginLoader] {
	return func(yield func(string, PluginLoader) bool) {
		for name, rawStage := range t.Stages {
			stage, err := t.loadStage(name, rawStage)
			if err != nil {
				slog.Error("iter stages", "name", name, "error", err)
				continue
			}
			if !yield(name, stage) {
				return
			}
		}
	}
}

func (t Template) WorkerVertexIterator() iter.Seq[graph.IOWorkerVertex[[]byte]] {
	return func(yield func(graph.IOWorkerVertex[[]byte]) bool) {
		for name, rawStage := range t.IterPluginLoader() {
			worker := rawStage.LoadPlugin(name)
			slog.Debug("vertex", "name", worker.GetName(), "parents", worker.GetParents())
			if worker != nil {
				if !yield(worker) {
					return
				}
			}
		}
	}
}
