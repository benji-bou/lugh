package main

import (
	"bufio"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/template"
	"github.com/benji-bou/lugh/helper"

	"github.com/urfave/cli/v2"

	_ "net/http/pprof"
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
	go func() {
		http.ListenAndServe(":8081", nil)
	}()
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
	g := graph.NewIO(graph.WithVertices(tpl.WorkerVertexIterator(c.String("plugins-path"))))
	return g.DrawGraph(c.String("draw-graph-only"))
}

func RunTemplate(c *cli.Context) error {
	tplPath := c.String("template")
	tpl, err := template.NewFile(tplPath)
	if err != nil {
		slog.Error("failed to start template", "error", err)
		return err
	}
	g := graph.NewIO(graph.WithVertices(tpl.WorkerVertexIterator(c.String("plugins-path"))))
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
