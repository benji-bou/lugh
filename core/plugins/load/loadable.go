package load

import (
	"fmt"

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

func ConfigAsMap(loaderWithConfigAsMap func(name, path string, config map[string]any) (any, error)) Loadable {
	return LoaderFunc(func(name, path string, config any) (any, error) {
		if configAsMap, ok := config.(map[string]any); ok {
			return loaderWithConfigAsMap(name, path, configAsMap)
		}
		return nil, fmt.Errorf("config is not a map[string]any")
	})
}

func Get(pluginGetter func() any) Loadable {
	return Configure(func(name, path string) (any, error) {
		return pluginGetter(), nil
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

type MiddlewareLoader func(next Loadable) Loadable
