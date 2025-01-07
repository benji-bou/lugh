package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/projectdiscovery/katana/pkg/engine/standard"
	"github.com/projectdiscovery/katana/pkg/output"
	"github.com/projectdiscovery/katana/pkg/types"
)

type KatanaOption = helper.Option[Katana]

type Katana struct {
	option *types.Options
}

func NewKatana(opt ...KatanaOption) *Katana {
	options := &types.Options{
		MaxDepth:     3,               // Maximum depth to crawl
		FieldScope:   "rdn",           // Crawling Scope Field
		BodyReadSize: math.MaxInt,     // Maximum response size to read
		Timeout:      10,              // Timeout is the time to wait for request in seconds
		Concurrency:  10,              // Concurrency is the number of concurrent crawling goroutines
		Parallelism:  10,              // Parallelism is the number of urls processing goroutines
		Delay:        0,               // Delay is the delay between each crawl requests in seconds
		RateLimit:    150,             // Maximum requests to send per second
		Strategy:     "breadth-first", // Visit strategy (depth-first, breadth-first)
	}
	return helper.ConfigurePtr(&Katana{option: options}, opt...)
}

func (*Katana) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *Katana) Config(conf []byte) error {
	err := json.Unmarshal(conf, mp.option)
	if err != nil {
		return err
	}
	return nil
}

func (mp *Katana) Work(_ context.Context, input []byte, yield func(elem []byte) error) error {
	mp.option.OnResult = func(r output.Result) {
		res, err := json.Marshal(r)
		if err != nil {
			slog.Error("", "error", fmt.Errorf("failed to Marshal katan output into json, %w", err))
			return
		}
		err = yield(res)
		if err != nil {
			slog.Error("failed to yield result", "error", err)
		}
	}
	crawlerOptions, err := types.NewCrawlerOptions(mp.option)
	if err != nil {
		return fmt.Errorf("build crawlerOptions failed: %w", err)
	}
	defer crawlerOptions.Close()
	crawler, err := standard.New(crawlerOptions)
	if err != nil {
		return fmt.Errorf("build crawler failed: %w", err)
	}
	defer crawler.Close()
	err = crawler.Crawl(string(input))
	if err != nil {
		return fmt.Errorf("could not crawl %s: %w", string(input), err)
	}
	return nil
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("Katana",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromWorker(NewKatana())),
	)
	plugin.Serve()
}
