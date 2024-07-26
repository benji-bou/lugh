package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"log"
	"log/slog"

	"os"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugin"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"golang.org/x/exp/maps"

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
		g, err := graph.NewGraph(graph.WithRawTemplate(content))
		if err != nil {
			return err
		}
		childLessVertex, err := g.GetChildlessVertex()
		if err != nil {
			return err
		}
		resC := make(chan *pluginctl.DataStream)
		rawOutputPlugin := plugin.NewRawOutputPlugin(plugin.WithChannel(resC))

		outputVertex, err := graph.NewSecVertex(fmt.Sprintf("webapi-output-%s", uuid.NewString()), graph.VertexWithPlugin(rawOutputPlugin))
		if err != nil {
			return err
		}
		err = g.AddSecVertex(outputVertex, maps.Values(childLessVertex)...)
		if err != nil {
			return err
		}
		err, errC := g.Start(context.Background())
		if err != nil {
			slog.Error("failed to start template", "error", err)
			return err
		}
		buffRes := &bytes.Buffer{}
		for {
			select {
			case data, ok := <-resC:
				if !ok {
					resRaw := buffRes.Bytes()
					slog.Error("res", "data", resRaw)
					err := c.Blob(200, "text/plain", resRaw)
					if err != nil {
						return err
					}
				} else {
					buffRes.Write(data.Data)
				}

			case e, ok := <-errC:
				if !ok {
					slog.Info("end of workflow")
					return nil
				}
				slog.Error("an error occured in a stage", "error", e)
			}
		}
	}))

}
