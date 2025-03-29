package main

import (
	"bufio"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"runtime"
	"strings"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/template"
	"github.com/benji-bou/lugh/helper"

	"github.com/urfave/cli/v2"
)

func main() {
	runtime.SetBlockProfileRate(1)
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
			&cli.StringFlag{
				Name:    "plugins-path",
				Aliases: []string{"p"},
				Usage:   "directory path of the plugins",
				Value:   "~/.lugh/plugins",
			},
			&cli.StringFlag{
				Name:    "raw-input",
				Aliases: []string{"i"},
				Usage:   "raw input to pass to the pipeline",
			},
			&cli.StringSliceFlag{
				Name:    "var",
				Aliases: []string{"v"},
				Usage:   "variable to pass to the pipeline. in the form of Key=Value",
			},
		},
		Action: func(c *cli.Context) error {
			defer grpc.CleanupClients()
			plugins.InitLoader()
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
	tpl, err := template.NewFile[template.Stage](tplPath, template.WithPluginPath(c.String("plugins-path")))
	if err != nil {
		return err
	}
	vertices, err := tpl.WorkerVertexIterator()
	if err != nil {
		return err
	}
	g := graph.NewIO(graph.WithVertices(vertices))
	return g.DrawGraph(c.String("draw-graph-only"))
}

func RunTemplate(c *cli.Context) error {
	tplPath := c.String("template")
	variables := make(map[string]interface{})
	for _, v := range c.StringSlice("var") {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) != 2 {
			return fmt.Errorf("invalid variable format: %s", v)
		}
		variables[parts[0]] = parts[1]
	}
	tpl, err := template.NewFile[template.Stage](tplPath,
		template.WithPluginPath(c.String("plugins-path")),
		template.WithVariables(variables),
	)
	if err != nil {
		slog.Error("failed to start template", "error", err)
		return err
	}
	vertices, err := tpl.WorkerVertexIterator()
	if err != nil {
		return err
	}
	g := graph.NewIO(graph.WithVertices(vertices))
	inputC := make(chan []byte)
	g.SetInput(inputC)
	ctx := graph.NewContext(c.Context)
	errC := g.Run(ctx)
	ctx.Synchronize()
	go SendRawInput(c, inputC)
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

func SendRawInput(c *cli.Context, inputC chan []byte) {
	defer close(inputC)
	if c.IsSet("raw-input") {
		slog.Debug("sending raw input")
		inputC <- []byte(c.String("raw-input"))
		slog.Debug("raw input sent")
	}
	fi, err := os.Stdin.Stat()
	if err != nil {
		slog.Warn("couldn't stat stdin", "error", err)
		return
	}
	if fi.Mode()&os.ModeCharDevice == 0 || fi.Mode()&os.ModeNamedPipe == 0 {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	slog.Debug("reading from stdin")
	// Loop until stdin is closed
	for scanner.Scan() {
		// Read each line from stdin
		line := scanner.Text()
		if line != "" {
			inputC <- []byte(line)
		}
	}
	// Check for any errors encountered during scanning
	if err := scanner.Err(); err != nil {
		slog.Warn("Error reading from stdin", "error", err)
	}
}
