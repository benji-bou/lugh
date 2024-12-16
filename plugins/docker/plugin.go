package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"
	"sync"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/benji-bou/SecPipeline/pluginctl"
	"github.com/benji-bou/chantools"
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

func (wh Docker) Run(ctx context.Context, input <-chan *pluginctl.DataStream) (<-chan *pluginctl.DataStream, <-chan error) {
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
	return chantools.NewWithErr(func(cDataStream chan<- *pluginctl.DataStream, eC chan<- error, params ...any) {

		for i := range input {
			inputCmd := strings.Fields(string(i.Data))
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

			statusCh, errCh := cli.ContainerWait(ctx, resp.ID, container.WaitConditionNotRunning)
			select {
			case err := <-errCh:
				if err != nil {
					panic(err)
				}
			case <-statusCh:
			}
			out, err := cli.ContainerLogs(ctx, resp.ID, container.LogsOptions{ShowStdout: true})
			if err != nil {
				panic(err)
			}
			byteResC := make(chan []byte)
			writer := chantools.NewWriter(byteResC)
			dataStreamRes := chantools.Map(byteResC, func(elem []byte) *pluginctl.DataStream {
				return &pluginctl.DataStream{Data: elem, ParentSrc: "Docker"}
			})
			wg := &sync.WaitGroup{}
			wg.Add(1)
			go func() {
				defer wg.Done()
				for elem := range dataStreamRes {
					cDataStream <- elem
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
	helper.SetLog(slog.LevelInfo, true)
	plugin := pluginctl.NewPlugin("docker",
		pluginctl.WithPluginImplementation(NewDocker()),
	)
	plugin.Serve()
}
