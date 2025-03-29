package load

// Package load provides a way to load plugins. It expose methods to register and load plugins.
// It also provides a way to wrap loaders to add behavior to the loading process.
// It is used by the template package to load plugins from templates.
// You register loader by providing a name and a loader function. The loader function is called when a plugin of the requested name is load.
// Providing to the loader name of the plugin to load, the path of the plugin and the configuration of the plugin.
// The loader function should return the plugin or an error if the plugin could not be loaded.
// You can registeer a default loader that will be used if no loader is found for the requested plugin name. By default the default loader is the GRPC Loader
// The loaded plugin can either be
//    - graph.IOWorker[[]byte
//    - graph.Worker[[]byte]
//    - graph.Producer[[]byte]
//    - graph.Consumer[[]byte]
//    - graph.Runner[[]byte]
// if it is not one of these types, the loader will return an error. `ErrPluginTypeNotSupported`

// This package expose Loadable interface that is used to load plugins. You register Loadable to load your plugins.
// This package provide helper function that warp Loadable to add behavior to the loading process. Such as `Configure`
// which automatically call the `Config` method of the plugin if it implements `pluginapi.PluginConfigurer`.

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
)

var ErrPluginTypeNotSupported = fmt.Errorf("plugin type not supported")

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
	return Default().Load(name, path, config)
}

func RegisterDefault(defaultLoader Loadable) {
	Default().RegisterDefault(defaultLoader)
}

func Register(name string, loader Loadable, optionalName ...string) {
	Default().Register(name, loader)
	for _, n := range optionalName {
		Default().Register(n, loader)
	}
}

// Loader is internal struct to manage plugins loaders.
type Loader struct {
	pluginsLoader map[string]Loadable
	defaultLoader Loadable
	rwMutex       sync.RWMutex
}

var onceValueLoader func() *Loader

// Default returns the singleton instance of loader.
func Default(defaultLoader ...Loadable) *Loader {
	if onceValueLoader == nil {
		onceValueLoader = sync.OnceValue(func() *Loader {
			var defLoader Loadable = Configure(func(name, path string) (any, error) { return GRPC(name, path) })
			if len(defaultLoader) > 0 {
				defLoader = defaultLoader[0]
			}
			fmt.Printf("default loader: %T\n", defLoader)
			return &Loader{
				pluginsLoader: make(map[string]Loadable),
				defaultLoader: defLoader,
				rwMutex:       sync.RWMutex{},
			}
		})
	}
	return onceValueLoader()
}

// RegisterDefault registers the default loader. when no plugin name will match
func (l *Loader) RegisterDefault(loader Loadable) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()
	l.defaultLoader = loader
}

// Register registers a new plugin loader.
func (l *Loader) Register(name string, loader Loadable) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()
	l.pluginsLoader[name] = loader
}

// Load loads a IOWorker by name and path and config.
func (l *Loader) Load(name string, path string, config any) (graph.IOWorker[[]byte], error) {
	l.rwMutex.RLock()
	loader, ok := l.pluginsLoader[name]
	l.rwMutex.RUnlock()
	if !ok {
		slog.Info("plugin loader not found, using default loader", "plugin", name)
		loader = l.defaultLoader
	}
	plugin, err := loader.Load(name, path, config)
	if err != nil {
		return nil, fmt.Errorf("plugin loader %s: %w", name, err)
	}
	return l.convertPluginToIOWorker(plugin)
}

// WrapLoader wraps a plugin loader with a middleware. This is useful for adding data to the plugin config. or override the result graph.IOWorker[[]byte]
func (l *Loader) WrapLoader(name string, next MiddlewareLoader) {
	l.rwMutex.Lock()
	defer l.rwMutex.Unlock()
	loader, ok := l.pluginsLoader[name]
	if !ok {
		slog.Info("plugin loader not found, using default loader", "plugin", name)
		loader = l.defaultLoader
	}
	l.pluginsLoader[name] = next(loader)
}

// convertPluginToIOWorker converts a plugin to a IOWorker.
func (l *Loader) convertPluginToIOWorker(plugin any) (graph.IOWorker[[]byte], error) {
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
		return nil, fmt.Errorf("%w: type given is %T, ", ErrPluginTypeNotSupported, plugin)
	}
}
