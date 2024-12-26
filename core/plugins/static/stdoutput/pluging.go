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
func (p Plugin) Work(ctx context.Context, input []byte) ([][]byte, error) {
	_, err := fmt.Printf("%s\n", string(input))
	return nil, err
}
