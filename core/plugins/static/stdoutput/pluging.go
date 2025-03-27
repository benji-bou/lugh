package stdoutput

import (
	"context"
	"fmt"
)

type Plugin struct{}

func New() Plugin {
	return Plugin{}
}

func (Plugin) Config(_ []byte) error {
	return nil
}

func (Plugin) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (Plugin) Consume(_ context.Context, input []byte) error {
	_, err := fmt.Printf("%s\n", string(input))
	return err
}
