package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"

	martian "github.com/benji-bou/lugh/plugins/proxy/martianProxy/martian"
	"github.com/swaggest/jsonschema-go"
)

type YieldWriter func(elem []byte) error

func (y YieldWriter) Write(data []byte) (int, error) {
	err := y(data)
	if err != nil {
		return 0, err
	}
	return len(data), nil
}

type MartianInputConfig struct {
	Modifier json.RawMessage `json:"modifier"`
}

type MartianPlugin struct {
	inputFormat []byte
	config      MartianInputConfig
}

func NewMartianPlugin() *MartianPlugin {
	r := jsonschema.Reflector{}
	schema, err := r.Reflect(MartianInputConfig{})
	if err != nil {
		slog.Error(grpc.ErrJsonSchemaConvertion.Error(), "plugin", "MartianHttpProxy", "type", "MartianInputConfig", "errors", err)
		os.Exit(-1)
	}
	j, err := json.Marshal(schema)
	if err != nil {
		slog.Error(grpc.ErrJsonConvertion.Error(), "plugin", "MartianHttpProxy", "type", "MartianInputConfig", "errors", err)
		os.Exit(-1)
	}
	return &MartianPlugin{inputFormat: j}
}

func (mp *MartianPlugin) GetInputSchema() ([]byte, error) {
	slog.Info("MartianPlugin GetInputSchema")
	return mp.inputFormat, nil
}

func (mp *MartianPlugin) Config(config []byte) error {
	configMartian := MartianInputConfig{}
	err := json.Unmarshal(config, &configMartian)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for MartianInputConfig because %w", err)
	}
	mp.config = configMartian
	return nil
}

func (mp *MartianPlugin) Produce(ctx context.Context, yield func(elem []byte) error) error {
	slog.Info("MartianPlugin run")
	// We use the option WithNonManagedChannel because we want let the diwo.NewWriter handle the close

	slog.Debug("started routine", "function", "Run", "plugin", "MartianPlugin")

	opt, err := mp.getOptions(YieldWriter(yield))
	if err != nil {
		return fmt.Errorf("martian initalization failed %w", err)
	}
	px, err := martian.NewProxy(":8080", ":4443", ":4242", opt...)
	if err != nil {
		return fmt.Errorf("martian creating proxy failed %w", err)
	}
	defer px.Close()
	slog.Debug("run martian proxy", "function", "Run", "plugin", "MartianPlugin")
	err = px.Run(ctx, true)
	if err != nil {
		return fmt.Errorf("martian proxy failed %w", err)
	}
	slog.Debug("martian proxy stoped", "function", "Run", "plugin", "MartianPlugin")
	return nil
}

func main() {

	helper.SetLog(slog.LevelDebug, true)
	plugin := grpc.NewPlugin("martianProxy",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromProducer(NewMartianPlugin())),
	)
	plugin.Serve()
}

func (mp MartianPlugin) getOptions(wC io.Writer) ([]helper.OptionError[martian.Proxy], error) {

	opt := []helper.OptionError[martian.Proxy]{
		martian.WitDefaultWriter(wC),
		martian.WithMitmCertsFile(time.Hour*24*365, "lugh", "lugh", false,
			"/Users/benjaminbouachour/Private/Projects/lugh/plugins/proxy/martianProxy/certs/cert.crt",
			"/Users/benjaminbouachour/Private/Projects/lugh/plugins/proxy/martianProxy/certs/cert.key",
			false,
		),
	}
	if len(mp.config.Modifier) > 0 {
		opt = append(opt, martian.WithModifiers([]byte(mp.config.Modifier)))
	} else {
		opt = append(opt,
			martian.WithHarWriterLog(wC),
			martian.WithLogLevel(2),
		)
	}

	return opt, nil
}
