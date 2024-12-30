package pluginapi

import (
	"github.com/benji-bou/lugh/core/graph"
)

type PluginConfigurer interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}

type IOWorkerPluginable interface {
	PluginConfigurer
	graph.IOWorker[[]byte]
}

type WorkerPluginable interface {
	PluginConfigurer
	graph.Worker[[]byte]
}

type RunnerIOPluginable interface {
	PluginConfigurer
	graph.Runner[[]byte]
}

type ProducerIOPluginable interface {
	PluginConfigurer
	graph.Producer[[]byte]
}

type ConsumerIOPluginable interface {
	PluginConfigurer
	graph.Consumer[[]byte]
}

type ioWorker struct {
	PluginConfigurer
	graph.IOWorker[[]byte]
}

func NewIOWorkerPluginFromWorker(plugin WorkerPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromWorker(plugin)}
}

func NewIOWorkerPluginFromRunner(plugin RunnerIOPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromRunner(plugin)}
}

func NewIOWorkerPluginFromProducer(plugin ProducerIOPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromProducer(plugin)}
}

func NewIOWorkerPluginFromConsumer(plugin ConsumerIOPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromConsumer(plugin)}
}
