package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/labstack/echo/v4"
)

type ConfigWebhook struct {
	Path   string `json:"path"`
	Method string `json:"method"`
}

type Webhook struct {
	config ConfigWebhook
}

type WebhookOption = helper.Option[Webhook]

func NewWebhook(opt ...WebhookOption) *Webhook {
	return helper.ConfigurePtr(&Webhook{}, opt...)
}

func (wh *Webhook) Config(config []byte) error {
	configWebhook := ConfigWebhook{}
	err := json.Unmarshal(config, &configWebhook)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for Webhook config: %w", err)
	}
	wh.config = configWebhook
	return nil
}

func (*Webhook) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (wh *Webhook) Produce(ctx context.Context, yield func(elem []byte) error) error {
	method := wh.config.Method
	if method == "" {
		method = "POST"
	}
	path := wh.config.Path
	if path == "" {
		path = "/hook"
	}
	return helper.RunServer(helper.WithContext(ctx), helper.WithAdd(method, path, func(c echo.Context) error {
		body := c.Request().Body
		defer body.Close()
		bRaw, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		err = yield(bRaw)
		if err != nil {
			return err
		}
		return nil
	}))
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("webhook",
		grpc.WithPluginImplementation(pluginapi.NewProducer(NewWebhook())),
	)
	plugin.Serve()
}
