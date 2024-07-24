package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
	"github.com/labstack/echo/v4"
)

type ConfigWebhook struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type Webhook struct {
	config ConfigWebhook
}

func (mp *Webhook) Config(config []byte) error {
	configWebhook := ConfigWebhook{}
	err := json.Unmarshal(config, &configWebhook)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for Webhook config: %w", err)
	}
	mp.config = configWebhook
	return nil
}

type WebhookOption = helper.Option[Webhook]

func NewWebhook(opt ...WebhookOption) *Webhook {
	return helper.ConfigurePtr(&Webhook{}, opt...)
}
func (wh Webhook) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (wh Webhook) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
	return chantools.NewWithErr(func(cDataStream chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {
		method := wh.config.Method
		if method == "" {
			method = "POST"
		}
		path := wh.config.Path
		if path == "" {
			path = "/hook"
		}
		helper.RunServer(helper.WithAdd(method, path, func(c echo.Context) error {
			body := c.Request().Body
			defer body.Close()
			bRaw, err := io.ReadAll(body)
			if err != nil {
				eC <- err
			} else {
				cDataStream <- &pluginctl.DataStream{Data: bRaw, ParentSrc: "webhook"}
			}
			return nil
		}))
	})

}

func main() {
	helper.SetLog(slog.LevelError)
	plugin := pluginctl.NewPlugin("Webhook",
		pluginctl.WithPluginImplementation(NewWebhook()),
	)
	plugin.Serve()
}
