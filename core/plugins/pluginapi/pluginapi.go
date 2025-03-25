package pluginapi

import (
	"github.com/benji-bou/lugh/core/graph"
)

type PluginConfigurer interface {
	GetInputSchema() ([]byte, error)
	Config(config []byte) error
}

type Worker = graph.Worker[[]byte]

type Runner = graph.Runner[[]byte]

type Producer = graph.Producer[[]byte]

type Consumer = graph.Consumer[[]byte]

type IOWorker = graph.IOWorker[[]byte]
