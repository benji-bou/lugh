package plugins

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
)

type IOWorkerPlugin struct {
	decorated pluginapi.IOPluginable
	graph.DefaultIOWorker[[]byte]
}

func NewIOWorkerPlugin(decorated pluginapi.IOPluginable) *IOWorkerPlugin {
	return &IOWorkerPlugin{
		decorated: decorated,
	}
}

func (wp *IOWorkerPlugin) Run(ctx context.Context) <-chan error {
	outputC, errC := wp.decorated.Run(ctx, wp.GetInput())
	wp.SetOutput(outputC)
	wp.UpdateRun(true)
	return errC
}

func LoadPlugin(name string, path string, config map[string]any) (pluginapi.IOPluginable, error) {
	plugin, err := getPluginBuilder(name, path)
	if err != nil {
		return nil, fmt.Errorf("Failed to load plugin %s: %v", name, err)
	}
	jsonConfig, err := json.Marshal(config)
	if err != nil {

		return nil, fmt.Errorf("Failed to marshal config for plugin %s: %v", name, err)
	}
	err = plugin.Config(jsonConfig)
	if err != nil {
		return nil, fmt.Errorf("Failed to configure plugin %s: %v", name, err)

	}
	return plugin, nil
}

func getPluginBuilder(name string, path string) (pluginapi.IOPluginable, error) {
	switch name {
	case "forward":
		return ForwardPlugin{}, nil
	default:
		if path != "" {
			return grpc.NewPlugin(name, grpc.WithPath(path)).Connect()
		}
		return grpc.NewPlugin(name).Connect()
	}

}

// import (
// 	"context"
// 	"log/slog"
// 	"os"

// 	"github.com/benji-bou/SecPipeline/helper"
// 	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
// 	"github.com/benji-bou/chantools"
// )

type EmptyPlugin struct {
}

func (spp EmptyPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (spp EmptyPlugin) Config(config []byte) error {
	return nil
}

func (spp EmptyPlugin) Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	return nil, nil
}

// type SecPipePluginOption = helper.Option[SecPipePlugin]

// type SecPipePlugin struct {
// 	grpc.SecPluginable
// 	pipe Pipeable
// }

// func WithPlugin(plugin grpc.SecPluginable) SecPipePluginOption {
// 	return func(configure *SecPipePlugin) {
// 		configure.SecPluginable = plugin
// 	}
// }

// func WithPipe(pipe Pipeable) SecPipePluginOption {
// 	return func(configure *SecPipePlugin) {
// 		configure.pipe = pipe
// 	}
// }

// func WithPluginNameAndPath(name string, path string) SecPipePluginOption {

// 	plugin, err :=
// 	if err != nil {
// 		// This is a fast-fail: Exit the program due to failing in plugin initialisation
// 		slog.Error("failed to initilize plugin", "name", name, "path", path)
// 		os.Exit(-1)
// 	}
// 	return WithPlugin(plugin)
// }

// func NewSecPipePlugin(opt ...SecPipePluginOption) grpc.SecPluginable {
// 	//use Default Empty pipe (which can be overriden by options) ensuring default non nil pipe here
// 	secPlugin := SecPipePlugin{
// 		pipe: NewEmptyPipe(),
// 	}
// 	return helper.Configure(secPlugin, opt...)
// }

// func (spp SecPipePlugin) Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
// 	pipeOutputC, _ := spp.pipe.Pipe(ctx, input)
// 	return spp.SecPluginable.Run(ctx, pipeOutputC)
// }

// type RawOutputPluginOption = helper.Option[RawOutputPlugin]

// func WithCallback(cb func(data []byte)) RawOutputPluginOption {
// 	return func(configure *RawOutputPlugin) {
// 		configure.cb = cb
// 	}
// }

// // WithChannel: when this option is set you pass a writable channel which will be used to write the input of the plugin into it
// // in this case RawOutputPlugin takes ownership of the channel and close it when it is done writing it. Means the plugin do not received input anymore
// func WithChannel(dataC chan<- []byte) RawOutputPluginOption {
// 	return func(configure *RawOutputPlugin) {
// 		configure.dataC = dataC
// 	}
// }

// func NewRawOutputPlugin(opt ...RawOutputPluginOption) RawOutputPlugin {

// 	rop := helper.Configure(RawOutputPlugin{}, opt...)
// 	rop.NoOutputPlugin = NewNoOutputPlugin(WithInputWorker(func(input <-chan []byte) {
// 		if rop.dataC != nil {
// 			defer close(rop.dataC)
// 		}
// 		for data := range input {
// 			if rop.cb != nil {
// 				rop.cb(data)
// 			}
// 			if rop.dataC != nil {
// 				rop.dataC <- data
// 			}
// 		}
// 	}))
// 	return rop
// }

// type RawOutputPlugin struct {
// 	NoOutputPlugin
// 	cb    func(data []byte)
// 	dataC chan<- []byte
// }

// type NoOutputPluginOption = helper.Option[NoOutputPlugin]

// func WithInputWorker(worker func(input <-chan []byte)) NoOutputPluginOption {
// 	return func(configure *NoOutputPlugin) {
// 		configure.worker = worker
// 	}
// }

// func NewNoOutputPlugin(opt ...NoOutputPluginOption) NoOutputPlugin {
// 	return helper.Configure(NoOutputPlugin{}, opt...)
// }

// type NoOutputPlugin struct {
// 	EmptyPlugin
// 	worker func(input <-chan []byte)
// }

// func (nop NoOutputPlugin) Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
// 	return chantools.NewWithErr(func(c chan<- []byte, eC chan<- error, params ...any) {
// 		if nop.worker != nil {
// 			nop.worker(input)
// 		}
// 	})
// }

type ForwardPlugin struct {
	EmptyPlugin
}

func (fp ForwardPlugin) Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	return input, make(<-chan error)
}

// func NewOnlyPipePlugin(pipe Pipeable) pluginapi.IOPluginable {
// 	return NewSecPipePlugin(WithPipe(pipe), WithPlugin(ForwardPlugin{}))
// }
