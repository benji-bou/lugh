package main

import (
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/benji-bou/SecPipeline/core"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
	"github.com/k0kubun/pp/v3"

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
			defer pluginctl.CleanupClients()
			martian, err := pluginctl.NewPlugin("martianProxy", pluginctl.WithPath("/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins")).Connect()
			if err != nil {
				slog.Error("failed to load martian plugin", "function", "main", "file", "main.go", "error", err)
				return fmt.Errorf("failed to load martian plugin, %w", err)
			}
			leaks, err := pluginctl.NewPlugin("leaks", pluginctl.WithPath("/Users/benjaminbouachour/Private/Projects/SecPipeline/bin/plugins")).Connect()
			if err != nil {
				slog.Error("failed to load leaks plugin", "function", "main", "file", "main.go", "error", err)
				return fmt.Errorf("failed to load leaks plugin, %w", err)
			}
			mDataC, mErrC := martian.Run(nil)

			lDataC, lErrC := core.NewPipe(mDataC, leaks).Pipe()
			aggrErrC := chantools.Merge(mErrC, lErrC)
			sigc := make(chan os.Signal, 1)
			signal.Notify(sigc, os.Interrupt)
			for {
				select {
				case err := <-aggrErrC:
					slog.Error("received error from plugin", "function", "main", "error", err)
					return err
				case leak := <-lDataC:
					pp.Print(string(leak))
				case <-sigc:
					return nil
				}
			}

			return nil
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}
