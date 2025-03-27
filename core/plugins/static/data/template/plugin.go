package template

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"html/template"

	"github.com/Masterminds/sprig/v3"
	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/helper"
	"gopkg.in/yaml.v2"
)

type Config struct {
	Pattern string `json:"pattern"`
	Format  string `json:"format"`
}

func Worker(config Config) (graph.Worker[[]byte], error) {
	opt := []workerOption{}
	switch config.Format {
	case "string":
		opt = append(opt, WithStringInput())
	case "yaml":
		opt = append(opt, WithYAMLInput())
	default:
		opt = append(opt, WithJSONInput())
	}
	opt = append(opt, WithTemplatePattern(config.Pattern))
	return new(opt...)
}

type workerOption = helper.OptionError[worker]

type worker struct {
	template    *template.Template
	unmarshaler func(in []byte, v any) error
}

func new(opt ...workerOption) (worker, error) {
	tmp, err := helper.ConfigureWithError(worker{}, opt...)
	if err != nil {
		return worker{}, fmt.Errorf("worker template initialization : %w", err)
	}
	if tmp.template == nil {
		return worker{}, fmt.Errorf("worker template initialization : template is nil")
	}
	return tmp, nil
}

func (w worker) Work(ctx context.Context, input []byte, yield func(elem []byte) error) error {
	var dataInput any
	err := w.unmarshaler(input, &dataInput)
	if err != nil {
		return fmt.Errorf("worker template: unmarshal input: %w", err)
	}
	buff := &bytes.Buffer{}
	err = w.template.ExecuteTemplate(buff, w.template.Name(), dataInput)
	if err != nil {
		return fmt.Errorf("worker template: template exection: %w", err)
	}
	res := buff.Bytes()
	return yield(res)
}

func WithJSONInput() workerOption {
	return func(configure *worker) error {
		configure.unmarshaler = json.Unmarshal
		return nil
	}
}

func WithYAMLInput() workerOption {
	return func(configure *worker) error {
		configure.unmarshaler = yaml.Unmarshal
		return nil
	}
}

func WithStringInput() workerOption {
	return func(configure *worker) error {
		configure.unmarshaler = func(in []byte, v any) error {
			switch vT := v.(type) {
			case *string:
				*vT = string(in)
			case *any:
				*vT = string(in)
			}
			return nil
		}
		return nil
	}
}

func WithTemplate(tmp *template.Template) workerOption {
	return func(configure *worker) error {
		configure.template = tmp
		return nil
	}
}

func WithTemplatePattern(pattern string) workerOption {
	return func(configure *worker) error {
		tmp, err := template.New("tenmplate_worker").Funcs(sprig.FuncMap()).Parse(pattern)
		if err != nil {
			return fmt.Errorf("couldn't parse go template pattern: %w", err)
		}
		configure.template = tmp
		return nil
	}
}
