package main

import (
	"context"
	"log"
	"log/slog"

	"os"
	"os/signal"

	"github.com/benji-bou/SecPipeline/core"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "SecPipeline",
		Usage: "SecPipeline can be use to construct cyber security pipeline based on modules",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "template",
				Aliases: []string{"t"},
				Usage:   "Pipeline template to execute",
			},
		},
		Action: func(c *cli.Context) error {

			helper.SetLog(slog.LevelInfo)
			template := c.String("template")
			slog.Info("", "template", template)
			defer func() {
				pluginctl.CleanupClients()
			}()
			tpl, err := core.NewFileTemplate("/Users/benjaminbouachour/Private/Projects/SecPipeline/templates/test.yml")

			if err != nil {
				return err
			}
			tpl.Start(context.Background())

			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, os.Interrupt)
			for {
				select {
				case <-sigc:
					return nil
				}
			}
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
