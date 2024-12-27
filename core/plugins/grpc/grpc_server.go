package grpc

import (
	context "context"
	"log/slog"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
)

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl pluginapi.IOWorkerPluginable
	Name string
}

func (m *GRPCServer) GetInputSchema(context.Context, *Empty) (*InputSchema, error) {
	rawSchema, err := m.Impl.GetInputSchema()
	is := &InputSchema{Config: rawSchema}
	return is, err
}
func (m *GRPCServer) Config(ctx context.Context, config *RunInputConfig) (*Empty, error) {
	return &Empty{}, m.Impl.Config(config.Config)
}

func (m *GRPCServer) Input(stream IOWorkerPlugins_InputServer) error {
	inputC := make(chan []byte)
	m.Impl.SetInput(inputC)
	defer close(inputC)
	runLoop := NewRunLoop()
	for {
		req, err := stream.Recv()
		if err != nil {
			return handleGRPCStreamError(err, m.Name)
		}
		toForward := runLoop.Recv(req)
		slog.Debug("recv data", "function", "Input", "Object", "GRPCServer", "name", m.Name, "data", req.Data)
		if toForward != nil {
			slog.Debug("send data to input", "function", "Input", "Object", "GRPCServer", "name", m.Name, "data", req.Data)
			inputC <- toForward.Data
		}
	}
}

func (m *GRPCServer) Output(empty *Empty, stream IOWorkerPlugins_OutputServer) error {
	runLoop := NewRunLoop()
	for dataOutput := range m.Impl.Output() {
		slog.Debug("receive data from outputC", "function", "Output", "Object", "GRPCServer", "data", dataOutput, "name", m.Name)
		for _, d := range runLoop.Send(&DataStream{Data: dataOutput, ParentSrc: m.Name}) {
			slog.Debug("sending data over stream", "function", "Output", "Object", "GRPCServer", "data", dataOutput, "name", m.Name)
			err := stream.Send(d)
			if err != nil {
				slog.Error("sending data over stream failed", "function", "Output", "Object", "GRPCServer", "error", err, "name", m.Name)
				return err
			}
		}
	}
	slog.Info("end of GRPCServer output stream", "name", m.Name)
	return nil
}

func (m *GRPCServer) Run(_ *Empty, s IOWorkerPlugins_RunServer) error {
	errC := m.Impl.Run(graph.NewContext(s.Context())) //, graph.NewWorkerSynchronization()
	slog.Info("start listening for errors", "name", m.Name)
	for err := range errC {
		slog.Info("error received", "error", err, "name", m.Name)
		s.Send(&Error{Message: err.Error()})
		slog.Info("error sent", "error", err, "name", m.Name)
	}
	slog.Info("end of error listening", "name", m.Name)
	return nil
}

func (m *GRPCServer) mustEmbedUnimplementedIOWorkerPluginsServer() {
	slog.Info("inside GRPCServer mustEmbedUnimplementedIOWorkerPluginsServer")
}
