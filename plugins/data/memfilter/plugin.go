package main

import (
	"context"
	"crypto/md5"
	"log/slog"
	"net/http"
	_ "net/http/pprof"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
)

type MemFilterOption = helper.Option[MemFilter]

func MaxBuffSize(buffSize int) MemFilterOption {
	return func(configure *MemFilter) {
		configure.buffSizeMax = buffSize
	}
}
func IllimitedBuffSize() MemFilterOption {
	return func(configure *MemFilter) {
		configure.buffSizeMax = -1
	}
}

func DefaultBuffSize() MemFilterOption {
	return IllimitedBuffSize()
	// return func(configure *MemFilter) {
	// 	configure.buffSizeMax = 1024
	// }
}

type MemFilter struct {
	buffSizeMax int
	inmem       map[[16]byte]struct{}
}

func NewMemFilter(opt ...MemFilterOption) *MemFilter {
	return helper.ConfigurePtr(&MemFilter{inmem: map[[16]byte]struct{}{}}, append([]MemFilterOption{DefaultBuffSize()}, opt...)...)
}

func (mp MemFilter) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp MemFilter) Config([]byte) error {
	return nil
}

func (mp *MemFilter) Run(context context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return chantools.NewWithErr(func(c chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
		for {
			select {
			case <-context.Done():
				return
			case i := <-input:
				slog.Debug("received data", "data", string(i.Data))
				hash := md5.Sum(i.Data)
				if _, exists := mp.inmem[hash]; exists {
					continue
				}
				if len(mp.inmem) >= mp.buffSizeMax && mp.buffSizeMax > 0 {
					for toDel := range mp.inmem {
						delete(mp.inmem, toDel)
						break
					}
				}
				mp.inmem[hash] = struct{}{}
				c <- i
			}
		}
	})
}

func main() {
	go func() {
		http.ListenAndServe("localhost:6061", nil)
	}()
	helper.SetLog(slog.LevelError)
	plugin := pluginctl.NewPlugin("",
		pluginctl.WithPluginImplementation(NewMemFilter()),
	)
	plugin.Serve()
}
