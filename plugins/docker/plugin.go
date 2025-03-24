package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strings"

	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

const (
	BuffSize int = 1024
)

type ConfigDocker struct {
	Image string `json:"image"`
	Host  string `json:"host"`
}

type Docker struct {
	config ConfigDocker
}

type DockerOption = helper.Option[Docker]

func NewDocker(opt ...DockerOption) *Docker {
	return helper.ConfigurePtr(&Docker{}, opt...)
}

func (wh *Docker) Config(config []byte) error {
	configDocker := ConfigDocker{}
	err := json.Unmarshal(config, &configDocker)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for Docker config: %w", err)
	}
	wh.config = configDocker
	return nil
}

func (Docker) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (wh Docker) Work(ctx context.Context, input []byte, yield func([]byte) error) error {
	hostOpt := client.WithHostFromEnv()
	if wh.config.Host != "" {
		hostOpt = client.WithHost(wh.config.Host)
	}
	cli, err := client.NewClientWithOpts(hostOpt, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker plugin get docker client %w", err)
	}
	defer cli.Close()

	inputCmd := strings.Fields(string(input))
	slog.Info("receive cmd", "cmd", inputCmd)
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: wh.config.Image,
		Cmd:   inputCmd,
		Tty:   true,
	}, nil, nil, nil, "")
	if err != nil {
		return fmt.Errorf("docker plugin create container failed %w", err)
	}

	if err = cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("docker plugin start container failed %w", err)
	}
	slog.Debug("start  container wait")

	statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err = <-errCh:
		if err != nil {
			return fmt.Errorf("docker plugin wait container failed %w", err)
		}
	case resp := <-statusCh:
		if resp.Error != nil {
			return fmt.Errorf("docker plugin wait container failed %s", resp.Error.Message)
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	slog.Debug("start get container logs")
	out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
	if err != nil {
		return fmt.Errorf("docker plugin get container logs failed %w", err)
	}

	buff := make([]byte, BuffSize)
	for {
		n, err := out.Read(buff)
		if n > 0 {
			err = yield(slices.Clone(buff[:n]))
			if err != nil {
				slog.Error("failed to yield output logs stream from container", "error", err)
			}
		}
		if errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("failed to read output logs stream from container: %w", err)
		}
	}
}

func main() {
	helper.SetLog(slog.LevelDebug, false)
	plugin := grpc.NewPlugin("docker",
		grpc.WithPluginImplementation(pluginapi.NewConfigurableWorker(NewDocker())),
	)
	plugin.Serve()
}
