package load

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
)

// GRPC Load a grpc plugin from a name and a path.
func GRPC(name string, path string) (any, error) {
	var runner any
	var err error
	if path != "" {
		runner, err = grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
	} else {
		runner, err = grpc.NewPlugin(name).Connect()
	}
	if err != nil {
		return nil, err
	}

	return runner, nil
}

func ConfigPlugin(pluginConfigurer pluginapi.PluginConfigurer, config any) error {
	if config == nil {
		return nil
	}
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}
	return pluginConfigurer.Config(configBytes)
}

func Worker(name string, path string, config any) (graph.IOWorker[[]byte], error) {
	return getLoader().Load(name, path, config)
}

func RegisterDefault(defaultLoader Loadable) {
	getLoader().RegisterDefault(defaultLoader)
}

func Register(name string, loader Loadable, optionalName ...string) {
	getLoader().Register(name, loader)
	for _, n := range optionalName {
		getLoader().Register(n, loader)
	}
}

// Configure is a helper function to create a Loadable that will load a plugin using input `loader` func  and configure the result if it implements PluginConfigurer.
func Configure(loader func(name, path string) (any, error)) Loadable {
	return LoaderFunc(func(name, path string, config any) (any, error) {
		plugin, err := loader(name, path)
		if err != nil {
			return nil, err
		}
		if configurablePlugin, ok := plugin.(pluginapi.PluginConfigurer); ok {
			if err := ConfigPlugin(configurablePlugin, config); err != nil {
				return nil, err
			}
		}
		return plugin, nil
	})
}

func Default(loader func() any) Loadable {
	return Configure(func(name, path string) (any, error) {
		return loader(), nil
	})
}

// Loadable is an interface that defines a method to load a plugin.
type Loadable interface {
	Load(name string, path string, config any) (any, error)
}

type LoaderFunc func(name string, path string, config any) (any, error)

func (f LoaderFunc) Load(name string, path string, config any) (any, error) {
	return f(name, path, config)
}

// loader is internal struct to manage plugins loaders.
type loader struct {
	pluginsLoader map[string]Loadable
	defaultLoader Loadable
	rwMutex       sync.RWMutex
}

var onceValueLoader func() *loader

// getLoader returns the singleton instance of loader.
func getLoader(defaultLoader ...Loadable) *loader {
	if onceValueLoader == nil {
		onceValueLoader = sync.OnceValue(func() *loader {
			var defLoader Loadable = Configure(func(name, path string) (any, error) { return GRPC(name, path) })
			if len(defaultLoader) > 0 {
				defLoader = defaultLoader[0]
			}
			fmt.Printf("default loader: %T\n", defLoader)
			return &loader{
				pluginsLoader: make(map[string]Loadable),
				defaultLoader: defLoader,
				rwMutex:       sync.RWMutex{},
			}
		})
	}
	return onceValueLoader()
}

// RegisterDefault registers the default loader. when no plugin name will match
func (l *loader) RegisterDefault(loader Loadable) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()
	l.defaultLoader = loader
}

// Register registers a new plugin loader.
func (l *loader) Register(name string, loader Loadable) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()
	l.pluginsLoader[name] = loader
}

// Load loads a IOWorker by name and path and config.
func (l *loader) Load(name string, path string, config any) (graph.IOWorker[[]byte], error) {
	l.rwMutex.RLock()
	defer l.rwMutex.RUnlock()
	loader, ok := l.pluginsLoader[name]
	if !ok {
		slog.Info("plugin loader not found, using default loader", "plugin", name)
		loader = l.defaultLoader
	}
	plugin, err := loader.Load(name, path, config)
	if err != nil {
		return nil, fmt.Errorf("failed to load plugin %s: %w", name, err)
	}
	return l.convertPluginToIOWorker(plugin)
}

// convertPluginToIOWorker converts a plugin to a IOWorker.
func (l *loader) convertPluginToIOWorker(plugin any) (graph.IOWorker[[]byte], error) {
	switch plugin.(type) {
	case graph.IOWorker[[]byte]:
		return plugin.(graph.IOWorker[[]byte]), nil
	case graph.Worker[[]byte]:
		return graph.NewIOWorkerFromWorker(plugin.(graph.Worker[[]byte])), nil
	case graph.Producer[[]byte]:
		return graph.NewIOWorkerFromProducer(plugin.(graph.Producer[[]byte])), nil
	case graph.Consumer[[]byte]:
		return graph.NewIOWorkerFromConsumer(plugin.(graph.Consumer[[]byte])), nil
	case graph.Runner[[]byte]:
		return graph.NewIOWorkerFromRunner(plugin.(graph.Runner[[]byte])), nil
	default:
		return nil, fmt.Errorf("unknown plugin type: %T", plugin)
	}
}
