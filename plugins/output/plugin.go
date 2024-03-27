package main

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
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

func (mp Output) Run(context context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return chantools.NewWithErr(func(c chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
		for {
			select {
			case <-context.Done():
				return
			case i := <-input:
				slog.Info(fmt.Sprintf("%s", string(i.Data)))
			}
		}
	})
}

func main() {
	helper.SetLog(slog.LevelInfo)
	plugin := pluginctl.NewPlugin("",
		pluginctl.WithPluginImplementation(NewOutput()),
	)
	plugin.Serve()
}
