package stdoutput

import (
	"context"
	"fmt"
)

type Plugin struct{}

func (Plugin) Config(_ []byte) error {
	return nil
}

func (Plugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (Plugin) Consume(ctx context.Context, input <-chan []byte) error {
	for {
		select {
		case i := <-input:
			_, err := fmt.Printf("%s\n", string(i))
			if err != nil {
				return err
			}
		case <-ctx.Done():
			return ctx.Err()
		}
	}
}
