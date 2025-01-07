package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/zricethezav/gitleaks/v8/detect"
)

type LeaksPluginOption = helper.Option[LeaksPlugin]

type LeaksPlugin struct {
	// dete
}

func NewLeaksPlugin(opt ...LeaksPluginOption) LeaksPlugin {
	return helper.Configure(LeaksPlugin{}, opt...)
}

func (LeaksPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (LeaksPlugin) Config([]byte) error {
	return nil
}

func (LeaksPlugin) Work(_ context.Context, input []byte, yield func(elem []byte) error) error {
	slog.Info("start run", "function", "Run", "plugin", "LeaksPlugin")
	detector, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		slog.Error("failed to start Run", "function", "Run", "plugin", "LeaksPlugin", "error", err)
		return fmt.Errorf("run Leaks failed, unable to create detector %w", err)
	}
	slog.Debug("received fragment to search for leaks", "function", "Run", "plugin", "LeaksPlugin")
	res := detector.Detect(detect.Fragment{Raw: string(input)})
	rawJSON, err := json.Marshal(res)
	if err != nil {
		slog.Error("failed to json marshal report finding", "function", "Run", "plugin", "LeaksPlugin", "error", err)
		return fmt.Errorf("failed to json marshal report finding: %w", err)
	}
	return yield(rawJSON)
}

func main() {
	helper.SetLog(slog.LevelDebug, true)

	p := grpc.NewPlugin("leaks",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromWorker(NewLeaksPlugin())),
	)

	p.Serve()
}
