package plugins

import (
	"encoding/json"
	"fmt"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/core/plugins/static/fileinput"
	"github.com/benji-bou/lugh/core/plugins/static/forward"
	"github.com/benji-bou/lugh/core/plugins/static/stdoutput"
	"github.com/benji-bou/lugh/core/plugins/static/transform"
)

func LoadPlugin(name string, path string, config any) (graph.IOWorker[[]byte], error) {
	plugin, err := getPlugin(name, path)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", name, err)
	}
	jsonConfig, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal config for plugin %s: %w", name, err)
	}
	if configurablePlugin, ok := plugin.(pluginapi.PluginConfigurer); ok {
		err = configurablePlugin.Config(jsonConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to configure plugin %s: %w", name, err)
		}
	}
	return plugin, nil
}

func getPlugin(name string, path string) (graph.IOWorker[[]byte], error) {
	var ioworker any
	switch name {
	case "forward":
		ioworker = forward.Worker[[]byte]()
	case "input", "fileinput":
		ioworker = fileinput.New()
	case "transform":
		ioworker = transform.New()
	case "output", "stdoutput":
		ioworker = stdoutput.Plugin{}
	default:
		var err error
		var runner pluginapi.Runner
		if path != "" {
			runner, err = grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
		} else {
			runner, err = grpc.NewPlugin(name).Connect()
		}
		if err != nil {
			return nil, err
		}
		ioworker = runner
	}
	switch ioworker.(type) {
	case graph.IOWorker[[]byte]:
		return ioworker.(graph.IOWorker[[]byte]), nil
	case graph.Worker[[]byte]:
		return graph.NewIOWorkerFromWorker(ioworker.(graph.Worker[[]byte])), nil
	case graph.Producer[[]byte]:
		return graph.NewIOWorkerFromProducer(ioworker.(graph.Producer[[]byte])), nil
	case graph.Consumer[[]byte]:
		return graph.NewIOWorkerFromConsumer(ioworker.(graph.Consumer[[]byte])), nil
	case graph.Runner[[]byte]:
		return graph.NewIOWorkerFromRunner(ioworker.(graph.Runner[[]byte])), nil
	default:
		return nil, fmt.Errorf("unknown plugin type: %T", ioworker)
	}
}
