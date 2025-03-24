package grpc

import (
	context "context"
	"errors"

	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	goplugin "github.com/hashicorp/go-plugin"
	grpc "google.golang.org/grpc"
)

var (
	ErrJSONSchemaConvertion error = errors.New("failed to convert to json format")
	ErrJSONConvertion       error = errors.New("failed to convert to/from json")
)

// This is the implementation of plugin.GRPCPlugin so we can serve/consume this.
type IOWorkerGRPCPlugin struct {
	// GRPCPlugin must still implement the Plugin interface
	goplugin.NetRPCUnsupportedPlugin
	Impl pluginapi.ConfigurableIOWorker
	Name string
	// Concrete implementation, written in Go. This is only used for plugins
	// that are written in Go.
}

func (p IOWorkerGRPCPlugin) GRPCServer(_ *goplugin.GRPCBroker, s *grpc.Server) error {
	RegisterIOWorkerPluginsServer(s, &GRPCServer{
		Impl: p.Impl,
		Name: p.Name,
	})
	return nil
}

func (p IOWorkerGRPCPlugin) GRPCClient(_ context.Context, _ *goplugin.GRPCBroker, c *grpc.ClientConn) (any, error) {
	return NewGRPCClient(NewIOWorkerPluginsClient(c), p.Name), nil
}
