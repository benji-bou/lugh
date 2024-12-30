package main

import (
	"fmt"
	"log/slog"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/benji-bou/diwo"
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

func (mp Output) Run(context graph.Context, input <-chan []byte) <-chan []byte {
	return diwo.New(func(c chan<- []byte) { {
		for {
			select {
			case <-context.Done():
				return
			case i, ok := <-input:
				if !ok {
					return
				}
				fmt.Printf("%s", string(i))
			}
		}
	})
}

func main() {
	helper.SetLog(slog.LevelDebug, true)
	plugin := grpc.NewPlugin("output",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewOutput())),
	)
	plugin.Serve()
}
