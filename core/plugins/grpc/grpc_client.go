package grpc

import (
	context "context"
	"errors"
	"fmt"
	"log/slog"
	sync "sync"

	"github.com/benji-bou/lugh/core/graph"
)

type GRPCClient struct {
	client                 IOWorkerPluginsClient
	Name                   string
	clientStreamOutputDone chan struct{}
	inputC                 <-chan []byte
	outputC                chan []byte
}

func NewGRPCClient(client IOWorkerPluginsClient, name string) *GRPCClient {
	return &GRPCClient{
		client:                 client,
		Name:                   name,
		clientStreamOutputDone: make(chan struct{}),
		outputC:                make(chan []byte),
	}
}

func (m *GRPCClient) SetInput(input <-chan []byte) {
	slog.Debug("setting input", "name", m.Name, "object", "GRPCClient", "function", "SetInput")
	m.inputC = input
}

func (m *GRPCClient) Output() <-chan []byte {
	return m.outputC
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

func (m *GRPCClient) Run(ctx graph.SyncContext) <-chan error {
	errC := make(chan error)
	wg := &sync.WaitGroup{}
	wgInit := &sync.WaitGroup{}
	wg.Add(3)
	wgInit.Add(3)
	ctx.Initializing()
	go func() {
		wgInit.Done()
		defer wg.Done()
		err := m.handleStreamInput(ctx)
		slog.Debug("end handleStreamInput", "name", m.Name, "error", err, "function", "Run", "object", "GRPCClient")
		if err != nil {
			errC <- err
		}
	}()

	go func() {
		wgInit.Done()
		defer wg.Done()
		err := m.handleStreamOutput(ctx)
		slog.Debug("end handleStreamOutput", "name", m.Name, "error", err, "function", "Run", "object", "GRPCClient")
		if err != nil {
			errC <- err
		}
	}()
	go func() {
		wgInit.Done()
		defer wg.Done()
		m.handleRun(ctx, errC)
		slog.Info("end handleRun", "name", m.Name, "function", "Run", "object", "GRPCClient")
	}()
	go func() {
		wgInit.Wait()
		slog.Debug("grpcclient synchronized", "name", m.Name, "function", "Run", "object", "GRPCClient")
		ctx.Initialized()
	}()
	go func() {
		wg.Wait()
		close(errC)
		slog.Info("closing errC", "name", m.Name, "function", "Run", "object", "GRPCClient")
	}()
	return errC
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

func (m *GRPCClient) handleStreamOutput(ctx context.Context) error {
	slog.Debug("starting to handle stream output", "name", m.Name, "function", "handleStreamOutput", "object", "GRPCClient")
	runloop := NewRunLoop()
	defer func() {
		slog.Debug("closing outputC", "name", m.Name, "function", "handleStreamOutput", "object", "GRPCClient")
		close(m.clientStreamOutputDone)
		close(m.outputC)
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
		toForward := runloop.Recv(req)
		if toForward != nil {
			slog.Debug("recv output data from grpc stream",
				"name", m.Name,
				"function", "handleStreamOutput",
				"object", "GRPCClient",
				"data", string(toForward.Data),
			)
			m.outputC <- toForward.Data
		}
	}
}

func (m *GRPCClient) handleStreamInput(ctx context.Context) (err error) {
	slog.Debug("starting to handle stream input", "name", m.Name, "function", "handleStreamInput", "object", "GRPCClient")
	stream, err := m.client.Input(ctx)

	defer func(stream IOWorkerPlugins_InputClient, name string) {
		err = closeStream(stream, name)
	}(stream, m.Name)
	if err != nil {
		err = fmt.Errorf("failed to retrieve client input stream named %s: %w", m.Name, err)
		return err
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
		case inputStreamData, ok := <-m.inputC:
			slog.Debug("recv input data",
				"name", m.Name,
				"function", "handleStreamInput",
				"object", "GRPCClient",
				"data", string(inputStreamData),
			)
			// `!ok` no more input will be received we can safely close the stream and return
			if !ok {
				slog.Debug("inputC closed. Closing grpc stream",
					"name", m.Name,
					"function", "handleStreamInput",
					"object", "GRPCClient",
					"data", string(inputStreamData),
				)
				return err
			}
			if !outputDone {
				slog.Debug("will send recv data",
					"name", m.Name,
					"function", "handleStreamInput",
					"object", "GRPCClient",
					"data", string(inputStreamData),
				)
				err := m.sendNewData(runloop, &DataStream{Data: inputStreamData, ParentSrc: m.Name}, stream)
				if err != nil {
					err = fmt.Errorf("failed to send data to plugin server: %w", err)
					return err
				}
			} else {
				slog.Debug("output stream is Done but keep receiving input data. Doing nothing",
					"name", m.Name,
					"function", "handleStreamInput",
					"object", "GRPCClient")
			}
		}
	}
}

func (m *GRPCClient) sendNewData(runloop *RunLoop, dataStream *DataStream, stream IOWorkerPlugins_InputClient) error {
	for _, dataToSend := range runloop.Send(&DataStream{Data: dataStream.Data, ParentSrc: m.Name}) {
		slog.Debug("will send data chunk",
			"name", m.Name,
			"function", "sendNewData",
			"object", "GRPCClient",
			"data", string(dataStream.Data),
		)
		err := stream.Send(dataToSend)
		if err != nil {
			slog.Error("stream send error",
				"function", "Run",
				"Object", "GRPCClient",
				"error", err,
				"name", m.Name,
			)
			return fmt.Errorf("failed to send data to client stream named %s: %w", m.Name, err)
		}
	}
	return nil
}
