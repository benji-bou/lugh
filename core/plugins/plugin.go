package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/core/plugins/static/fileinput"
	"github.com/benji-bou/lugh/core/plugins/static/stdoutput"
	"github.com/benji-bou/lugh/core/plugins/static/transform"
)

func LoadPlugin(name string, path string, config any) (pluginapi.IOWorkerPluginable, error) {
	plugin, err := getPlugin(name, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", name, err)
	}
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config for plugin %s: %w", name, err)
	}
	err = plugin.Config(jsonConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to configure plugin %s: %w", name, err)
	}
	return plugin, nil
}

func getPlugin(name string, path string) (pluginapi.IOWorkerPluginable, error) {
	switch name {
	case "input", "fileinput":
		return pluginapi.NewWorker(fileinput.New()), nil
	case "transform":
		t := transform.New()
		return pluginapi.NewWorker(t), nil
	case "output", "stdoutput":
		return pluginapi.NewConsumer(stdoutput.Plugin{}), nil
	default:
		var err error
		var runner pluginapi.RunnerIOPluginable
		if path != "" {
			runner, err = grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
		} else {
			runner, err = grpc.NewPlugin(name).Connect()
		}
		if err != nil {
			return nil, err
		}
		return pluginapi.NewRunner(runner), nil
	}
}
