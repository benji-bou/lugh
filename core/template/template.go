package template

import (
	"fmt"
	"iter"
	"os"

	"github.com/benji-bou/SecPipeline/core/graph"
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

func (t Template) SecVertexIterator() iter.Seq[graph.SecVertexer] {
	return func(yield func(graph.SecVertexer) bool) {
		for name, stage := range t.Stages {
			stage.name = name
			if !yield(stage) {
				return
			}
		}

	}
}
