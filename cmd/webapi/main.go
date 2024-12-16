package main

import (
	"log"
	"log/slog"

	"os"

	"github.com/benji-bou/SecPipeline/core/api"
	"github.com/benji-bou/SecPipeline/core/api/ctrl"
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
			helper.SetLog(slog.LevelDebug, true)
			return api.Listen(ctrl.NewPipeline())
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
