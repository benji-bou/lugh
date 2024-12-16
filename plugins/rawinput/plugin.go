package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
)

type ConfigRawInput struct {
	Data string `json:"data"`
}

type RawInput struct {
	config ConfigRawInput
}

func (mp *RawInput) Config(config []byte) error {
	configRawInput := ConfigRawInput{}
	err := json.Unmarshal(config, &configRawInput)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for RawInput config: %w", err)
	}
	mp.config = configRawInput
	return nil
}

type RawInputOption = helper.Option[RawInput]

func NewRawInput(opt ...RawInputOption) *RawInput {
	return helper.ConfigurePtr(&RawInput{}, opt...)
}
func (wh RawInput) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (wh RawInput) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return chantools.NewWithErr(func(cDataStream chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
		slog.Info("will send data", "data", wh.config.Data)
		cDataStream <- &pluginctl.DataStream{Data: []byte(wh.config.Data), ParentSrc: "rawInput"}
		slog.Info("did send data", "data", wh.config.Data)
	}, chantools.WithName[*pluginctl.DataStream]("raw-input"))

}

func main() {
	helper.SetLog(slog.LevelDebug, true)
	plugin := pluginctl.NewPlugin("rawInput",
		pluginctl.WithPluginImplementation(NewRawInput()),
	)
	plugin.Serve()
}
