package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os/exec"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
)

type ShellOption = helper.Option[Shell]

type Shell struct {
	cmd  string
	args []string
}

func NewShell(opt ...ShellOption) *Shell {
	return helper.ConfigurePtr(&Shell{}, opt...)
}

func (mp Shell) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (mp *Shell) Config(conf []byte) error {
	config := struct {
		Cmd  string   `json:"cmd"`
		Args []string `json:"args"`
	}{}
	err := json.Unmarshal(conf, &config)
	if err != nil {
		return err
	}
	mp.cmd = config.Cmd
	mp.args = config.Args
	return nil
}

func (mp Shell) Run(context context.Context, input <-chan []byte, yield func(elem []byte) error) error {

	cmd := exec.Command(mp.cmd, mp.args...)
	inputCmd, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("shell stdinpipe failed %w", err)
	}
	defer inputCmd.Close()
	outputCmd, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("shell stdoutpipe failed %w", err)
	}
	defer outputCmd.Close()

	err = cmd.Start()
	defer cmd.Wait()
	if err != nil {
		return fmt.Errorf("shell cmd start failed %w", err)
	}
	errC := make(chan error)

	go func(outputCmd io.Reader) {
		defer close(errC)
		scanner := bufio.NewScanner(outputCmd)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) > 0 {
				slog.Debug("shell output reader", "value", line)
				yield(line)
			}
		}
		if err := scanner.Err(); err != nil {
			errC <- err
		}
	}(outputCmd)

	for {
		select {
		case <-context.Done():
			return nil
		case err := <-errC:
			return err
		case i, ok := <-input:
			if !ok {
				return nil
			}
			_, err := inputCmd.Write(i)
			if err != nil {
				return err
			}
		}
	}

	return errors.New("unsupported behavior")
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("shell",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewShell())),
	)
	plugin.Serve()
}
