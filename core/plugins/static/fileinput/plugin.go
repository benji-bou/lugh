package fileinput

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Plugin struct {
	filepath   string `json:"filepath" yaml:"filepath"`
	lineByLine bool   `json:"lineByLine" yaml:"lineByLine"`
}

func New() *Plugin {
	return &Plugin{}
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
	return nil
}

func (tp *Plugin) Produce(ctx context.Context, yield func(elem []byte) error) error {
	if tp.filepath == "" {
		return errors.New("filepath is required")
	}

	if !tp.lineByLine {
		data, err := os.ReadFile(tp.filepath)
		if err != nil {
			return fmt.Errorf("failed to read file: %w", err)
		}
		return yield(data)
	} else {
		file, err := os.Open(tp.filepath)
		if err != nil {
			return fmt.Errorf("failed to open file: %w", err)
		}
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			err := yield(scanner.Bytes())
			if err != nil {
				return fmt.Errorf("failed to yield: %w", err)
			}
		}
		if err := scanner.Err(); err != nil {
			return fmt.Errorf("failed to scan file: %w", err)
		}
	}
	return nil
}

func (tp *Plugin) Work(ctx context.Context, input []byte, yield func(elem []byte) error) error {
	tp.filepath = string(input)
	return tp.Produce(ctx, yield)
}
