package pluginapi

import (
	"github.com/benji-bou/lugh/core/graph"
)

type PluginConfigurer interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}

type ConfigurableWorker interface {
	PluginConfigurer
	graph.Worker[[]byte]
}

type ConfigurableIORunner interface {
	PluginConfigurer
	graph.Runner[[]byte]
}

type ConfigurableIOProducer interface {
	PluginConfigurer
	graph.Producer[[]byte]
}

type ConfigurableIOConsumer interface {
	PluginConfigurer
	graph.Consumer[[]byte]
}

type ConfigurableIOWorker struct {
	PluginConfigurer
	graph.IOWorker[[]byte]
}

func NewConfigurableWorker(plugin ConfigurableWorker) ConfigurableIOWorker {
	return NewConfigurableIOWorker(graph.NewIOWorkerFromWorker(plugin), WithConfigurer(plugin))
}

func NewConfigurableRunner(plugin ConfigurableIORunner) ConfigurableIOWorker {
	return NewConfigurableIOWorker(graph.NewIOWorkerFromRunner(plugin), WithConfigurer(plugin))
}

func NewConfigurableProducer(plugin ConfigurableIOProducer) ConfigurableIOWorker {
	return NewConfigurableIOWorker(graph.NewIOWorkerFromProducer(plugin), WithConfigurer(plugin))
}

func NewConfigurableConsumer(plugin ConfigurableIOConsumer) ConfigurableIOWorker {
	return NewConfigurableIOWorker(graph.NewIOWorkerFromConsumer(plugin), WithConfigurer(plugin))
}

type emptyConfigurer struct{}

func (ec emptyConfigurer) GetInputSchema() ([]byte, error) { return nil, nil }
func (ec emptyConfigurer) Config(config []byte) error      { return nil }

func NewConfigurableIOWorker(worker graph.IOWorker[[]byte], opt ...func(*ConfigurableIOWorker)) ConfigurableIOWorker {
	ioWorker := ConfigurableIOWorker{PluginConfigurer: emptyConfigurer{}, IOWorker: worker}
	for _, o := range opt {
		o(&ioWorker)
	}
	return ioWorker
}

func WithConfigurer(configurer PluginConfigurer) func(*ConfigurableIOWorker) {
	return func(iw *ConfigurableIOWorker) {
		iw.PluginConfigurer = configurer
	}
}
