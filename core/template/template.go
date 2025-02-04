package template

import (
	"fmt"
	"iter"
	"log/slog"
	"os"

	"github.com/benji-bou/lugh/core/graph"
	"gopkg.in/yaml.v3"
)

type PluginLoader interface {
	LoadPlugin(name string, defaultPluginsPath string) graph.IOWorkerVertex[[]byte]
}

type Template[S PluginLoader] struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Version     string       `yaml:"version" json:"version"`
	Author      string       `yaml:"author" json:"author"`
	Stages      map[string]S `yaml:"stages" json:"stages"`
}

func (t Template[S]) Raw() ([]byte, error) {
	tplBytes, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal template to yaml, %w", err)
	}
	return tplBytes, nil
}

func NewFile(path string) (Template[Stage], error) {
	return NewTemplateFromFile[Stage](path)
}

func NewTemplateFromFile[S PluginLoader](path string) (Template[Stage], error) {
	content, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return Template[Stage]{}, err
	}
	return NewTemplate[Stage](content)
}

func New(raw []byte) (Template[Stage], error) {
	return NewTemplate[Stage](raw)
}

func NewTemplate[S PluginLoader](raw []byte) (Template[S], error) {
	tpl := Template[S]{}
	err := yaml.Unmarshal(raw, &tpl)
	return tpl, err
}

func (t Template[S]) WorkerVertexIterator(defaultPluginsPath string) iter.Seq[graph.IOWorkerVertex[[]byte]] {
	return func(yield func(graph.IOWorkerVertex[[]byte]) bool) {
		for name, rawStage := range t.Stages {
			worker := rawStage.LoadPlugin(name, defaultPluginsPath)
			slog.Debug("vertex", "name", worker.GetName(), "parents", worker.GetParents())
			if worker != nil {
				if !yield(worker) {
					return
				}
			}
		}
	}
}
