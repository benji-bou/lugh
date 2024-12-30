package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/benji-bou/diwo"
	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
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

func (wh RawInput) Run(context graph.Context, input <-chan []byte) <-chan []byte {
	return diwo.New(func(cDataStream chan<- []byte, eC chan<- error, params ...any) {
		slog.Info("will send data", "data", wh.config.Data)
		cDataStream <- []byte(wh.config.Data)
		slog.Info("did send data", "data", wh.config.Data)
	}, diwo.WithName[[]byte]("raw-input"))

}

func main() {
	helper.SetLog(slog.LevelDebug, false)
	plugin := grpc.NewPlugin("rawInput",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewRawInput())),
	)
	plugin.Serve()
}
