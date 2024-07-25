package template

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Template struct {
	Name        string           `yaml:"name" json:"name"`
	Description string           `yaml:"description" json:"description"`
	Version     string           `yaml:"version" json:"version"`
	Author      string           `yaml:"author" json:"author"`
	Stages      map[string]Stage `yaml:"stages" json:"stages"`
}

func (t *Template) AddStageToBottom(name string, stage Stage) {
	parents := make([]string, 0, 1)
	for name, stage := range t.Stages {
		if len(stage.Parents) == 0 {
			parents = append(parents, name)
		}
	}
	stage.Parents = parents
	t.Stages[name] = stage
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
