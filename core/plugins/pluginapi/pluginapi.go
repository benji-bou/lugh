package pluginapi

import (
	"github.com/benji-bou/SecPipeline/core/graph"
)

type PluginConfigurer interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}

type IOWorkerPluginable interface {
	PluginConfigurer
	graph.IOWorker[[]byte]
}

type SyncWorkerPluginable interface {
	PluginConfigurer
	graph.Worker[[]byte]
}

type RunIOPluginable interface {
	PluginConfigurer
	graph.Runner[[]byte]
}

type ioWorker struct {
	PluginConfigurer
	graph.IOWorker[[]byte]
}

func NewIOWorkerPluginFromSync(plugin SyncWorkerPluginable, name string) IOWorkerPluginable {
	return ioWorker{plugin, graph.NewIOWorkerFromWorker(plugin)}
}

func NewIOWorkerPluginFromRunner(plugin RunIOPluginable) IOWorkerPluginable {
	return ioWorker{plugin, graph.NewIOWorkerFromRunner(plugin)}
}
