package main

import (
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/benji-bou/diwo"
	"github.com/zricethezav/gitleaks/v8/detect"
)

type LeaksPluginOption = helper.Option[LeaksPlugin]

type LeaksPlugin struct {
	// dete
}

func NewLeaksPlugin(opt ...LeaksPluginOption) LeaksPlugin {
	return helper.Configure(LeaksPlugin{}, opt...)
}

func (mp LeaksPlugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp LeaksPlugin) Config([]byte) error {
	return nil
}

func (mp LeaksPlugin) Run(context graph.Context, input <-chan []byte) <-chan []byte {
	slog.Info("start run", "function", "Run", "plugin", "LeaksPlugin")
	detector, err := detect.NewDetectorDefaultConfig()
	if err != nil {
		slog.Error("failed to start Run", "function", "Run", "plugin", "LeaksPlugin", "error", err)
		return nil, diwo.Once(fmt.Errorf("Run Leaks failed, unable to create detector %w", err))
	}
	return diwo.New(func(c chan<- []byte) { {
		for i := range input {
			slog.Debug("received fragment to search for leaks", "function", "Run", "plugin", "LeaksPlugin")
			res := detector.Detect(detect.Fragment{Raw: string(i)})
			rawJson, err := json.Marshal(res)
			if err != nil {
				slog.Error("failed to json marshal report finding", "function", "Run", "plugin", "LeaksPlugin", "error", err)
				eC <- fmt.Errorf("failed to json marshal report finding: %w", err)

			} else {
				c <- rawJson
			}
		}
		slog.Debug("leaving goroutine", "funtion", "Run", "plugin", "leaks")
	}, diwo.WithParam[[]byte](detector), diwo.WithName[[]byte]("leaks"))
}

func main() {
	helper.SetLog(slog.LevelDebug, true)

	p := grpc.NewPlugin("leaks",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewLeaksPlugin())),
	)

	p.Serve()
}
