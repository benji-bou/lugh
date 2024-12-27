package main

import (
	"fmt"
	"log/slog"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/chantools"
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

func (mp Output) Run(context graph.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	return chantools.NewWithErr(func(c chan<- []byte, eC chan<- error, params ...any) {
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
