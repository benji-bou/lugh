package pluginctl

import (
	context "context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/benji-bou/chantools"
	"github.com/hashicorp/go-plugin"
)

type SecPipelinePluginable interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
	Run(input <-chan []byte) (<-chan []byte, <-chan error)
}

type GRPCClient struct {
	client SecPipelinePluginsClient
	broker *plugin.GRPCBroker
}

func (m *GRPCClient) GetInputSchema() ([]byte, error) {
	resp, err := m.client.GetInputSchema(context.Background(), &Empty{})
	return resp.Config, err
}

func (m *GRPCClient) Config(config []byte) error {
	in := &RunInputConfig{Config: config}
	_, err := m.client.Config(context.Background(), in)
	return err

}

func (m *GRPCClient) Run(input <-chan []byte) (<-chan []byte, <-chan error) {
	stream, err := m.client.Run(context.Background())
	if err != nil {
		return nil, chantools.Once(fmt.Errorf("grpc client call to run failed %w", err))
	}

	chantools.ForEach(input, func(data []byte) {
		stream.Send(&DataStream{Data: data})
	})

	return chantools.NewWithErr(func(dataC chan<- []byte, errC chan<- error, params ...any) {
		stream := params[0].(SecPipelinePlugins_RunClient)
		for {
			req, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				slog.Debug("end of stream", "function", "Run", "Object", "GRPCClient")
				return
			} else if err != nil {
				slog.Error("stream error", "function", "Run", "Object", "GRPCClient", "error", err)
				errC <- err
				return
			}
			dataC <- req.Data
		}
	}, chantools.WithParam[[]byte](stream))
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	broker *plugin.GRPCBroker
	Impl   SecPipelinePluginable
}

func (m *GRPCServer) GetInputSchema(context.Context, *Empty) (*InputSchema, error) {
	rawSchema, err := m.Impl.GetInputSchema()
	is := &InputSchema{Config: rawSchema}
	return is, err
}
func (m *GRPCServer) Config(ctx context.Context, config *RunInputConfig) (*Empty, error) {
	return &Empty{}, m.Impl.Config(config.Config)
}

func (m *GRPCServer) Run(stream SecPipelinePlugins_RunServer) error {

	inputDataC, inputErrC := chantools.NewWithErr(func(dataC chan<- []byte, errC chan<- error, params ...any) {
		stream := params[0].(SecPipelinePlugins_RunServer)
		for {
			req, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				slog.Debug("end of stream", "function", "Run", "Object", "GRPCClient")
				return
			} else if err != nil {
				slog.Error("stream error", "function", "Run", "Object", "GRPCClient", "error", err)
				errC <- err
				return
			}
			dataC <- req.Data
		}
	}, chantools.WithParam[[]byte](stream))

	outputDataC, outputErrC := m.Impl.Run(inputDataC)
	for {
		select {
		case data, ok := <-outputDataC:
			if !ok {
				slog.Debug("data chan closed", "function", "Run", "Object", "GRPCServer")
				return nil
			}
			err := stream.Send(&DataStream{Data: data})
			if err != nil {
				slog.Error("sending data over stream failed", "function", "Run", "Object", "GRPCServer", "error", err)
				return err
			}
		case err, ok := <-outputErrC:
			if !ok {
				slog.Debug("output error chan closed", "function", "Run", "Object", "GRPCServer")
				return nil
			}
			slog.Error("output error chan received error", "function", "Run", "Object", "GRPCServer", "error", err)
			return err
		case err, ok := <-inputErrC:
			if !ok {
				slog.Debug("input error chan closed", "function", "Run", "Object", "GRPCServer")
				return nil
			}
			slog.Error("input error chan received error", "function", "Run", "Object", "GRPCServer", "error", err)
		}
	}

}

func (m *GRPCServer) mustEmbedUnimplementedSecPipelinePluginsServer() {
	slog.Info("inside GRPCServer mustEmbedUnimplementedSecPipelinePluginsServer")
}
