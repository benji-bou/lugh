package main

import (
	"context"
	"log"
	"log/slog"
	"os"
	"os/signal"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/template"
	"github.com/benji-bou/lugh/helper"

	"github.com/urfave/cli/v2"
)

func main() {
	app := &cli.App{
		Name:  "lugh",
		Usage: "lugh can be use to construct cyber security pipeline based on modules",
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
			defer grpc.CleanupClients()

			helper.SetLog(slog.LevelDebug, false)
			if c.IsSet("draw-graph-only") {
				return DrawGraphOnly(c)
			}
			return RunTemplate(c)
		},
	}
	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func DrawGraphOnly(c *cli.Context) error {
	tplPath := c.String("template")
	tpl, err := template.NewFile(tplPath)
	if err != nil {
		return err
	}

	g := graph.NewIO(graph.WithVertices(tpl.WorkerVertexIterator()))

	return g.DrawGraph(c.String("draw-graph-only"))
}

func RunTemplate(c *cli.Context) error {
	tplPath := c.String("template")
	tpl, err := template.NewFile(tplPath)
	if err != nil {
		slog.Error("failed to start template", "error", err)
		return err
	}
	g := graph.NewIO(graph.WithVertices(tpl.WorkerVertexIterator()))
	errC := g.Run(graph.NewContext(context.Background()))
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt)
	for {
		select {
		case e, ok := <-errC:
			if !ok {
				return nil
			}
			slog.Error("an error occurred in a stage", "error", e.Error())
		case <-sigc:
			return nil
		}
	}
}
