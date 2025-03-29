package pipe

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/benji-bou/diwo"
	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/load"
	"gopkg.in/yaml.v3"
)

type Plugin struct {
	defaultPluginsPath string
	ioworkers          []graph.IOWorker[[]byte]
}

func New(pluginPath string) *Plugin {
	return &Plugin{defaultPluginsPath: pluginPath}
}

func (*Plugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (p *Plugin) ExtractPluginPath(config any) string {
	configMap, ok := config.(map[string]any)
	if !ok {
		return p.defaultPluginsPath
	}
	pluginPath, ok := configMap["pluginPath"]
	if !ok {
		return p.defaultPluginsPath
	}
	pluginPathStr, ok := pluginPath.(string)
	if !ok {
		return p.defaultPluginsPath
	}
	return pluginPathStr
}

func (p *Plugin) Config(config []byte) error {
	decodedConfig := []map[string]map[string]any{}
	err := yaml.Unmarshal(config, &decodedConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	p.ioworkers = make([]graph.IOWorker[[]byte], 0, len(decodedConfig))
	for _, pluginYAMLConfigs := range decodedConfig {
		if len(pluginYAMLConfigs) > 1 {
			return errors.New("a transformer can only have one config per step")
		}
		//nolint:gocritic // just test
		for pluginName, pluginConfig := range pluginYAMLConfigs {
			pluginPath := p.ExtractPluginPath(pluginConfig)
			subplugin, err := load.Worker(pluginName, pluginPath, pluginConfig)
			if err != nil {
				log.Fatalf("load plugin: %s, %v", pluginName, err)
				return nil
			}
			p.ioworkers = append(p.ioworkers, subplugin)
		}
	}
	return nil
}

func (p *Plugin) Run(ctx context.Context, input <-chan []byte, yield func(elem []byte, err error) error) error {
	if input == nil {
		return errors.New("running pipe plugin: input is nil")
	}
	syncCtx := graph.NewContext(ctx)
	errsC := make([]<-chan error, len(p.ioworkers))
	for _, iow := range p.ioworkers {
		iow.SetInput(input)
		input = iow.Output()
		errsC = append(errsC, iow.Run(syncCtx))
	}
	syncCtx.Synchronize()
	errC := diwo.Merge(errsC...)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case err := <-errC:
			if err := yield(nil, err); err != nil {
				return err
			}
		case elem, ok := <-input:
			if !ok {
				return nil
			}
			if err := yield(elem, nil); err != nil {
				return err
			}
		}
	}
}
