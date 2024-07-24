package main

import (
	"context"
	"io"
	"log/slog"

	"github.com/benji-bou/chantools"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
)

func main() {
	cli, err := client.NewClientWithOpts(client.WithHost("unix:///Users/benjamin/.orbstack/run/docker.sock"), client.WithAPIVersionNegotiation())
	if err != nil {
		panic(err)
	}
	defer cli.Close()
	ctx := context.Background()
	resp, err := cli.ContainerCreate(ctx, &container.Config{
		Image: "holehe:latest",
		Cmd: []string{
			"holehe",
			"bouachour.benjamin@gmail.com",
		},
		Tty: true,
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
	defer writer.Close()
	defer out.Close()
	io.Copy(writer, out)
	for data := range byteResC {
		slog.Info("data", "data", data)
	}
}
