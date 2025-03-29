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
	"github.com/benji-bou/lugh/core/plugins/static/include"
	"github.com/benji-bou/lugh/core/plugins/static/pipe"
	"github.com/benji-bou/lugh/core/plugins/static/stdoutput"
	tpl "github.com/benji-bou/lugh/core/template"
	"github.com/mitchellh/mapstructure"
)

func InitLoader() {
	load.RegisterDefault(load.Configure(load.GRPC))
	load.Register("forward", load.Get(func() any {
		return forward.Worker[[]byte]()
	}))
	load.Register("fileinput", load.Get(func() any {
		return fileinput.New()
	}))
	load.Register("pipe", load.Configure(func(name, path string) (any, error) {
		return pipe.New(path), nil
	}), "transform")
	load.Register("output", load.Get(func() any {
		return stdoutput.New()
	}), "stdoutput")
	load.Register("split", load.ConfigAsMap(func(name, path string, config map[string]any) (any, error) {
		sep := "\n"
		if s, ok := config["sep"].(string); ok {
			sep = s
		}
		return split.Worker(sep), nil
	}))
	load.Register("base64", load.Get(func() any {
		return base64.Base64Decode()
	}))
	load.Register("insert", load.ConfigAsMap(func(name, path string, config map[string]any) (any, error) {
		insertStr := "\n"
		if s, ok := config["content"].(string); ok {
			insertStr = s
		}
		return insert.Worker(insertStr), nil
	}))

	load.Register("regex", load.ConfigAsMap(func(name, path string, config map[string]any) (any, error) {
		var regConfig regex.Config
		if err := mapstructure.Decode(config, &regConfig); err != nil {
			return nil, fmt.Errorf("regex loader: decode config: %w", err)
		}
		return regex.Worker(regConfig)
	}))

	load.Register("template", load.ConfigAsMap(func(name, path string, config map[string]any) (any, error) {
		var tplConfig template.Config
		if err := mapstructure.Decode(config, &tplConfig); err != nil {
			return nil, fmt.Errorf("template loader: decode config: %w", err)
		}
		return template.Worker(tplConfig)
	}), "goTemplate")

	load.Register("include", load.ConfigAsMap(func(name, path string, config map[string]any) (any, error) {
		var includeConfig include.Config
		if err := mapstructure.Decode(config, &includeConfig); err != nil {
			return nil, fmt.Errorf("include loader: decode config: %w", err)
		}
		return include.Worker[tpl.Stage](includeConfig)
	}))
}
