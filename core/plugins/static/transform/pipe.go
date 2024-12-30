package transform

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"github.com/Masterminds/sprig/v3"

	"github.com/benji-bou/lugh/helper"
	"gopkg.in/yaml.v3"
)

type goTemplateOption = helper.OptionError[goTemplate]

type goTemplate struct {
	template    *template.Template
	unmarshaler func(in []byte, v any) error
}

func (gt goTemplate) Transform(in []byte) ([][]byte, error) {

	var dataInput interface{}
	err := gt.unmarshaler(in, &dataInput)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal input: %w", err)
	}
	buff := &bytes.Buffer{}
	err = gt.template.ExecuteTemplate(buff, gt.template.Name(), dataInput)
	if err != nil {
		return nil, fmt.Errorf("failed to execute template: %w", err)
	}
	return [][]byte{buff.Bytes()}, nil

}

func WithJsonInput() goTemplateOption {
	return func(configure *goTemplate) error {
		configure.unmarshaler = json.Unmarshal
		return nil
	}
}

func WithYamlInput() goTemplateOption {
	return func(configure *goTemplate) error {
		configure.unmarshaler = yaml.Unmarshal
		return nil
	}
}

func WithStringInput() goTemplateOption {
	return func(configure *goTemplate) error {
		configure.unmarshaler = func(in []byte, v any) error {
			switch vT := v.(type) {
			case *interface{}:
				*vT = string(in)
			}
			return nil
		}
		return nil
	}
}

func WithTemplate(tmp *template.Template) goTemplateOption {
	return func(configure *goTemplate) error {
		configure.template = tmp
		return nil
	}
}

func WithTemplatePattern(pattern string) goTemplateOption {
	return func(configure *goTemplate) error {
		tmp, err := template.New("GoTemplatePattern").Funcs(sprig.FuncMap()).Parse(pattern)
		if err != nil {
			return fmt.Errorf("couldn't parse go template pattern: %w", err)
		}
		configure.template = tmp
		return nil
	}
}

type GoTemplateConfig struct {
	Format  string `yaml:"format"`
	Pattern string `yaml:"pattern"`
}

func GoTemplateWithConfig(config GoTemplateConfig) (Transformer, error) {
	opt := []goTemplateOption{}
	switch config.Format {
	case "string":
		opt = append(opt, WithStringInput())
	case "yaml":
		opt = append(opt, WithYamlInput())
	default:
		opt = append(opt, WithJsonInput())
	}
	opt = append(opt, WithTemplatePattern(config.Pattern))
	return GoTemplate(opt...)
}

func GoTemplate[OPT goTemplateOption](opt ...OPT) (Transformer, error) {
	tmp, err := helper.ConfigureWithError(goTemplate{}, opt...)
	if err != nil {
		return nil, fmt.Errorf("failed to generate go template Transform, fallback to default Transform: %w", err)
	}
	if tmp.template == nil {
		return forwardTransform(), nil
	}
	return tmp, nil
}

func Base64Decode() Transformer {
	return Map(func(elem []byte) ([]byte, error) {
		res, err := base64.RawStdEncoding.DecodeString(string(elem))
		if err != nil {
			return elem, fmt.Errorf("unable to decode base64: %w", err)
		}
		return res, nil
	})
}

func Regex(regexpPattern string, n int) (Transformer, error) {
	regex, err := regexp.Compile(regexpPattern)
	if err != nil {
		return nil, fmt.Errorf("failed to compile regexp pattern %s: %w", regexpPattern, err)
	}
	return Map(func(elem []byte) ([]byte, error) {
		resAllArr := regex.FindAllSubmatch(elem, -1)
		resArr := make([][]byte, 0, len(resAllArr))
		for _, match := range resAllArr {
			if len(match) == 0 {
				continue
			} else if n >= len(match) {
				resArr = append(resArr, match[0])
			} else {
				resArr = append(resArr, match[n])
			}
		}
		res := bytes.Join(resArr, []byte("\n"))
		return res, nil
	}), nil
}

func InsertString(insert string) Transformer {
	return Map(func(elem []byte) ([]byte, error) {
		res := append(elem, []byte(insert)...)
		return res, nil
	})
}

func Split(sep string) Transformer {
	return Func(func(elem []byte) ([][]byte, error) {
		inputStr := string(elem)
		res := strings.Split(inputStr, sep)
		return helper.Map(res, func(elem string) []byte { return []byte(elem) }), nil
	})
}

func NewTransform(name string, config yaml.Node) (Transformer, error) {
	switch name {
	case "goTemplate":
		configGoTpl := GoTemplateConfig{}
		err := config.Decode(&configGoTpl)
		if err != nil {
			return nil, fmt.Errorf("failed to build goTemplate Transform because: %w", err)
		}
		return GoTemplateWithConfig(configGoTpl)
	case "base64":
		return Base64Decode(), nil
	case "regex":
		configRegex := struct {
			Pattern string `yaml:"pattern"`
			Select  int    `yaml:"select"`
		}{}

		err := config.Decode(&configRegex)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex Transform because: %w", err)
		}
		return Regex(configRegex.Pattern, configRegex.Select)
	case "insert":
		configInsert := struct {
			Content string `yaml:"content"`
		}{}
		err := config.Decode(&configInsert)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex Transform because: %w", err)
		}
		return InsertString(configInsert.Content), nil
	case "split":
		configSplit := struct {
			Content string `yaml:"sep"`
		}{}
		err := config.Decode(&configSplit)
		if err != nil {
			return nil, fmt.Errorf("failed to build regex Transform because: %w", err)
		}
		return Split(configSplit.Content), nil
	default:
		return forwardTransform(), errors.New("failed to build sectpl. Template unknown")
	}
}

func errTransform(err error) Transformer {
	return Map(func([]byte) ([]byte, error) {
		return nil, err
	})
}

func forwardTransform() Transformer {
	return Map(func(input []byte) ([]byte, error) {
		return input, nil
	})
}
