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

func (*Shell) GetInputSchema() ([]byte, error) {
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

func (*Shell) handleCmdOutput(outputCmd io.ReadCloser, yield func(elem []byte, err error) error) (err error) {
	defer func(outputCmd io.ReadCloser) {
		err = outputCmd.Close()
		if err != nil {
			err = fmt.Errorf("close stdoutput failed %w", err)
		}
	}(outputCmd)
	scanner := bufio.NewScanner(outputCmd)
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) > 0 {
			slog.Debug("shell output reader", "value", line)
			err := yield(line, nil)
			if err != nil {
				slog.Error("yield output failed", "error", err)
				return err
			}
		}
	}
	if err := scanner.Err(); err != nil {
		if !errors.Is(err, io.EOF) {
			return fmt.Errorf("scanner failed %w", err)
		}
	}
	return nil
}

func (*Shell) handleCmdInput(ctx context.Context, inputCmd io.WriteCloser, inputC <-chan []byte) (err error) {
	defer func(inputCmd io.WriteCloser) {
		err = inputCmd.Close()
		if err != nil {
			err = fmt.Errorf("shell stdinpipe close failed %w", err)
		}
	}(inputCmd)
	for {
		select {
		case <-ctx.Done():
			return nil
		case i, ok := <-inputC:
			if !ok {
				return nil
			}
			_, err := inputCmd.Write(i)
			if err != nil {
				return err
			}
		}
	}
}

func (mp *Shell) Run(ctx context.Context, input <-chan []byte, yield func(elem []byte, err error) error) (err error) {
	cmd := exec.Command(mp.cmd, mp.args...) // #nosec G204
	inputCmd, err := cmd.StdinPipe()
	if err != nil {
		return fmt.Errorf("shell stdinpipe failed %w", err)
	}

	outputCmd, err := cmd.StdoutPipe()
	if err != nil {
		return fmt.Errorf("shell stdoutpipe failed %w", err)
	}
	err = cmd.Start()
	if err != nil {
		return fmt.Errorf("shell cmd start failed %w", err)
	}
	defer func(cmd *exec.Cmd) {
		err = cmd.Wait()
		if err != nil {
			err = fmt.Errorf("wait for cmd failed %w", err)
		}
	}(cmd)
	go func(outputCmd io.ReadCloser, yield func(elem []byte, err error) error) {
		err := mp.handleCmdOutput(outputCmd, yield)
		if err != nil {
			slog.Error("Run output error", "error", err)
		}
	}(outputCmd, yield)

	return mp.handleCmdInput(ctx, inputCmd, input)
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("shell",
		grpc.WithPluginImplementation(pluginapi.NewConfigurableRunner(NewShell())),
	)
	plugin.Serve()
}
