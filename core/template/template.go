package template

import (
	"bytes"
	"fmt"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/benji-bou/lugh/core/graph"
	"gopkg.in/yaml.v3"
)

type PluginLoader interface {
	LoadPlugin(name string, defaultPluginsPath string, variables map[string]interface{}) (graph.IOWorkerVertex[[]byte], error)
}

type Template[S PluginLoader] struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Version     string       `yaml:"version" json:"version"`
	Author      string       `yaml:"author" json:"author"`
	Stages      map[string]S `yaml:"stages" json:"stages"`
	Variables   map[string]interface{}
}

func (t Template[S]) Raw() ([]byte, error) {
	tplBytes, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal template to yaml, %w", err)
	}
	return tplBytes, nil
}

func NewFile(path string, variables map[string]interface{}) (Template[Stage], error) {
	return NewTemplateFromFile[Stage](path, variables)
}

func NewTemplateFromFile[S PluginLoader](path string, variables map[string]interface{}) (Template[Stage], error) {
	content, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return Template[Stage]{}, err
	}
	return NewTemplate[Stage](content, variables)
}

func New(raw []byte, variables map[string]interface{}) (Template[Stage], error) {
	return NewTemplate[Stage](raw, variables)
}

func NewTemplate[S PluginLoader](raw []byte, variables map[string]interface{}) (Template[S], error) {
	raw, err := InterpolateVariable(raw, variables)
	if err != nil {
		return Template[S]{}, fmt.Errorf("parsing template, %w", err)
	}
	tpl := Template[S]{Variables: variables}
	err = yaml.Unmarshal(raw, &tpl)
	return tpl, err
}

func InterpolateVariable(raw []byte, variables map[string]interface{}) ([]byte, error) {
	goTpl, err := template.New("TemplateInterpolation").Funcs(sprig.FuncMap()).Parse(string(raw))
	if err != nil {
		return raw, fmt.Errorf("as go template failed, %w", err)
	}
	interpolatedTemplate := &bytes.Buffer{}
	err = goTpl.Execute(interpolatedTemplate, variables)
	if err != nil {
		return nil, fmt.Errorf("executing variable interpolation, %w", err)
	}
	res := interpolatedTemplate.Bytes()
	fmt.Printf("raw: %s", string(res))
	return res, nil
}

func (t Template[S]) WorkerVertexIterator(defaultPluginsPath string) ([]graph.IOWorkerVertex[[]byte], error) {
	workerVertices := make([]graph.IOWorkerVertex[[]byte], 0, len(t.Stages))
	for name, rawStage := range t.Stages {
		worker, err := rawStage.LoadPlugin(name, defaultPluginsPath, t.Variables)
		if err != nil {
			return nil, err
		}
		workerVertices = append(workerVertices, worker)
	}
	return workerVertices, nil
}
