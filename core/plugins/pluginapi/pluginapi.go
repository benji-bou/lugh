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

func NewWorker(plugin WorkerPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromWorker(plugin)}
}

func NewRunner(plugin RunnerIOPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromRunner(plugin)}
}

func NewProducer(plugin ProducerIOPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromProducer(plugin)}
}

func NewConsumer(plugin ConsumerIOPluginable) ioWorker {
	return ioWorker{plugin, graph.NewIOWorkerFromConsumer(plugin)}
}
