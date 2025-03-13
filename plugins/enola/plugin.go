package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/benji-bou/enola"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
)

type ConfigEnola struct {
	Sites []string `json:"sites"`
}

type Enola struct {
	config ConfigEnola
}

// Work implements pluginapi.WorkerPluginable.
func (mp *Enola) Work(ctx context.Context, input []byte, yield func(elem []byte) error) error {
	ctxEnola, cancel := context.WithCancel(ctx)
	defer cancel()
	sh, err := enola.New(ctxEnola)
	if err != nil {
		return fmt.Errorf("failed to create enola shell: %w", err)
	}
	sites := strings.Join(mp.config.Sites, " ")
	resC, err := sh.SetSite(sites).Check(string(input))
	if err != nil {
		return fmt.Errorf("enola failed to check input: %w", err)
	}
	for res := range resC {
		if res.Found {
			resRawJSON, err := json.Marshal(res)
			if err != nil {
				slog.Warn("failed to marshal enola result", "error", err)
				continue
			}
			if err := yield(resRawJSON); err != nil {
				return fmt.Errorf("enola stoped because yield failed: %w", err)
			}
		}
	}
	return nil
}

func (mp *Enola) Config(config []byte) error {
	configEnola := ConfigEnola{}
	err := json.Unmarshal(config, &configEnola)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for Enola config: %w", err)
	}
	mp.config = configEnola
	return nil
}

type EnolaOption = helper.Option[Enola]

func NewEnola(opt ...EnolaOption) *Enola {
	return helper.ConfigurePtr(&Enola{}, opt...)
}

func (wh Enola) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func main() {
	helper.SetLog(slog.LevelDebug, false)
	plugin := grpc.NewPlugin("enola",
		grpc.WithPluginImplementation(pluginapi.NewWorker(NewEnola())),
	)
	plugin.Serve()
}
