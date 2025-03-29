package template

import (
	"bytes"
	"fmt"
	"maps"
	"os"
	"text/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/load"
	"github.com/benji-bou/lugh/helper"
	"gopkg.in/yaml.v3"
)

type TemplateConfig struct {
	PluginPath string
	Variables  map[string]interface{}
	Loader     *load.Loader
}

// wrapTemplatePlugin wraps the `templates plugins“ loader to pass template config to plugins.
// This is useful for plugins that need to know the template config, like the `include` plugin.
func (tc *TemplateConfig) wrapTemplatePlugin() {
	tc.Loader.WrapLoader("include", func(next load.Loadable) load.Loadable {
		return load.ConfigAsMap(func(name string, path string, config map[string]any) (any, error) {
			if includesVar, ok := config["variables"]; !ok {
				config["variables"] = tc.Variables
			} else if includes, ok := includesVar.(map[string]any); !ok {
				return nil, fmt.Errorf("invalid include variables type. Expected map got %T", includesVar)
			} else {
				variables := make(map[string]any)
				maps.Copy(variables, tc.Variables)
				maps.Copy(variables, includes)
				config["variables"] = variables
			}
			config["pluginpath"] = tc.PluginPath
			return next.Load(name, path, config)
		})
	})
}

type TemplateOption = helper.Option[TemplateConfig]

func WithLoader(loader *load.Loader) TemplateOption {
	return func(t *TemplateConfig) {
		t.Loader = loader
		t.wrapTemplatePlugin()
	}
}

func WithDefaultLoader() TemplateOption {
	return WithLoader(load.Default())
}

func WithVariables(variables map[string]interface{}) TemplateOption {
	return func(t *TemplateConfig) {
		maps.Copy(t.Variables, variables)
	}
}

func WithPluginPath(path string) TemplateOption {
	return func(t *TemplateConfig) {
		t.PluginPath = path
	}
}

type (
	PluginLoader interface {
		LoadPlugin(name string, tplCtx TemplateConfig) (graph.IOWorkerVertex[[]byte], error)
	}
)

type Template[S PluginLoader] struct {
	Name        string       `yaml:"name" json:"name"`
	Description string       `yaml:"description" json:"description"`
	Version     string       `yaml:"version" json:"version"`
	Author      string       `yaml:"author" json:"author"`
	Stages      map[string]S `yaml:"stages" json:"stages"`
	config      TemplateConfig
}

func (t Template[S]) Raw() ([]byte, error) {
	tplBytes, err := yaml.Marshal(t)
	if err != nil {
		return nil, fmt.Errorf("failed to mashal template to yaml, %w", err)
	}
	return tplBytes, nil
}

func NewFile[S PluginLoader](path string, opt ...TemplateOption) (Template[S], error) {
	return NewTemplateFromFile[S](path, opt...)
}

func NewTemplateFromFile[S PluginLoader](path string, opt ...TemplateOption) (Template[S], error) {
	content, err := os.ReadFile(path) // #nosec G304
	if err != nil {
		return Template[S]{}, err
	}
	return NewTemplate[S](content, opt...)
}

func New[S PluginLoader](raw []byte, opt ...TemplateOption) (Template[S], error) {
	return NewTemplate[S](raw, opt...)
}

func NewTemplate[S PluginLoader](raw []byte, opt ...TemplateOption) (Template[S], error) {
	defaultOptions := make([]TemplateOption, 0, len(opt)+1)
	defaultOptions = append(defaultOptions, WithDefaultLoader())
	defaultOptions = append(defaultOptions, opt...)
	tplConfig := helper.Configure(TemplateConfig{Variables: map[string]interface{}{}}, defaultOptions...)
	raw, err := InterpolateVariable(raw, tplConfig.Variables)
	if err != nil {
		return Template[S]{}, fmt.Errorf("parsing template, %w", err)
	}
	tpl := Template[S]{config: tplConfig}
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

func (t Template[S]) WorkerVertexIterator() ([]graph.IOWorkerVertex[[]byte], error) {
	workerVertices := make([]graph.IOWorkerVertex[[]byte], 0, len(t.Stages))
	for name, rawStage := range t.Stages {
		worker, err := rawStage.LoadPlugin(name, t.config)
		if err != nil {
			return nil, err
		}
		workerVertices = append(workerVertices, worker)
	}
	return workerVertices, nil
}
