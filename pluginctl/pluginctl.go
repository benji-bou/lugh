package pluginctl

import (
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/hashicorp/go-hclog"
	"github.com/hashicorp/go-plugin"
)

var DefaultHandshake = plugin.HandshakeConfig{
	// This isn't required when using VersionedPlugins
	ProtocolVersion:  1,
	MagicCookieKey:   "BASIC_SECPIPELINE_PLUGIN",
	MagicCookieValue: "hello",
}

type PluginOption = helper.Option[Plugin]
type Plugin struct {
	path      string
	name      string
	cmd       *exec.Cmd
	plugin    plugin.Plugin
	handshake plugin.HandshakeConfig
	client    *plugin.Client
}

func NewPlugin(name string, opt ...PluginOption) *Plugin {
	defaultOption := []PluginOption{
		withDefaultPath(),
		WithHandshakeConfig(DefaultHandshake),
		WithGRPCPlugin(),
	}
	mandatoryOption := []PluginOption{
		withCmd(false),
	}

	allOptionChain := append(defaultOption, append(opt, mandatoryOption...)...)
	pl := helper.ConfigurePtr(&Plugin{name: name}, allOptionChain...)
	return pl
}

func WithHandshakeConfig(handshakeConfig plugin.HandshakeConfig) PluginOption {
	return func(p *Plugin) {
		p.handshake = handshakeConfig
	}
}

func withCmd(force bool) PluginOption {
	return func(configure *Plugin) {
		if force || configure.cmd == nil {
			withDefaultPluginProcess()(configure)
		}
	}
}

func withDefaultPath() PluginOption {

	p, err := os.UserHomeDir()
	if err != nil {
		slog.Info("unable to find default user Home path for plugins", "error", err)
		p, err = os.UserConfigDir()
		if err != nil {
			slog.Warn("unable to find deafult path for plugins")
			return nil
		}
	}
	realPath := filepath.Join(p, ".secpipeline", "plugins")
	return WithPath(realPath)

}

func WithPath(path string) PluginOption {
	return func(p *Plugin) {
		err := os.MkdirAll(path, 0744)
		if err != nil {
			slog.Error("unable to create default config path", "path", path, "error", err)
			return
		}
		p.path = path
	}
}

func withDefaultPluginProcess() PluginOption {
	return func(p *Plugin) {
		p.cmd = exec.Command("sh", "-c", filepath.Join(p.path, p.name))
	}
}

func WithPluginProcessPath(path string) PluginOption {
	return func(p *Plugin) {
		p.cmd = exec.Command("sh", "-c", path)
	}
}

func WithCmdConfig(cmd *exec.Cmd) PluginOption {
	return func(p *Plugin) {
		p.cmd = cmd
	}
}

func WithPluginImplementation(plugin SecPluginable) PluginOption {
	return func(p *Plugin) {
		p.plugin = SecPipelineGRPCPlugin{Impl: plugin, Name: p.name}
	}
}

func WithGRPCPlugin() PluginOption {
	return func(p *Plugin) {
		p.plugin = SecPipelineGRPCPlugin{Name: p.name}
	}
}

func (p Plugin) Serve() {
	log := hclog.Default().Named(p.name)
	log.SetLevel(hclog.Debug)

	slog.Debug("start serving plugin", "names", p.name)
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: p.handshake,
		Plugins:         plugin.PluginSet{"plugin": p.plugin},
		GRPCServer:      plugin.DefaultGRPCServer,
		Logger:          log,
	})
	slog.Debug("stop serving plugin", "name", p.name)
}

func (p *Plugin) Connect() (SecPluginable, error) {
	log := hclog.Default().Named(p.name)
	log.SetLevel(hclog.Debug)
	p.client = plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig:  p.handshake,
		Plugins:          plugin.PluginSet{"plugin": p.plugin},
		Cmd:              p.cmd,
		AllowedProtocols: []plugin.Protocol{plugin.ProtocolGRPC},
		Managed:          true,
		Logger:           log,
	})
	cp, err := p.client.Client()
	if err != nil {
		slog.Error("failed to connect to plugin", "function", "Connect", "Object", "Plugin", "file", "pluginctl.go", "error", err)
		return nil, fmt.Errorf("failed to connect to plugin, %w", err)
	}
	res, err := cp.Dispense("plugin")
	if err != nil {
		slog.Error("failed to dispense plugin", "function", "Connect", "Object", "Plugin", "file", "pluginctl.go", "error", err)
		return nil, fmt.Errorf("failed to dispense plugin, %w", err)
	}

	resSec, ok := res.(SecPluginable)
	if !ok {
		slog.Error("failed to dispense plugin not a SecPluginable", "function", "Connect", "Object", "Plugin", "file", "pluginctl.go")
		return nil, fmt.Errorf("failed to dispense plugin  not a SecPluginable")
	}
	return resSec, nil
}

func (p *Plugin) Cleanup() {
	if p.client != nil {
		p.client.Kill()
		p.client = nil
	}
}

func CleanupClients() {
	plugin.CleanupClients()
}
