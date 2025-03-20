package grpc

import (
	context "context"
	"errors"
	"fmt"
	"log/slog"

	"google.golang.org/grpc"
)

type GRPCClient struct {
	client                 IOWorkerPluginsClient
	Name                   string
	clientStreamOutputDone chan struct{}
}

func NewGRPCClient(client IOWorkerPluginsClient, name string) *GRPCClient {
	return &GRPCClient{
		client:                 client,
		Name:                   name,
		clientStreamOutputDone: make(chan struct{}),
	}
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

func (m *GRPCClient) Run(ctx context.Context, inputC <-chan []byte, yield func(elem []byte, err error) error) error {
	runStream, err := m.client.Run(ctx)
	if err != nil {
		return fmt.Errorf("failed to create run stream: %w", err)
	}

	inputDoneC := make(chan struct{})
	go func() {
		m.handleInputStream(ctx, inputC, runStream)
		close(inputDoneC)
	}()
	err = m.handleOutputStream(runStream, yield)
	<-inputDoneC
	return err
}

func (m *GRPCClient) handleInputStream(ctx context.Context, inputC <-chan []byte, runStream grpc.BidiStreamingClient[DataStream, RunStream]) {
	outputDone := false
	runloop := NewRunLoop()
	defer runStream.CloseSend()
	for {
		var outputDoneC <-chan struct{}
		if !outputDone {
			outputDoneC = m.clientStreamOutputDone
		}
		select {
		case <-ctx.Done():
			slog.Debug("Runner: GRPCCLient: context done", "GRPCClient", m.Name)
			return
		case _, ok := <-outputDoneC:
			// `!ok` here `clientStreamOutputDone` is closed means that we won't receive anymore data from the plugin server.
			// Not necessary to send data to the plugin server if no response will ever be sent back
			if !ok {
				outputDone = true
			}
		case inputStreamData, ok := <-inputC:
			// `!ok` no more input will be received we can safely close the stream and return
			if !ok {
				slog.Debug("input channel closed", "GRPCClient", m.Name)
				return
			}
			if !outputDone {
				slog.Debug("Runner: GRPCCLient: sending data to plugin server", "GRPCClient", m.Name)
				err := m.sendNewData(runloop, &DataStream{Data: inputStreamData, ParentSrc: m.Name}, runStream)
				if err != nil {
					slog.Error("Runner: GRPCCLient:failed to send data to plugin server", "GRPCClient", m.Name,
						"function", "handleGRPCPluginInput", "error", err)
				}
			} else {
				slog.Debug("Runner: GRPCCLient: Output stream is Done but keep receiving input data. Doing nothing",
					"GRPCClient", m.Name,
					"function", "handleGRPCPluginInput")
			}
		}
	}
}

func (m *GRPCClient) handleOutputStream(runStream grpc.BidiStreamingClient[DataStream, RunStream], yield func(elem []byte, err error) error) error {
	defer func() {
		slog.Debug("Closing GRPCClient output stream", "GRPCClient", m.Name)
		close(m.clientStreamOutputDone)
	}()
	runloop := NewRunLoop()
	for {
		req, err := runStream.Recv()
		if err != nil {
			slog.Info("GRPCClient: Plugin client received stream error", "error", err, "name", m.Name)
			err = handleGRPCStreamError(err, m.Name)
			if err != nil {
				return fmt.Errorf("run stream %s error %w", m.Name, err)
			}
			return nil
		}
		if req.Error != nil {
			if errYield := yield(nil, errors.New(req.Error.Message)); errYield != nil {
				return errYield
			}
			continue
		}
		toForward := runloop.Recv(req.Data)
		if toForward != nil {
			if errYield := yield(toForward.Data, nil); errYield != nil {
				return errYield
			}
		}
	}
}

func (m *GRPCClient) sendNewData(runloop *RunLoop, dataStream *DataStream, stream grpc.BidiStreamingClient[DataStream, RunStream]) error {
	for _, dataToSend := range runloop.Send(&DataStream{Data: dataStream.Data, ParentSrc: m.Name}) {
		err := stream.Send(dataToSend)
		if err != nil {
			return fmt.Errorf("send data to client stream named %s: %w", m.Name, err)
		}
	}
	return nil
}
