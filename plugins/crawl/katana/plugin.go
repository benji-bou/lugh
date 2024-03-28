package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

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
	return helper.ConfigurePtr(&Katana{}, opt...)
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
					err = crawler.Crawl(string(i.Data))
					if err != nil {
						eC <- fmt.Errorf("could not crawl %s: %w", string(i.Data), err)
					}
					return
				}

			}
		}
	})
}

func main() {
	helper.SetLog(slog.LevelError)
	plugin := pluginctl.NewPlugin("",
		pluginctl.WithPluginImplementation(NewKatana()),
	)
	plugin.Serve()
}
