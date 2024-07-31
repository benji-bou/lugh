package main

import (
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
	"github.com/labstack/echo/v4/middleware"
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
	return helper.RunServer(helper.WithMiddleware(middleware.CORS()), helper.WithPost("/run", func(c echo.Context) error {
		body := c.Request().Body
		defer body.Close()
		content, err := io.ReadAll(body)
		if err != nil {
			return err
		}
		fmt.Printf("%s", string(content))
		g, err := graph.NewGraph(graph.WithRawTemplate(content))
		if err != nil {
			c.JSONBlob(500, []byte(fmt.Sprintf(`{"error": %s}`, err.Error())))
			return err
		}
		childLessVertex, err := g.GetChildlessVertex()
		if err != nil {
			c.JSONBlob(500, []byte(fmt.Sprintf(`{"error": %s}`, err.Error())))
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
			c.JSONBlob(500, []byte(fmt.Sprintf(`{"error": %s}`, err.Error())))
			return err
		}
		err, errC := g.Start(c.Request().Context())
		if err != nil {
			slog.Error("failed to start template", "error", err)
			c.JSONBlob(500, []byte(fmt.Sprintf(`{"error": %s}`, err.Error())))
			return err
		}
		buffRes := make([]string, 0)
		for {
			select {
			case data, ok := <-resC:
				if !ok {
					slog.Error("res", "data", buffRes)
					err := c.JSON(200, struct {
						Data []string `json:"data"`
					}{Data: buffRes})
					if err != nil {
						return err
					}
					return nil
				} else if len(data.Data) > 0 {
					buffRes = append(buffRes, string(data.Data))
				}

			case e, ok := <-errC:
				if !ok {
					slog.Info("end of workflow")
					c.JSON(500, "toto")
					return nil
				}
				slog.Error("an error occured in a stage", "error", e)
				c.JSONBlob(500, []byte(fmt.Sprintf(`{"error": %s}`, e.Error())))
			}
		}
	}))

}
