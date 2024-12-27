package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"os/exec"

	"github.com/benji-bou/SecPipeline/core/graph"
	"github.com/benji-bou/SecPipeline/core/plugins/grpc"
	"github.com/benji-bou/SecPipeline/core/plugins/pluginapi"
	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/chantools"
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

func (mp Shell) startCmdAndPipeInput(context context.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	return chantools.NewWithErr(func(c chan<- []byte, eC chan<- error, params ...any) {

		cmd := exec.Command(mp.cmd, mp.args...)
		inputCmd, err := cmd.StdinPipe()
		if err != nil {
			eC <- err
			return
		}
		defer inputCmd.Close()
		outputCmd, err := cmd.StdoutPipe()
		if err != nil {
			eC <- err
			return
		}
		defer outputCmd.Close()

		err = cmd.Start()
		defer cmd.Wait()
		if err != nil {
			eC <- err
			return
		}

		go func(outputCmd io.Reader) {
			reader := bufio.NewReader(outputCmd)
			for {
				str, err := reader.ReadString('\n')
				if len(str) > 0 {
					slog.Debug("shell output reader", "value", str)
					c <- []byte(str)
				}
				if err != nil {
					slog.Error("quit shell output reader", "error", err)
					return
				}
			}
		}(outputCmd)

		for {
			select {
			case <-context.Done():
				return
			case i, ok := <-input:
				if !ok {
					return
				}
				inputCmd.Write(i)
			}
		}
	})
}

func (mp Shell) Run(context graph.Context, input <-chan []byte) (<-chan []byte, <-chan error) {
	if mp.cmd != "" {
		return mp.startCmdAndPipeInput(context, input)
	}
	return make(<-chan []byte), chantools.Once(errors.New("unsupported behavior"))
}

func main() {
	helper.SetLog(slog.LevelError, true)
	plugin := grpc.NewPlugin("shell",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewShell())),
	)
	plugin.Serve()
}
