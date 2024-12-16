package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
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

func (mp Katana) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *Katana) Config(conf []byte) error {
	err := json.Unmarshal(conf, mp.option)
	if err != nil {
		return err
	}
	return nil
}

func (mp *Katana) Run(context context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return chantools.NewWithErr(func(c chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
		mp.option.OnResult = func(r output.Result) {
			res, err := json.Marshal(r)
			if err != nil {
				eC <- fmt.Errorf("failed to Marshal katan output into json, %w", err)
				return
			}
			c <- &pluginctl.DataStream{
				Data:      res,
				TotalLen:  int64(len(res)),
				ParentSrc: "Katana",
			}
		}
		crawlerOptions, err := types.NewCrawlerOptions(mp.option)
		if err != nil {
			eC <- fmt.Errorf("build crawlerOptions failed: %w", err)
			return
		}
		defer crawlerOptions.Close()
		crawler, err := standard.New(crawlerOptions)
		if err != nil {
			eC <- fmt.Errorf("build crawler failed: %w", err)
			return
		}
		defer crawler.Close()
		for {
			select {
			case <-context.Done():
				return
			case i, ok := <-input:
				if !ok {
					return
				}
				err = crawler.Crawl(string(i.Data))
				if err != nil {
					slog.Error(fmt.Sprintf("could not crawl %s: %s", string(i.Data), err.Error()))
					// eC <- fmt.Errorf("could not crawl %s: %w", string(i.Data), err)
				}

			}
		}
	})
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := pluginctl.NewPlugin("",
		pluginctl.WithPluginImplementation(NewKatana()),
	)
	plugin.Serve()
}
