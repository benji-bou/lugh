package plugin

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/benji-bou/SecPipeline/core/template"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
)

type EmptySecPlugin struct {
}

func (spp EmptySecPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (spp EmptySecPlugin) Config(config []byte) error {
	return nil
}

func (spp EmptySecPlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return nil, nil
}

type SecPipePluginOption = helper.OptionError[SecPipePlugin]

type SecPipePlugin struct {
	pipe   Pipeable
	plugin pluginctl.SecPipelinePluginable
}

func WithPlugin(plugin pluginctl.SecPipelinePluginable) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		configure.plugin = plugin
		return nil
	}
}

func WithPipe(pipe Pipeable) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		configure.pipe = pipe
		return nil
	}
}

func WithPluginNameAndPath(name string, path string) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		plugin, err := pluginctl.NewPlugin(name, pluginctl.WithPath(path)).Connect()
		if err != nil {
			return fmt.Errorf("failed to build plugin %s with  because: %w", name, err)
		}
		return WithPlugin(plugin)(configure)
	}
}

func WithPipeFromStage(st template.Stage) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		pipeCount := len(st.Pipe)
		if st.Pipe == nil || pipeCount == 0 {
			return nil
		}
		pipe, err := NewStagePipe(st)
		if err != nil {
			return err
		}
		configure.pipe = pipe
		return nil
	}
}

func WithPluginConfigFromStage(st template.Stage) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		if st.Config != nil && len(st.Config) > 0 {
			cJson, err := json.Marshal(st.Config)
			if err != nil {
				return fmt.Errorf("failed to marshal config %s  because: %w", st.Plugin, err)
			}
			return WithPluginConfig(cJson)(configure)
		}
		return nil
	}

}

func WithPluginConfig(config []byte) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		err := configure.Config(config)
		if err != nil {
			return fmt.Errorf("failed to configure plugin: %w", err)
		}
		return nil

	}
}

func WithStage(st template.Stage) SecPipePluginOption {
	return func(configure *SecPipePlugin) error {
		if err := WithPluginNameAndPath(st.Plugin, st.PluginPath)(configure); err != nil {
			return err
		}
		if err := WithPipeFromStage(st)(configure); err != nil {
			return err
		}
		if err := WithPluginConfigFromStage(st)(configure); err != nil {
			return err
		}
		return nil
	}
}

func NewSecPipePlugin(opt ...SecPipePluginOption) (pluginctl.SecPipelinePluginable, error) {
	//use Default Empty pipe (which can be overriden by options) ensuring default non nil pipe here
	secPlugin := SecPipePlugin{
		pipe: NewEmptyPipe(),
	}
	return helper.ConfigureWithError(secPlugin, opt...)
}

func (spp SecPipePlugin) GetInputSchema() ([]byte, error) {
	return spp.plugin.GetInputSchema()
}

func (spp SecPipePlugin) Config(config []byte) error {
	return spp.plugin.Config(config)
}

func (spp SecPipePlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	pipeOutputC, _ := spp.pipe.Pipe(ctx, input)
	return spp.plugin.Run(ctx, pipeOutputC)
}
