package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
	"github.com/benji-bou/SecPipeline/core/plugins/static/transform"
)

func LoadPlugin(name string, path string, config any) (pluginapi.IOWorkerPluginable, error) {
	plugin, err := getPluginBuilder(name, path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load plugin %s: %v", name, err)
	}
	jsonConfig, err := json.Marshal(config)
	if err != nil {

		return nil, fmt.Errorf("Failed to marshal config for plugin %s: %v", name, err)
	}
	err = plugin.Config(jsonConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to configure plugin %s: %v", name, err)

	}
	return plugin, nil
}

func getPluginBuilder(name string, path string) (pluginapi.IOWorkerPluginable, error) {
	switch name {
	case "transform":
		t := transform.New()
		return pluginapi.NewIOWorkerPluginFromSync(t), nil
	default:
		if path != "" {
			return grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
		}
		return grpc.NewPlugin(name).Connect()
	}

}
