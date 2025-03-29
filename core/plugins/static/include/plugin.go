package include

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/template"
)

var ErrEmptyIncludePath = errors.New("include path is empty")

type Config struct {
	Filepath   string         `yaml:"filepath"`
	Variables  map[string]any `yaml:"variables"`
	PluginPath string         `yaml:"pluginpath"`
}

func Worker[S template.PluginLoader](config Config) (graph.IOWorker[[]byte], error) {
	if config.Filepath == "" {
		return nil, ErrEmptyIncludePath
	}
	config.Variables["is_included"] = true
	tpl, err := template.NewFile[S](config.Filepath, template.WithPluginPath(config.PluginPath), template.WithVariables(config.Variables))
	if err != nil {
		slog.Error("include template failed", "error", err)
		return nil, fmt.Errorf("include template %s failed: %w", config.Filepath, err)
	}
	vertices, err := tpl.WorkerVertexIterator()
	if err != nil {
		return nil, fmt.Errorf("include template %s get vertices: %w", config.Filepath, err)
	}
	g := graph.NewIO(graph.WithVertices(vertices))
	return g, nil
}
