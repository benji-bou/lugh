package pluginctl

import (
	"bytes"
	context "context"
	"errors"
	"io"
	"log/slog"

	"github.com/benji-bou/chantools"
	"github.com/google/uuid"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type SecPluginable interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
	Run(ctx context.Context, input <-chan *DataStream) (<-chan *DataStream, <-chan error)
}

type GRPCClient struct {
	client SecPipelinePluginsClient
	Name   string
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

func (m *GRPCClient) Run(ctx context.Context, input <-chan *DataStream) (<-chan *DataStream, <-chan error) {
	clientStreamOutputDone := make(chan struct{})
	runLoop := NewRunLoop()
	outputDataC, outputErrC := chantools.NewWithErr(func(dataC chan<- *DataStream, errC chan<- error, params ...any) {
		defer close(clientStreamOutputDone)
		stream, err := m.client.Output(ctx, &Empty{})
		if err != nil {
			slog.Error("output stream error", "err", err, "function", "Run", "Object", "GRPCClient", "name", m.Name)
			errC <- err
		}
		for {
			req, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Debug("end of stream", "function", "Run", "Object", "GRPCClient", "name", m.Name)
					return
				} else if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
					slog.Debug("stream canceled", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
					return
				} else {
					slog.Error("stream error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
					errC <- err
					return
				}
			}
			slog.Debug("recv data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "err", err)
			toForward := runLoop.Recv(req)
			if toForward != nil {
				dataC <- toForward
			}
		}
	}, chantools.WithName[*DataStream](m.Name+"-"+uuid.NewString()))

	go func() {
		stream, err := m.client.Input(ctx)
		// isClosed := false
		if err != nil {
			slog.Debug("failed to retrieve client input stream", "name", m.Name, "err", err)
			return
		}
		for {
			select {
			case _, ok := <-clientStreamOutputDone:
				// `!ok` here `clientStreamOutputDone` is closed means that we won't receive anymore data from the plugin server.
				// Not necessary to send data to the plugin server if no response will ever be sent back
				if !ok {
					err := stream.CloseSend()
					if err != nil {
						slog.Debug("failed to close client stream", "name", m.Name, "err", err)
					}
					// isClosed = true
					//Here we do not return to prevent blocking the input channel and creating a go routine zombie on the other side of the channel
					//TODO: Handle properly the `chantools.Broadcast` cancelation of listen with some kind of context
				}
			case inputStreamData, ok := <-input:
				// `!ok` no more input will be recieved we can safely close the stream and return
				if !ok {
					err := stream.CloseSend()
					if err != nil {
						slog.Debug("failed to close client stream", "name", m.Name, "err", err)
					}

					return
				}
				// if !isClosed {
				err := stream.Send(inputStreamData)
				if err != nil {
					slog.Error("stream send error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)

					// }
				}

			}
		}

	}()

	return outputDataC, outputErrC
}

type pluginServerDataC struct {
	outputErrC  <-chan error
	outputDataC <-chan *DataStream
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl        SecPluginable
	Name        string
	PluginDataC chan pluginServerDataC
}

func (m *GRPCServer) GetInputSchema(context.Context, *Empty) (*InputSchema, error) {
	rawSchema, err := m.Impl.GetInputSchema()
	is := &InputSchema{Config: rawSchema}
	return is, err
}
func (m *GRPCServer) Config(ctx context.Context, config *RunInputConfig) (*Empty, error) {
	return &Empty{}, m.Impl.Config(config.Config)
}

func (m *GRPCServer) Input(stream SecPipelinePlugins_InputServer) error {
	inputDataC := make(chan *DataStream)
	defer close(inputDataC)
	runLoop := NewRunLoop()
	outputDataC, outputErrC := m.Impl.Run(context.Background(), inputDataC)
	m.PluginDataC <- pluginServerDataC{outputErrC: outputErrC, outputDataC: outputDataC}

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
			inputDataC <- toForward
		}
	}
}

func (m *GRPCServer) Output(empty *Empty, stream SecPipelinePlugins_OutputServer) error {
	pluginDataC := <-m.PluginDataC
	runLoop := NewRunLoop()
	for {
		select {
		case data, ok := <-pluginDataC.outputDataC:
			if !ok {
				slog.Debug("data chan closed", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			}
			for _, d := range runLoop.Send(data) {
				slog.Debug("sending data over stream", "function", "Run", "Object", "GRPCServer", "data", data, "name", m.Name)
				err := stream.Send(d)
				if err != nil {
					slog.Error("sending data over stream failed", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
					return err
				}
			}
		case err, ok := <-pluginDataC.outputErrC:
			if !ok {
				slog.Debug("output error chan closed", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			}
			slog.Error("output error chan received error", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
			return err

		}

	}

}

func (m *GRPCServer) mustEmbedUnimplementedSecPipelinePluginsServer() {
	slog.Info("inside GRPCServer mustEmbedUnimplementedSecPipelinePluginsServer")
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
