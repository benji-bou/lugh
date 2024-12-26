package grpc

import (
	context "context"
	"errors"
	"fmt"
	"log/slog"
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

func (m *GRPCClient) Run(ctx context.Context, inputC <-chan []byte, outputC chan<- []byte, errC chan<- error) {
	go m.handleStreamInput(ctx, inputC)
	go m.handleStreamOutput(ctx, outputC)
	go m.handleRun(ctx, errC)
}

func (m *GRPCClient) handleRun(ctx context.Context, errC chan<- error) {
	slog.Debug("starting to handle stream run", "name", m.Name, "function", "handleRun", "object", "GRPCClient")
	runStream, err := m.client.Run(ctx, &Empty{})
	if err != nil {
		errC <- fmt.Errorf("run stream %s error %w", m.Name, err)
		return
	}
	for {
		req, err := runStream.Recv()

		if err != nil {
			err = handleGRPCStreamError(err, m.Name)
			if err != nil {
				errC <- fmt.Errorf("run stream %s error %w", m.Name, err)
			}
			return
		}
		errC <- errors.New(req.Message)
	}
}

func (m *GRPCClient) handleStreamOutput(ctx context.Context, outputC chan<- []byte) error {
	slog.Debug("starting to handle stream output", "name", m.Name, "function", "handleStreamOutput", "object", "GRPCClient")
	runloop := NewRunLoop()
	defer func() {
		slog.Debug("closing outputC", "name", m.Name, "function", "handleStreamOutput", "object", "GRPCClient")
		close(m.clientStreamOutputDone)
	}()
	stream, err := m.client.Output(ctx, &Empty{})
	if err != nil {
		return fmt.Errorf("output stream %s error  %w", m.Name, err)
	}
	for {
		req, err := stream.Recv()
		if err != nil {
			slog.Debug("got error from grpc stream recv", "name", m.Name, "err", err, "function", "handleStreamOutput", "object", "GRPCClient")
			return handleGRPCStreamError(err, m.Name)
		}
		slog.Debug("recv output data from grpc stream", "name", m.Name, "function", "handleStreamOutput", "object", "GRPCClient")
		toForward := runloop.Recv(req)
		if toForward != nil {
			outputC <- toForward.Data
		}
	}
}

func (m *GRPCClient) handleStreamInput(ctx context.Context, inputC <-chan []byte) error {
	slog.Debug("starting to handle stream input", "name", m.Name, "function", "handleStreamInput", "object", "GRPCClient")
	stream, err := m.client.Input(ctx)
	if err != nil {
		return fmt.Errorf("failed to retrieve client input stream named %s: %w", m.Name, err)
	}
	outputDone := false
	runloop := NewRunLoop()
	for {
		select {
		case _, ok := <-m.clientStreamOutputDone:
			// `!ok` here `clientStreamOutputDone` is closed means that we won't receive anymore data from the plugin server.
			// Not necessary to send data to the plugin server if no response will ever be sent back
			if !ok {
				outputDone = true
			}
		case inputStreamData, ok := <-inputC:
			// `!ok` no more input will be recieved we can safely close the stream and return
			if !ok {
				slog.Debug("inputC closed. Closing grpc stream", "name", m.Name, "function", "handleStreamInput", "object", "GRPCClient")
				return closeStream(stream, m.Name)
			}
			if !outputDone {
				err := m.sendNewData(runloop, &DataStream{Data: inputStreamData, ParentSrc: m.Name}, stream)
				if err != nil {
					return fmt.Errorf("failed to send data to plugin server: %w", err)
				}
			} else {
				slog.Debug("output stream is Done but keep receiving input data. Doing nothing", "name", m.Name, "function", "handleStreamInput", "object", "GRPCClient")
			}
		}
	}
}

func (m *GRPCClient) sendNewData(runloop *RunLoop, dataStream *DataStream, stream IOWorkerPlugins_InputClient) error {
	for _, dataToSend := range runloop.Send(&DataStream{Data: dataStream.Data, ParentSrc: m.Name}) {
		err := stream.Send(dataToSend)
		if err != nil {
			slog.Error("stream send error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
			return fmt.Errorf("failed to send data to client stream named %s: %w", m.Name, err)
		}
	}
	return nil
}
