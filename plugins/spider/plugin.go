package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"sync"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/chantools"
	spider "github.com/benji-bou/gospider/core"
)

type SpiderOption = helper.Option[Spider]

type Spider struct {
	workerCount int
	crawlerOpt  []spider.CrawlerOption
	sitemap     bool
	robots      bool
}

func NewSpider(opt ...SpiderOption) *Spider {
	return helper.ConfigurePtr(&Spider{workerCount: 5}, opt...)
}

func (mp Spider) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *Spider) Config([]byte) error {
	return nil
}

// Run expect a json array of strings listing sites to visists
func (mp Spider) Run(context graph.Context, input <-chan []byte) (<-chan []byte, <-chan error) {

	return chantools.NewWithErr(func(c chan<- []byte, eC chan<- error, params ...any) {
		inputSiteC := make(chan string)
		mp.Worker(context, inputSiteC)
		for {
			select {
			case <-context.Done():
				return
			case i := <-input:
				inputSites := []string{}
				err := json.Unmarshal(i, &inputSites)
				if err != nil {
					eC <- fmt.Errorf("failed to unmarshal input data: %w", err)
				} else {
					for _, s := range inputSites {
						inputSiteC <- s
					}
				}
			}
		}

	})
}

func (mp Spider) Worker(ctx context.Context, site <-chan string) (<-chan []byte, <-chan error) {
	return chantools.NewWithErr(func(c chan<- []byte, eC chan<- error, params ...any) {
		var waitWorker sync.WaitGroup
		waitWorker.Add(mp.workerCount)
		for i := 0; i < mp.workerCount; i++ {
			go func(ctx context.Context, site <-chan string) {
				defer waitWorker.Done()
				crawler := spider.NewCrawler(mp.crawlerOpt...)
				crawler.StreamScrawl(ctx, site)
			}(ctx, site)
		}
		waitWorker.Wait()
	})
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("spider",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewSpider())),
	)
	plugin.Serve()
}
