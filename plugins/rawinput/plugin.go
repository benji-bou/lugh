package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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

func (wh RawInput) Produce(context context.Context, yield func(elem []byte) error) error {
	slog.Info("will send data", "data", wh.config.Data)
	return yield([]byte(wh.config.Data))
}

func main() {
	helper.SetLog(slog.LevelDebug, false)
	plugin := grpc.NewPlugin("rawInput",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromProducer(NewRawInput())),
	)
	plugin.Serve()
}
