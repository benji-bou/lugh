package plugins

import (
	"context"

	"github.com/benji-bou/SecPipeline/core/graph"
)

type IOPluginable interface {
	Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error)
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}

type IOWorkerPlugin struct {
	decorated IOPluginable
	graph.DefaultIOWorker[[]byte]
}

func NewIOWorkerPlugin(decorated IOPluginable) *IOWorkerPlugin {
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

// import (
// 	"context"
// 	"log/slog"
// 	"os"

// 	"github.com/benji-bou/SecPipeline/helper"
// 	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
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

// func (spp EmptySecPlugin) Run(ctx context.Context, input <-chan *grpc.DataStream) (<-chan *grpc.DataStream, <-chan error) {
// 	return nil, nil
// }

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

// func (spp SecPipePlugin) Run(ctx context.Context, input <-chan *grpc.DataStream) (<-chan *grpc.DataStream, <-chan error) {
// 	pipeOutputC, _ := spp.pipe.Pipe(ctx, input)
// 	return spp.SecPluginable.Run(ctx, pipeOutputC)
// }

// type RawOutputPluginOption = helper.Option[RawOutputPlugin]

// func WithCallback(cb func(data *grpc.DataStream)) RawOutputPluginOption {
// 	return func(configure *RawOutputPlugin) {
// 		configure.cb = cb
// 	}
// }

// // WithChannel: when this option is set you pass a writable channel which will be used to write the input of the plugin into it
// // in this case RawOutputPlugin takes ownership of the channel and close it when it is done writing it. Means the plugin do not received input anymore
// func WithChannel(dataC chan<- *grpc.DataStream) RawOutputPluginOption {
// 	return func(configure *RawOutputPlugin) {
// 		configure.dataC = dataC
// 	}
// }

// func NewRawOutputPlugin(opt ...RawOutputPluginOption) RawOutputPlugin {

// 	rop := helper.Configure(RawOutputPlugin{}, opt...)
// 	rop.NoOutputPlugin = NewNoOutputPlugin(WithInputWorker(func(input <-chan *grpc.DataStream) {
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
// 	cb    func(data *grpc.DataStream)
// 	dataC chan<- *grpc.DataStream
// }

// type NoOutputPluginOption = helper.Option[NoOutputPlugin]

// func WithInputWorker(worker func(input <-chan *grpc.DataStream)) NoOutputPluginOption {
// 	return func(configure *NoOutputPlugin) {
// 		configure.worker = worker
// 	}
// }

// func NewNoOutputPlugin(opt ...NoOutputPluginOption) NoOutputPlugin {
// 	return helper.Configure(NoOutputPlugin{}, opt...)
// }

// type NoOutputPlugin struct {
// 	EmptySecPlugin
// 	worker func(input <-chan *grpc.DataStream)
// }

// func (nop NoOutputPlugin) Run(ctx context.Context, input <-chan *grpc.DataStream) (<-chan *grpc.DataStream, <-chan error) {
// 	return chantools.NewWithErr(func(c chan<- *grpc.DataStream, eC chan<- error, params ...any) {
// 		if nop.worker != nil {
// 			nop.worker(input)
// 		}
// 	})
// }

// type ForwardPlugin struct {
// 	EmptySecPlugin
// }

// func (fp ForwardPlugin) Run(ctx context.Context, input <-chan *grpc.DataStream) (<-chan *grpc.DataStream, <-chan error) {
// 	return input, make(<-chan error)
// }

// func NewOnlyPipePlugin(pipe Pipeable) grpc.SecPluginable {
// 	return NewSecPipePlugin(WithPipe(pipe), WithPlugin(ForwardPlugin{}))
// }
