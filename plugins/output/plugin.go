package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
)

type OutputOption = helper.Option[Output]

type Output struct{}

func NewOutput(opt ...OutputOption) Output {
	return helper.Configure(Output{}, opt...)
}

func (Output) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (Output) Config([]byte) error {
	return nil
}

func (Output) Consume(ctx context.Context, input []byte) error {
	_, err := fmt.Printf("%s", string(input))
	if err != nil {
		return fmt.Errorf("output failed %w", err)
	}
	return nil
}

func main() {
	helper.SetLog(slog.LevelDebug, true)
	plugin := grpc.NewPlugin("output",
		grpc.WithPluginImplementation(pluginapi.NewConsumer(NewOutput())),
	)
	plugin.Serve()
}
