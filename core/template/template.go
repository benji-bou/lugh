package template

import (
	"fmt"
	"iter"
	"log/slog"
	"os"

	"github.com/benji-bou/lugh/core/graph"
	"gopkg.in/yaml.v3"
)

type Template struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Version     string           `yaml:"version" json:"version"`
	Author      string           `yaml:"author" json:"author"`
	Stages      map[string]Stage `yaml:"stages" json:"stages"`
}

func (t Template) Raw() ([]byte, error) {
	tplBytes, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal template to yaml, %w", err)
	}
	return tplBytes, nil
}

func NewFileTemplate[S Stage](path string) (Template, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return Template{}, err
	}
	return NewRawTemplate(content)
}

func NewRawTemplate[S Stage](raw []byte) (Template, error) {
	tpl := Template{}
	err := yaml.Unmarshal(raw, &tpl)
	return tpl, err

}

func (t Template) WorkerVertexIterator() iter.Seq[graph.IOWorkerVertex[[]byte]] {
	return func(yield func(graph.IOWorkerVertex[[]byte]) bool) {
		for name, stage := range t.Stages {
			worker := stage.LoadPlugin(name)
			slog.Debug("vertex", "name", worker.GetName(), "parents", worker.GetParents())
			if worker != nil {
				if !yield(worker) {
					return
				}
			}
		}
	}
}
