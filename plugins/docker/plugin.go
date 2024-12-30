package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/benji-bou/diwo"
	"github.com/benji-bou/lugh/core/graph"
	"github.com/benji-bou/lugh/core/plugins/grpc"
	"github.com/benji-bou/lugh/core/plugins/pluginapi"
	"github.com/benji-bou/lugh/helper"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

type ConfigDocker struct {
	Image string `json:"image"`
	Host  string `json:"host"`
}

type Docker struct {
	config ConfigDocker
}

func (mp *Docker) Config(config []byte) error {
	configDocker := ConfigDocker{}
	err := json.Unmarshal(config, &configDocker)
	if err != nil {
		return fmt.Errorf("failed to unmarshal json for Docker config: %w", err)
	}
	mp.config = configDocker
	return nil
}

type DockerOption = helper.Option[Docker]

func NewDocker(opt ...DockerOption) *Docker {
	return helper.ConfigurePtr(&Docker{}, opt...)
}
func (wh Docker) GetInputSchema() ([]byte, error) {
	return nil, nil
}

func (wh Docker) Run(ctx graph.Context, input <-chan []byte) <-chan graph.RunItem[[]byte] {
	hostOpt := client.WithHostFromEnv()
	if wh.config.Host != "" {
		hostOpt = client.WithHost(wh.config.Host)
	}
	cli, err := client.NewClientWithOpts(hostOpt, client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()

	// reader, err := cli.ImagePull(ctx, wh.config.Image, image.PullOptions{})
	// if err != nil {
	// 	panic(err)
	// }

	// defer reader.Close()
	// cli.ImagePull is asynchronous.
	// The reader needs to be read completely for the pull operation to complete.
	// If stdout is not required, consider using io.Discard instead of os.Stdout.
	// io.Copy(os.Stdout, reader)
	return diwo.New(func(cDataStream chan<- graph.RunItem[[]byte]) {

		for i := range input {
			inputCmd := strings.Fields(string(i))
			slog.Info("receive cmd", "cmd", inputCmd)
			resp, err := cli.ContainerCreate(ctx, &container.Config{
				Image: wh.config.Image,
				Cmd:   inputCmd,
				Tty:   true,
			}, nil, nil, nil, "")
			if err != nil {
				panic(err)
			}

			if err := cli.ContainerStart(ctx, resp.ID, container.StartOptions{}); err != nil {
				panic(err)
			}
			slog.Debug("start  container wait")

			statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				if err != nil {
					panic(err)
				}
			case <-statusCh:
			}
			slog.Debug("start get container logs")
			out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
			if err != nil {
				panic(err)
			}
			byteResC := make(chan []byte)
			writer := diwo.NewWriter(byteResC)

			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for elem := range byteResC {
					slog.Debug("send to output logs")
					cDataStream <- graph.RunItem[[]byte]{Item: elem, Err: nil}
				}
				slog.Debug("end of forward data response ")
			}()
			slog.Debug("copy")
			io.Copy(writer, out)
			slog.Debug("waiting end of forward")
			writer.Close()
			wg.Wait()
			slog.Debug("end of wait will close")
			out.Close()
		}

	})
}

func main() {
	helper.SetLog(slog.LevelDebug, false)
	plugin := grpc.NewPlugin("docker",
		grpc.WithPluginImplementation(pluginapi.NewIOWorkerPluginFromRunner(NewDocker())),
	)
	plugin.Serve()
}
