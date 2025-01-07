package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
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
	case "transform":
		t := transform.New()
		return pluginapi.NewIOWorkerPluginFromWorker(t), nil
	case "output", "stdoutput":
		return pluginapi.NewIOWorkerPluginFromConsumer(stdoutput.Plugin{}), nil
	default:
		if path != "" {
			return grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
		}
		return grpc.NewPlugin(name).Connect()
	}
}
