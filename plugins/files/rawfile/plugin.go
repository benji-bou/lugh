package main

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/chantools"
)

type RawFileOption = helper.Option[RawFile]

type ConfigRawFile struct {
	Filepath string `json:"filepath"`
}

type RawFile struct {
	config ConfigRawFile
}

func NewRawFile(opt ...RawFileOption) *RawFile {
	return helper.ConfigurePtr(&RawFile{}, opt...)
}
func (mp RawFile) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *RawFile) Config(config []byte) error {
	configRawFile := ConfigRawFile{}
	err := json.Unmarshal(config, &configRawFile)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for RawFile config: %w", err)
	}
	mp.config = configRawFile
	return nil
}
func (mp RawFile) Run(context graph.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	basePath, err := os.UserHomeDir()
	if err != nil {
		basePath = "./"
	}
	defaultFilePath := filepath.Join(basePath, ".secpipeline", "result.txt")
	if mp.config.Filepath != "" {
		defaultFilePath = mp.config.Filepath
	}
	f, err := os.OpenFile(defaultFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0644)
	if err != nil {
		return nil, chantools.Once(fmt.Errorf("NewRawFile open file for  plugin failed, %w", err))
	}
	defer f.Close()
	for i := range input {
		_, err := f.Write(i)
		if err != nil {
			return nil, chantools.Once(fmt.Errorf("NewRawFile write file plugin failed, %w", err))
		}
	}
	return nil, nil
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("rawfile",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewRawFile())),
	)
	plugin.Serve()
}
