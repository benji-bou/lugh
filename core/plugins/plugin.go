package plugins

type IOWorkerPluginable interface {
	graph.IOWorker
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}

// import (
// 	"context"
// 	"log/slog"
// 	"os"

// 	"github.com/benji-bou/SecPipeline/helper"
// 	"github.com/benji-bou/SecPipeline/pluginctl"
// 	"github.com/benji-bou/chantools"
// )

// type EmptySecPlugin struct {
// }

// func (spp EmptySecPlugin) GetInputSchema() ([]byte, error) {
// 	return nil, nil
// }

// func (spp EmptySecPlugin) Config(config []byte) error {
// 	return nil
// }

// func (spp EmptySecPlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
// 	return nil, nil
// }

// type SecPipePluginOption = helper.Option[SecPipePlugin]

// type SecPipePlugin struct {
// 	pluginctl.SecPluginable
// 	pipe Pipeable
// }

// func WithPlugin(plugin pluginctl.SecPluginable) SecPipePluginOption {
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

// func NewSecPipePlugin(opt ...SecPipePluginOption) pluginctl.SecPluginable {
// 	//use Default Empty pipe (which can be overriden by options) ensuring default non nil pipe here
// 	secPlugin := SecPipePlugin{
// 		pipe: NewEmptyPipe(),
// 	}
// 	return helper.Configure(secPlugin, opt...)
// }

// func (spp SecPipePlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
// 	pipeOutputC, _ := spp.pipe.Pipe(ctx, input)
// 	return spp.SecPluginable.Run(ctx, pipeOutputC)
// }

// type RawOutputPluginOption = helper.Option[RawOutputPlugin]

// func WithCallback(cb func(data *pluginctl.DataStream)) RawOutputPluginOption {
// 	return func(configure *RawOutputPlugin) {
// 		configure.cb = cb
// 	}
// }

// // WithChannel: when this option is set you pass a writable channel which will be used to write the input of the plugin into it
// // in this case RawOutputPlugin takes ownership of the channel and close it when it is done writing it. Means the plugin do not received input anymore
// func WithChannel(dataC chan<- *pluginctl.DataStream) RawOutputPluginOption {
// 	return func(configure *RawOutputPlugin) {
// 		configure.dataC = dataC
// 	}
// }

// func NewRawOutputPlugin(opt ...RawOutputPluginOption) RawOutputPlugin {

// 	rop := helper.Configure(RawOutputPlugin{}, opt...)
// 	rop.NoOutputPlugin = NewNoOutputPlugin(WithInputWorker(func(input <-chan *pluginctl.DataStream) {
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
// 	cb    func(data *pluginctl.DataStream)
// 	dataC chan<- *pluginctl.DataStream
// }

// type NoOutputPluginOption = helper.Option[NoOutputPlugin]

// func WithInputWorker(worker func(input <-chan *pluginctl.DataStream)) NoOutputPluginOption {
// 	return func(configure *NoOutputPlugin) {
// 		configure.worker = worker
// 	}
// }

// func NewNoOutputPlugin(opt ...NoOutputPluginOption) NoOutputPlugin {
// 	return helper.Configure(NoOutputPlugin{}, opt...)
// }

// type NoOutputPlugin struct {
// 	EmptySecPlugin
// 	worker func(input <-chan *pluginctl.DataStream)
// }

// func (nop NoOutputPlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
// 	return chantools.NewWithErr(func(c chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
// 		if nop.worker != nil {
// 			nop.worker(input)
// 		}
// 	})
// }

// type ForwardPlugin struct {
// 	EmptySecPlugin
// }

// func (fp ForwardPlugin) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
// 	return input, make(<-chan error)
// }

// func NewOnlyPipePlugin(pipe Pipeable) pluginctl.SecPluginable {
// 	return NewSecPipePlugin(WithPipe(pipe), WithPlugin(ForwardPlugin{}))
// }
