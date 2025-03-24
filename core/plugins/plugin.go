package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/core/plugins/static/fileinput"
	"github.com/benji-bou/lugh/core/plugins/static/forward"
	"github.com/benji-bou/lugh/core/plugins/static/stdoutput"
	"github.com/benji-bou/lugh/core/plugins/static/transform"
)

func LoadPlugin(name string, path string, config any) (pluginapi.ConfigurableIOWorker, error) {
	plugin, err := getPlugin(name, path)
	if err != nil {
		return pluginapi.ConfigurableIOWorker{}, fmt.Errorf("failed to load plugin %s: %w", name, err)
	}
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return pluginapi.ConfigurableIOWorker{}, fmt.Errorf("failed to marshal config for plugin %s: %w", name, err)
	}
	err = plugin.Config(jsonConfig)
	if err != nil {
		return pluginapi.ConfigurableIOWorker{}, fmt.Errorf("failed to configure plugin %s: %w", name, err)
	}
	return plugin, nil
}

func getPlugin(name string, path string) (pluginapi.ConfigurableIOWorker, error) {
	switch name {
	case "forward":
		return pluginapi.NewConfigurableIOWorker(forward.ForwardWorker[[]byte]()), nil
	case "input", "fileinput":
		return pluginapi.NewConfigurableWorker(fileinput.New()), nil
	case "transform":
		t := transform.New()
		return pluginapi.NewConfigurableWorker(t), nil
	case "output", "stdoutput":
		return pluginapi.NewConfigurableConsumer(stdoutput.Plugin{}), nil
	default:
		var err error
		var runner pluginapi.ConfigurableIORunner
		if path != "" {
			runner, err = grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
		} else {
			runner, err = grpc.NewPlugin(name).Connect()
		}
		if err != nil {
			return pluginapi.ConfigurableIOWorker{}, err
		}
		return pluginapi.NewConfigurableRunner(runner), nil
	}
}
