package main

import (
	"context"
	"io"
	"log"
	"log/slog"

	"os"

	"github.com/benji-bou/SecPipeline/core"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/labstack/echo/v4"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "SecPipeline",
		Usage: "SecPipeline can be use to construct cyber security pipeline based on modules",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "draw-graph-only",
				Usage: "Only construct pipeline graph and drow it in DOT notation. To display use `dot -Tsvg <filepath>`",
			},
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Pipeline template to execute",
			},
		},
		Action: func(c *cli.Context) error {
			defer func() {
				pluginctl.CleanupClients()
			}()
			helper.SetLog(slog.LevelDebug)
			return StartAPI(c)
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func StartAPI(c *cli.Context) error {
	return helper.RunServer(helper.WithPost("/run", func(c echo.Context) error {
		body := c.Request().Body
		defer body.Close()
		content, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		tpl, err := core.NewRawTemplate(content)
		if err != nil {
			return err
		}
		err, errC := tpl.Start(context.Background())
		if err != nil {
			slog.Error("failed to start template", "error", err)
			return err
		}
		for {
			select {
			case e, ok := <-errC:
				if !ok {
					return nil
				}
				slog.Error("an error occured in a stage", "error", e)
			}
		}
	}))

}
