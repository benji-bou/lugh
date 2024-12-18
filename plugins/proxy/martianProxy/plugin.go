package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"os"
	"time"

	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/helper"

	martian "github.com/benji-bou/SecPipeline/plugins/proxy/martianProxy/martian"
	"github.com/benji-bou/chantools"
	"github.com/swaggest/jsonschema-go"
)

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

func (mp MartianPlugin) Run(ctx context.Context, _ <-chan []byte) (<-chan []byte, <-chan error) {
	slog.Info("MartianPlugin run")

	// We use the option WithNonManagedChannel because we want let the chantools.NewWriter handle the close
	return chantools.NewWithErr(func(dataC chan<- []byte, errC chan<- error, params ...any) {
		slog.Debug("started routine", "function", "Run", "plugin", "MartianPlugin")
		wC := chantools.NewWriter(dataC)
		defer wC.Close()

		opt, err := mp.getOptions(wC)
		if err != nil {
			errC <- err
			return
		}
		px, err := martian.NewProxy(":8080", ":4443", ":4242", opt...)
		if err != nil {
			errC <- err
			return
		}
		defer px.Close()

		slog.Debug("run martian proxy", "function", "Run", "plugin", "MartianPlugin")
		err = px.Run(ctx, true)
		if err != nil {
			slog.Error("proxy run failed", "function", "Run", "plugin", "MartianProxy", "error", err)
			errC <- err
		}
		slog.Debug("martian proxy stoped", "function", "Run", "plugin", "MartianPlugin")
	}, chantools.WithNonManagedChannel[[]byte](), chantools.WithName[[]byte]("martianProxyRoutine"))

}

func main() {

	helper.SetLog(slog.LevelDebug, true)
	plugin := grpc.NewPlugin("martianProxy",
		grpc.WithPluginImplementation(NewMartianPlugin()),
	)
	plugin.Serve()
}

func (mp MartianPlugin) getOptions(wC io.WriteCloser) ([]helper.OptionError[martian.Proxy], error) {

	opt := []helper.OptionError[martian.Proxy]{
		martian.WitDefaultWriter(wC),
		martian.WithMitmCertsFile(time.Hour*24*365, "SecPipeline", "SecPipeline", false,
			"/Users/benjaminbouachour/Private/Projects/SecPipeline/plugins/proxy/martianProxy/certs/cert.crt",
			"/Users/benjaminbouachour/Private/Projects/SecPipeline/plugins/proxy/martianProxy/certs/cert.key",
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
