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

func (mp Output) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp Output) Config([]byte) error {
	return nil
}

func (mp Output) Consume(context context.Context, input <-chan []byte) error {
	for {
		select {
		case <-context.Done():
			return nil
		case i, ok := <-input:
			if !ok {
				return nil
			}
			_, err := fmt.Printf("%s", string(i))
			if err != nil {
				return fmt.Errorf("output failed %w", err)
			}
		}
	}
}

func main() {
	helper.SetLog(slog.LevelDebug, true)
	plugin := grpc.NewPlugin("output",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromConsumer(NewOutput())),
	)
	plugin.Serve()
}
