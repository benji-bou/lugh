package grpc

import (
	"bytes"
	context "context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/benji-bou/SecPipeline/core/plugins"
	"github.com/benji-bou/chantools"
	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
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

func (m *GRPCClient) handleStreamOutput(ctx context.Context, outputC chan<- []byte) error {

	runloop := NewRunLoop()
	defer close(m.clientStreamOutputDone)
	stream, err := m.client.Output(ctx, &Empty{})
	if err != nil {
		return fmt.Errorf("output stream %s error  %w", m.Name, err)
	}
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("end of stream", "function", "Run", "Object", "GRPCClient", "name", m.Name)
				return nil
			} else if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
				slog.Debug("stream canceled", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
				return nil
			} else {
				slog.Error("stream error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
				return fmt.Errorf("stream %s error %w", m.Name, err)
			}
		}
		slog.Debug("recv data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "err", err)
		toForward := runloop.Recv(req)
		if toForward != nil {
			outputC <- toForward.Data
		}
	}
}

func (m *GRPCClient) handleStreamInput(ctx context.Context, inputC <-chan []byte) error {

	outputDone := false
	stream, err := m.client.Input(ctx)
	// isClosed := false
	runloop := NewRunLoop()
	if err != nil {
		return fmt.Errorf("failed to retrieve client input stream named %s: %w", m.Name, err)
	}
	for {
		select {
		case _, ok := <-m.clientStreamOutputDone:
			// `!ok` here `clientStreamOutputDone` is closed means that we won't receive anymore data from the plugin server.
			// Not necessary to send data to the plugin server if no response will ever be sent back
			if !ok {
				outputDone = true

				// isClosed = true
				//Here we do not return to prevent blocking the input channel and creating a go routine zombie on the other side of the channel
				//TODO: Handle properly the `chantools.Broadcast` cancelation of listen with some kind of context
			}
		case inputStreamData, ok := <-inputC:
			// `!ok` no more input will be recieved we can safely close the stream and return
			if !ok {
				err := stream.CloseSend()
				if err != nil {
					return fmt.Errorf("failed to close client stream named v%s: %w", m.Name, err)
				}
				return nil
			}
			if !outputDone {
				for _, dataToSend := range runloop.Send(&DataStream{Data: inputStreamData, ParentSrc: m.Name}) {
					err := stream.Send(dataToSend)
					if err != nil {
						slog.Error("stream send error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
					}
				}

			} else {
				slog.Debug("output stream is Done but keep receiving input data. Doing nothing", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
			}
		}
	}
}

func (m *GRPCClient) Run(ctx context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	outputC, errOutputC := chantools.NewWithErr(func(c chan<- []byte, eC chan<- error, params ...any) {
		err := m.handleStreamOutput(ctx, c)
		if err != nil {
			eC <- err
		}
	})
	errInputC := chantools.New(func(eC chan<- error, params ...any) {
		err := m.handleStreamInput(ctx, input)
		if err != nil {
			eC <- err
		}
	})
	return outputC, chantools.Merge(errOutputC, errInputC)
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl   plugins.IOPluginable
	inputC chan []byte
	Name   string
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
	m.inputC = make(chan []byte)
	defer close(m.inputC)
	runLoop := NewRunLoop()
	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				slog.Debug("end of stream", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			} else if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
				slog.Debug("stream canceled", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
				return nil
			} else {
				slog.Error("stream error", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
				return err
			}
		}
		slog.Debug("recv data", "function", "Run", "Object", "GRPCServer", "name", m.Name)
		toForward := runLoop.Recv(req)
		if toForward != nil {
			m.inputC <- toForward.Data
		}
	}
}

func (m *GRPCServer) Output(empty *Empty, stream IOWorkerPlugins_OutputServer) error {

	runLoop := NewRunLoop()
	outputRawC, outputErrC := m.Impl.Run(context.Background(), m.inputC)

	for {
		select {
		case data, ok := <-outputRawC:
			if !ok {
				slog.Debug("data chan closed", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			}
			for _, d := range runLoop.Send(&DataStream{Data: data, ParentSrc: m.Name}) {
				slog.Debug("sending data over stream", "function", "Run", "Object", "GRPCServer", "data", data, "name", m.Name)
				err := stream.Send(d)
				if err != nil {
					slog.Error("sending data over stream failed", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
					return err
				}
			}
		case err, ok := <-outputErrC:
			if !ok {
				slog.Debug("output error chan closed", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			}
			slog.Error("output error chan received error", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
			return err

		}

	}

}

func (m *GRPCServer) mustEmbedUnimplementedIOWorkerPluginsServer() {
	slog.Info("inside GRPCServer mustEmbedUnimplementedIOWorkerPluginsServer")
}

type RunLoop struct {
	buffer map[string][]*DataStream
	lim    int
}

func NewRunLoop() *RunLoop {

	return &RunLoop{buffer: make(map[string][]*DataStream, 0), lim: 1024 * 1024 * 3}
}

func (rl *RunLoop) Recv(stream *DataStream) *DataStream {
	id := stream.Id
	bufStream, exist := rl.buffer[id]
	if !exist && stream.IsComplete {
		return stream
	} else if !exist {
		bufStream = make([]*DataStream, 0, stream.TotalLen/int64(rl.lim))
	}
	bufStream = append(bufStream, stream)
	if stream.IsComplete {
		buffer := bytes.NewBuffer(make([]byte, 0, stream.TotalLen))
		for _, ds := range bufStream {
			_, err := buffer.Write(ds.Data)
			if err != nil {
				slog.Error("failed to write data", "function", "Recv", "object", "runloop", "error", err)
			}
		}
		stream.Data = buffer.Bytes()
		return stream
	}
	return nil
}

func (rl *RunLoop) Send(stream *DataStream) []*DataStream {
	stream.Id = uuid.NewString()

	buf := stream.Data
	totalLen := len(buf)
	res := make([]*DataStream, 0, totalLen/rl.lim)
	var chunk []byte

	for len(chunk) >= rl.lim {
		chunk, buf = buf[:rl.lim], buf[rl.lim:]
		res = append(res, &DataStream{Data: chunk, Id: stream.Id, IsComplete: false, TotalLen: int64(totalLen)})
	}
	if len(buf) > 0 {
		res = append(res, &DataStream{Data: buf[:], Id: stream.Id, IsComplete: true, TotalLen: int64(totalLen)})
	}
	return res
}
