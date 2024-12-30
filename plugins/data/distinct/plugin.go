package main

import (
	"bytes"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	_ "net/http/pprof"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/benji-bou/diwo"
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
	buffSizeMax      int
	inmem            map[[16]byte]struct{}
	goTemplateFilter *template.Template
}

func NewMemFilter(opt ...MemFilterOption) *MemFilter {
	return helper.ConfigurePtr(&MemFilter{inmem: map[[16]byte]struct{}{}}, append([]MemFilterOption{DefaultBuffSize()}, opt...)...)
}

func (mp *MemFilter) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *MemFilter) Config(config []byte) error {
	configFilter := struct {
		GoTemplateFilter string `json:"goTemplateFilter"`
	}{}
	err := json.Unmarshal(config, &configFilter)
	if err != nil {
		return fmt.Errorf("couldn't unmarshal Distinct plugin config because %w", err)
	}
	tpl, err := template.New("distinct").Parse(configFilter.GoTemplateFilter)
	if err != nil {
		return fmt.Errorf("couldn't generate Distinct go template pattern because %w", err)
	}
	mp.goTemplateFilter = tpl
	return nil
}

func (mp *MemFilter) Run(context graph.Context, input <-chan []byte) <-chan []byte {
	return diwo.New(func(c chan<- []byte) { {
		for {
			select {
			case <-context.Done():
				return
			case i, ok := <-input:
				if !ok {
					return
				}
				slog.Debug("received data", "data", string(i))
				buff := &bytes.Buffer{}
				if mp.goTemplateFilter != nil {
					mp.goTemplateFilter.Execute(buff, i)
				}
				if buff.Len() == 0 {
					buff.Write(i)
				}

				hash := md5.Sum(buff.Bytes())
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
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("distinct",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewMemFilter())),
	)
	plugin.Serve()
}
