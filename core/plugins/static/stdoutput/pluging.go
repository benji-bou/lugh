package stdoutput

import (
	"context"
	"fmt"
)

type Plugin struct {
}

func (p Plugin) Config(config []byte) error {
	return nil
}
func (p Plugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (p Plugin) Consume(ctx context.Context, input <-chan []byte) error {
	for i := range input {
		_, err := fmt.Printf("%s\n", string(i))
		if err == nil {
			return err
		}
	}
	return nil
}
