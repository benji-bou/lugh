package main

import (
	"context"
	"log"
	"log/slog"

	"os"
	"os/signal"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/template"
	"github.com/benji-bou/SecPipeline/helper"

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
				Name:     "template",
				Aliases:  []string{"t"},
				Usage:    "Pipeline template to execute",
				Required: true,
			},
		},
		Action: func(c *cli.Context) error {
			defer func() {
				grpc.CleanupClients()
			}()
			helper.SetLog(slog.LevelDebug, false)
			if c.IsSet("draw-graph-only") {
				return DrawGraphOnly(c)
			} else {
				return RunTemplate(c)
			}
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func DrawGraphOnly(c *cli.Context) error {
	tplPath := c.String("template")
	tpl, err := template.NewFileTemplate(tplPath)
	if err != nil {
		return err
	}

	g := graph.NewGraph(graph.WithIOWorkerVertexIterator(tpl.WorkerVertexIterator()))

	return g.DrawGraph(c.String("draw-graph-only"))
}

func RunTemplate(c *cli.Context) error {
	tplPath := c.String("template")
	tpl, err := template.NewFileTemplate(tplPath)
	if err != nil {
		slog.Error("failed to start template", "error", err)
		return err
	}
	g := graph.NewGraph(graph.WithIOWorkerVertexIterator(tpl.WorkerVertexIterator()))
	errC := g.Run(context.Background())
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	for {
		select {
		case e, ok := <-errC:
			if !ok {
				return nil
			}
			slog.Error("an error occured in a stage", "error", e.Error())
		case <-sigc:
			return nil
		}
	}
}
