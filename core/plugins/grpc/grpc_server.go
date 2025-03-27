package grpc

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	grpc "google.golang.org/grpc"
)

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Worker     pluginapi.IOWorker
	Configurer pluginapi.PluginConfigurer
	Name       string
}

func (m *GRPCServer) GetInputSchema(context.Context, *Empty) (*InputSchema, error) {
	if m.Configurer != nil {

		rawSchema, err := m.Configurer.GetInputSchema()
		is := &InputSchema{Config: rawSchema}
		return is, err
	}
	return nil, fmt.Errorf("plugin %s does not implement PluginConfigurer", m.Name)
}

func (m *GRPCServer) Config(_ context.Context, config *RunInputConfig) (*Empty, error) {
	if m.Configurer != nil {
		return &Empty{}, m.Configurer.Config(config.Config)
	}
	return nil, fmt.Errorf("plugin %s does not implement PluginConfigurer", m.Name)
}

func (m *GRPCServer) input(ctx graph.SyncContext, stream grpc.BidiStreamingServer[DataStream, RunStream]) error {
	inputC := make(chan []byte)
	m.Worker.SetInput(inputC)
	defer close(inputC)
	runLoop := NewRunLoop()
	ctx.Initialized()
	for {
		if ctx.Err() != nil {
			return ctx.Err()
		}
		req, err := stream.Recv()
		if err != nil {
			slog.Error("Plugin server stream error", "error", err, "name", m.Name)
			return handleGRPCStreamError(err, m.Name)
		}
		slog.Debug("Plugin server received data to input to plugin", "name", m.Name, "data", req, "err", err)
		toForward := runLoop.Recv(req)
		if toForward != nil {
			slog.Debug("Plugin server forwarding data to plugin ", "name", m.Name)
			inputC <- toForward.Data
			slog.Debug("Plugin server forwarded data to plugin ", "name", m.Name)
		}
	}
}

func (m *GRPCServer) Run(stream grpc.BidiStreamingServer[DataStream, RunStream]) error {
	currentctx, cancelCtx := context.WithCancel(stream.Context())
	defer cancelCtx()
	ctxSync := graph.NewContext(currentctx)
	outputC := m.Worker.Output()
	ctxSync.Initializing()
	go m.input(ctxSync, stream)
	errC := m.Worker.Run(ctxSync)
	runLoop := NewRunLoop()
	ctxSync.Synchronize()
	for {
		if errC == nil && outputC == nil {
			return nil
		}
		select {
		case err, ok := <-errC:
			if !ok {
				errC = nil
				continue
			}
			slog.Info("error received", "error", err, "name", m.Name)
			err = stream.Send(&RunStream{Error: &Error{Message: err.Error()}})
			if err != nil {
				slog.Error("sending error data over stream failed",
					"function", "Output",
					"Object", "GRPCServer",
					"error", err,
					"name", m.Name,
				)
				return err
			}
		case <-ctxSync.Done():
			return ctxSync.Err()
		case dataOutput, ok := <-outputC:
			if !ok {
				outputC = nil
				continue
			}
			slog.Debug("Plugin server received data to output from plugin ", "name", m.Name)
			for _, d := range runLoop.Send(&DataStream{Data: dataOutput, ParentSrc: m.Name}) {
				err := stream.Send(&RunStream{Data: d})
				if err != nil {
					slog.Error("sending data over stream failed",
						"function", "Output",
						"Object", "GRPCServer",
						"error", err,
						"name", m.Name,
					)
					return err
				}
			}
		}
	}

	slog.Info("GRPC Server Run ended (run error channel of underlying plugin is closed)", "name", m.Name)
	return nil
}

func (*GRPCServer) mustEmbedUnimplementedIOWorkerPluginsServer() {
	slog.Info("inside GRPCServer mustEmbedUnimplementedIOWorkerPluginsServer")
}
