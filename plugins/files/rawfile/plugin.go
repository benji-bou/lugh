package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
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

func (*RawFile) GetInputSchema() ([]byte, error) {
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

func (mp *RawFile) Consume(_ context.Context, input []byte) error {
	basePath, err := os.UserHomeDir()
	if err != nil {
		basePath = "./"
	}
	defaultFilePath := filepath.Join(basePath, ".lugh", "result.txt")
	if mp.config.Filepath != "" {
		defaultFilePath = mp.config.Filepath
	}
	f, err := os.OpenFile(defaultFilePath, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600) //nolint:mnd // basic file permision
	if err != nil {
		return fmt.Errorf("NewRawFile open file for  plugin failed, %w", err)
	}
	defer f.Close()
	_, err = f.Write(input)
	if err != nil {
		return fmt.Errorf("NewRawFile write file plugin failed, %w", err)
	}
	return nil
}

func main() {
	helper.SetLog(slog.LevelDebug, false)
	plugin := grpc.NewPlugin("rawfile",
		grpc.WithPluginImplementation(pluginapi.NewConfigurableConsumer(NewRawFile())),
	)
	plugin.Serve()
}
