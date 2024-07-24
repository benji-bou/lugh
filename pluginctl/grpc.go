package pluginctl

import (
	"bytes"
	context "context"
	"errors"
	"fmt"
	"io"
	"log/slog"

	"github.com/benji-bou/chantools"
	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

type SecPipelinePluginable interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
	Run(ctx context.Context, input <-chan *DataStream) (<-chan *DataStream, <-chan error)
}

type SecPipelinePluginLifecycle interface {
	SendLifecycleEvent(ctx context.Context, event *LifecycleEvent) error
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
	stream, err := m.client.Run(ctx)
	if err != nil {
		return nil, chantools.Once(fmt.Errorf("grpc client call to run failed %w", err))
	}
	runLoop := NewRunLoop()
	chantools.ForEachWithEnd(input, func(data *DataStream) {
		slog.Debug("will send data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "req", data)
		for _, d := range runLoop.Send(data) {
			slog.Debug("send data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "req", d)

			err := stream.Send(d)
			if err != nil {
				slog.Error("stream send error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name)
			}
		}

	}, func() {
		slog.Debug("end of input stream", "function", "Run", "Object", "GRPCClient", "name", m.Name)
		slog.Info("send a NO_MORE_MESSAGE Event", "function", "Run", "Object", "GRPCClient", "name", m.Name)
		m.client.SendLifecycleEvent(ctx, &LifecycleEvent{Code: LifecycleEventCode_NO_MORE_MESSAGES})
	})

	return chantools.NewWithErr(func(dataC chan<- *DataStream, errC chan<- error, params ...any) {
		stream := params[0].(SecPipelinePlugins_RunClient)
		for {
			req, err := stream.Recv()
			if errors.Is(err, io.EOF) {
				slog.Debug("end of stream", "function", "Run", "Object", "GRPCClient", "name", m.Name, "req", req)
				return
			} else if err != nil {
				slog.Error("stream recv error", "function", "Run", "Object", "GRPCClient", "error", err, "name", m.Name, "req", req)
				errC <- err
				return
			} else {
				slog.Debug("recv data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "req", req)
				toForward := runLoop.Recv(req)
				if toForward != nil {
					slog.Debug("will forward data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "req", req)
					dataC <- toForward
					slog.Debug("did forward data", "function", "Run", "Object", "GRPCClient", "name", m.Name, "req", req)
				}
			}
		}
	}, chantools.WithParam[*DataStream](stream), chantools.WithName[*DataStream](m.Name+"-"+uuid.NewString()))
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl         SecPipelinePluginable
	Name         string
	runCancelCtx context.CancelFunc
}

func (m *GRPCServer) GetInputSchema(context.Context, *Empty) (*InputSchema, error) {
	rawSchema, err := m.Impl.GetInputSchema()
	is := &InputSchema{Config: rawSchema}
	return is, err
}
func (m *GRPCServer) Config(ctx context.Context, config *RunInputConfig) (*Empty, error) {
	return &Empty{}, m.Impl.Config(config.Config)
}

func (m *GRPCServer) SendLifecycleEvent(ctx context.Context, event *LifecycleEvent) (*Empty, error) {
	var err error
	slog.Debug("Receive Lifecycle Event", "function", "SendLifecycleEvent", "Object", "GRPCServer", "name", m.Name, "event", event.Code)
	if plugin, ok := m.Impl.(SecPipelinePluginLifecycle); ok {
		err = plugin.SendLifecycleEvent(ctx, event)
	} else {
		slog.Debug("Cancelling GRPCserver context", "function", "SendLifecycleEvent", "Object", "GRPCServer", "name", m.Name)
		m.runCancelCtx()
	}

	return &Empty{}, err
}

func (m *GRPCServer) Run(stream SecPipelinePlugins_RunServer) error {
	runLoop := NewRunLoop()
	var ctx context.Context
	ctx, m.runCancelCtx = context.WithCancel(stream.Context())
	inputDataC, inputErrC := chantools.NewWithErr(func(dataC chan<- *DataStream, errC chan<- error, params ...any) {
		stream := params[0].(SecPipelinePlugins_RunServer)
		for {
			req, err := stream.Recv()
			if err != nil {
				if errors.Is(err, io.EOF) {
					slog.Debug("end of stream", "function", "Run", "Object", "GRPCServer", "name", m.Name)
					return
				} else if e, ok := status.FromError(err); ok {
					switch e.Code() {
					case codes.Canceled:
						slog.Debug("stream canceled", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
						return
					}
				} else {
					slog.Error("stream error", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
					errC <- err
					return
				}

			}
			slog.Debug("recv data", "function", "Run", "Object", "GRPCServer", "name", m.Name, "req", req)
			toForward := runLoop.Recv(req)
			if toForward != nil {
				dataC <- toForward
			}
		}
	}, chantools.WithParam[*DataStream](stream), chantools.WithName[*DataStream]("GRPCSServer Receive"))
	defer func() {
		slog.Debug("quiting GRPC Run", "function", "Run", "Object", "GRPCServer", "name", m.Name)
	}()
	outputDataC, outputErrC := m.Impl.Run(ctx, inputDataC)
	for {
		select {
		case data, ok := <-outputDataC:
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
		case err, ok := <-outputErrC:
			if !ok {
				slog.Debug("output error chan closed", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			}
			slog.Error("output error chan received error", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
			return err
		case err, ok := <-inputErrC:
			if !ok {
				slog.Debug("input error chan closed", "function", "Run", "Object", "GRPCServer", "name", m.Name)
				return nil
			}
			slog.Error("input error chan received error", "function", "Run", "Object", "GRPCServer", "error", err, "name", m.Name)
			// case <-stream.Context().Done():
			//
			// 	return nil
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
