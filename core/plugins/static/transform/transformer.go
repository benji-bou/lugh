package transform

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"gopkg.in/yaml.v3"
)

type Transformer interface {
	// Transform takes a value and returns a transformed version of it.
	Transform(input []byte) ([][]byte, error)
}

type Func func(input []byte) ([][]byte, error)

func (tf Func) Transform(input []byte) ([][]byte, error) {
	return tf(input)
}

type Map func(input []byte) ([]byte, error)

func (tm Map) Transform(input []byte) ([][]byte, error) {
	res, err := tm(input)
	return [][]byte{res}, err
}

type Plugin struct {
	Transformers []Transformer
}

func New() *Plugin {
	return &Plugin{Transformers: make([]Transformer, 0)}
}

func (*Plugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (tp *Plugin) Config(config []byte) error {
	decodedConfig := []map[string]yaml.Node{}
	err := yaml.Unmarshal(config, &decodedConfig)
	if err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}
	if tp.Transformers == nil {
		tp.Transformers = make([]Transformer, 0, len(decodedConfig))
	} else {
		tp.Transformers = tp.Transformers[:0]
	}
	for _, transformConfigs := range decodedConfig {
		if len(transformConfigs) > 1 {
			return errors.New("a transformer can only have one config per step")
		}
		//nolint:gocritic // just test
		for transformName, transformConfig := range transformConfigs {
			t, err := NewTransform(transformName, &transformConfig)
			if err != nil {
				return fmt.Errorf("create transform: %s: %w", transformName, err)
			}
			tp.Transformers = append(tp.Transformers, t)
		}
	}
	return nil
}

func (tp *Plugin) Work(_ context.Context, input []byte, yield func(elem []byte) error) error {
	deepCopy := func(input [][]byte) [][]byte {
		copyInput := make([][]byte, len(input))
		for i, in := range input {
			copyInput[i] = make([]byte, len(in))
			copy(copyInput[i], in)
		}
		return copyInput
	}

	nextInput := [][]byte{input}

	for _, t := range tp.Transformers {
		currentInput := deepCopy(nextInput)
		nextInput = nextInput[:0]
		for _, i := range currentInput {
			tmpNextInput, err := t.Transform(i)
			if err != nil {
				return err
			}
			nextInput = append(nextInput, tmpNextInput...)
		}
	}
	for _, ni := range nextInput {
		err := yield(ni)
		if err != nil {
			slog.Error("Error yielding", "err", err)
			return err
		}
	}
	return nil
}
