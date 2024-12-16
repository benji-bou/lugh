package ctrl

import (
	"fmt"
	"io"
	"log/slog"

	"github.com/benji-bou/SecPipeline/core/api"
	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/template"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/labstack/echo/v4"
)

func NewPipeline() api.Ctrler {
	return Pipeline{}
}

type Pipeline struct {
}

func (p Pipeline) Route() []helper.SrvOption {
	return []helper.SrvOption{
		helper.WithPost("/run", p.run),
	}
}

func (p Pipeline) ExtractGraphFromBody(c echo.Context) (*graph.SecGraph, error) {
	body := c.Request().Body
	defer body.Close()
	content, err := io.ReadAll(body)
	if err != nil {
		return nil, err
	}
	fmt.Printf("%s", string(content))
	tpl, err := template.NewRawTemplate(content)
	if err != nil {
		return nil, err
	}

	g := graph.NewGraph(graph.WithSecVertexerIterator(tpl.SecVertexIterator()))
	return g, nil

}

func (p Pipeline) run(c echo.Context) error {
	g, err := p.ExtractGraphFromBody(c)
	if err != nil {
		c.JSONBlob(500, []byte(fmt.Sprintf(`{"error": %s}`, err.Error())))
		return err
	}

	resC := make(chan *pluginctl.DataStream)
	err = g.ChanOutputGraph(resC)
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
}
