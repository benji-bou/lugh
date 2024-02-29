package main

import (
	"encoding/json"
	"log/slog"
	"os"
	"time"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	martian "github.com/benji-bou/SecPipeline/plugins/proxy/martianProxy/martian"
	"github.com/benji-bou/chantools"
	"github.com/swaggest/jsonschema-go"
)

type MartianInputConfig struct {
	scope        string `json:"scope" required:"false"`
	listenAddr   string
	outputFormat string
	_            struct{} `title:"Martian HttpProxy plugin" description:"Input configuration for Martion HTTP Proxy plugin"`
}

type MartianPlugin struct {
	inputFormat []byte
}

func NewMartianPlugin() MartianPlugin {
	r := jsonschema.Reflector{}
	schema, err := r.Reflect(MartianInputConfig{})
	if err != nil {
		slog.Error(pluginctl.ErrJsonSchemaConvertion.Error(), "plugin", "MartianHttpProxy", "type", "MartianInputConfig", "errors", err)
		os.Exit(-1)
	}
	j, err := json.Marshal(schema)
	if err != nil {
		slog.Error(pluginctl.ErrJsonConvertion.Error(), "plugin", "MartianHttpProxy", "type", "MartianInputConfig", "errors", err)
		os.Exit(-1)
	}
	return MartianPlugin{inputFormat: j}
}

func (mp MartianPlugin) GetInputSchema() ([]byte, error) {
	slog.Info("MartianPlugin GetInputSchema")
	return mp.inputFormat, nil
}

func (mp MartianPlugin) Config(config []byte) error {
	return nil
}

func (mp MartianPlugin) Run(_ <-chan []byte) (<-chan []byte, <-chan error) {
	slog.Info("MartianPlugin run")
	return chantools.NewWithErr(func(dataC chan<- []byte, errC chan<- error, params ...any) {
		slog.Debug("started routine", "function", "Run", "plugin", "MartianPlugin")
		wC := chantools.NewWriter(dataC)
		defer wC.Close()

		px := martian.NewProxy(":8080", ":4443", ":4242",
			martian.WithMitmCertsFile(time.Hour*24*365, "SecPipeline", "SecPipeline", false,
				"/Users/benjaminbouachour/Private/Projects/SecPipeline/plugins/proxy/martianProxy/certs/cert.crt",
				"/Users/benjaminbouachour/Private/Projects/SecPipeline/plugins/proxy/martianProxy/certs/cert.key",
				false,
			),

			martian.WithHarWriterLog(wC),
			martian.WithLogLevel(2),
			// martian.WithStdLog(),
		)
		defer px.Close()
		slog.Debug("run martian proxy", "function", "Run", "plugin", "MartianPlugin")
		err := px.Run(true)
		if err != nil {
			slog.Error("proxy run failed", "function", "Run", "plugin", "MartianProxy", "error", err)
			errC <- err
		}
		slog.Debug("martian proxy stoped", "function", "Run", "plugin", "MartianPlugin")
	})
}

func main() {
	helper.SetLog(slog.LevelError)
	plugin := pluginctl.NewPlugin("martianProxy",
		pluginctl.WithPluginImplementation(NewMartianPlugin()),
	)
	plugin.Serve()
}
