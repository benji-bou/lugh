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
)

type SecPipelinePluginable interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
	Run(ctx context.Context, input <-chan *DataStream) (<-chan *DataStream, <-chan error)
}

type GRPCClient struct {
	client SecPipelinePluginsClient
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
	chantools.ForEach(input, func(data *DataStream) {
		for _, d := range runLoop.Send(data) {
			stream.Send(d)
		}
	})

	return chantools.NewWithErr(func(dataC chan<- *DataStream, errC chan<- error, params ...any) {
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
			toForward := runLoop.Recv(req)
			if toForward != nil {
				dataC <- toForward
			}
		}
	}, chantools.WithParam[*DataStream](stream))
}

// Here is the gRPC server that GRPCClient talks to.
type GRPCServer struct {
	// This is the real implementation
	Impl SecPipelinePluginable
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
	runLoop := NewRunLoop()
	inputDataC, inputErrC := chantools.NewWithErr(func(dataC chan<- *DataStream, errC chan<- error, params ...any) {
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
			toForward := runLoop.Recv(req)
			if toForward != nil {
				dataC <- toForward
			}
		}
	}, chantools.WithParam[*DataStream](stream), chantools.WithName[*DataStream]("GRPCSServer Receive"))
	defer func() {
		slog.Debug("quiting GRPC Run", "function", "Run", "Object", "GRPCServer")
	}()
	outputDataC, outputErrC := m.Impl.Run(stream.Context(), inputDataC)
	for {
		select {
		case data, ok := <-outputDataC:
			if !ok {
				slog.Debug("data chan closed", "function", "Run", "Object", "GRPCServer")
				return nil
			}
			for _, d := range runLoop.Send(data) {
				err := stream.Send(d)
				if err != nil {
					slog.Error("sending data over stream failed", "function", "Run", "Object", "GRPCServer", "error", err)
					return err
				}
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
