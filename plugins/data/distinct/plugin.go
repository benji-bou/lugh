package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"html/template"
	"log/slog"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
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
}

type MemFilter struct {
	buffSizeMax      int
	inmem            map[[32]byte]struct{}
	goTemplateFilter *template.Template
}

func NewMemFilter(opt ...MemFilterOption) *MemFilter {
	return helper.ConfigurePtr(&MemFilter{inmem: map[[32]byte]struct{}{}}, append([]MemFilterOption{DefaultBuffSize()}, opt...)...)
}

func (*MemFilter) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *MemFilter) Config(config []byte) error {
	slog.Debug("config", "configuration", string(config))
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
	slog.Debug("config done", "template", configFilter.GoTemplateFilter)
	mp.goTemplateFilter = tpl
	return nil
}

func (mp *MemFilter) Work(_ context.Context, input []byte, yield func(elem []byte) error) error {
	slog.Debug("received data", "data", string(input))
	buff := &bytes.Buffer{}
	if mp.goTemplateFilter != nil {
		slog.Debug("will excute template")
		err := mp.goTemplateFilter.Execute(buff, string(input))
		if err != nil {
			return fmt.Errorf("couldn't execute Distinct go template pattern because %w", err)
		}
	}
	if buff.Len() == 0 {
		slog.Debug("template execution result len == 0 directly write input to buff")
		buff.Write(input)
	}

	hash := sha256.Sum256(buff.Bytes())
	if _, exists := mp.inmem[hash]; exists {
		slog.Debug("generated hash already exists", "hash", hash)
		return nil
	}
	if len(mp.inmem) >= mp.buffSizeMax && mp.buffSizeMax > 0 {
		for toDel := range mp.inmem {
			delete(mp.inmem, toDel)
			break
		}
	}
	mp.inmem[hash] = struct{}{}
	slog.Debug("yield value  hash does not exists", "hash", hash)
	return yield(input)
}

func main() {
	helper.SetLog(slog.LevelDebug, true)
	plugin := grpc.NewPlugin("distinct",
		grpc.WithPluginImplementation(pluginapi.NewConfigurableWorker(NewMemFilter())),
	)
	plugin.Serve()
}
