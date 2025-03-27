package plugins

import (
	"fmt"

	"github.com/benji-bou/lugh/core/plugins/load"
	"github.com/benji-bou/lugh/core/plugins/static/data/base64"
	"github.com/benji-bou/lugh/core/plugins/static/data/forward"
	"github.com/benji-bou/lugh/core/plugins/static/data/insert"
	"github.com/benji-bou/lugh/core/plugins/static/data/regex"
	"github.com/benji-bou/lugh/core/plugins/static/data/split"
	"github.com/benji-bou/lugh/core/plugins/static/data/template"
	"github.com/benji-bou/lugh/core/plugins/static/fileinput"
	"github.com/benji-bou/lugh/core/plugins/static/pipe"
	"github.com/benji-bou/lugh/core/plugins/static/stdoutput"
	"github.com/mitchellh/mapstructure"
)

func InitLoader() {
	load.RegisterDefault(load.Configure(load.GRPC))
	load.Register("forward", load.Default(func() any {
		return forward.Worker[[]byte]()
	}))
	load.Register("fileinput", load.Default(func() any {
		return fileinput.New()
	}))
	load.Register("pipe", load.Configure(func(name, path string) (any, error) {
		return pipe.New(path), nil
	}), "transform")
	load.Register("output", load.Default(func() any {
		return stdoutput.New()
	}), "stdoutput")
	load.Register("split", load.LoaderFunc(func(name, path string, config any) (any, error) {
		sep := "\n"
		if configMap, ok := config.(map[string]any); ok {
			if s, ok := configMap["sep"].(string); ok {
				sep = s
			}
		}
		return split.Worker(sep), nil
	}))
	load.Register("base64", load.Default(func() any {
		return base64.Base64Decode()
	}))
	load.Register("insert", load.LoaderFunc(func(name, path string, config any) (any, error) {
		insertStr := "\n"
		if configMap, ok := config.(map[string]any); ok {
			if s, ok := configMap["content"].(string); ok {
				insertStr = s
			}
		}
		return insert.Worker(insertStr), nil
	}))

	load.Register("regex", load.LoaderFunc(func(name, path string, config any) (any, error) {
		var regConfig regex.Config
		if err := mapstructure.Decode(config, &regConfig); err != nil {
			return nil, fmt.Errorf("regex loader: decode config: %w", err)
		}
		return regex.Worker(regConfig)
	}))

	load.Register("template", load.LoaderFunc(func(name, path string, config any) (any, error) {
		var tplConfig template.Config
		if err := mapstructure.Decode(config, &tplConfig); err != nil {
			return nil, fmt.Errorf("template loader: decode config: %w", err)
		}
		return template.Worker(tplConfig)
	}), "goTemplate")
}
