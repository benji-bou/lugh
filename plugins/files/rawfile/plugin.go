package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
)

type RawFileOption = helper.Option[RawFile]

type RawFile struct{}

func NewRawFile(opt ...RawFileOption) RawFile {
	return helper.Configure(RawFile{}, opt...)
}
func (mp RawFile) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp RawFile) Config([]byte) error {

	return nil
}
func (mp RawFile) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	f, err := os.OpenFile("result.txt", os.O_WRONLY, 0644)
	if err != nil {
		return nil, chantools.Once(fmt.Errorf("NewRawFile open file for  plugin failed, %w", err))
	}
	defer f.Close()
	for i := range input {
		_, err := f.Write(i.Data)
		if err != nil {
			return nil, chantools.Once(fmt.Errorf("NewRawFile write file plugin failed, %w", err))
		}
	}
	return nil, nil
}

func main() {
	helper.SetLog(slog.LevelError)
	plugin := pluginctl.NewPlugin("martianProxy",
		pluginctl.WithPluginImplementation(NewRawFile()),
	)
	plugin.Serve()
}
