package grpc

import (
	"errors"
	"fmt"
	"io"
	"log/slog"

	grpc "google.golang.org/grpc"
	codes "google.golang.org/grpc/codes"
	status "google.golang.org/grpc/status"
)

func handleGRPCStreamError(err error, name string) error {
	if err != nil {
		if errors.Is(err, io.EOF) {
			slog.Debug("end of stream", "name", name)
			return nil
		} else if e, ok := status.FromError(err); ok && e.Code() == codes.Canceled {
			slog.Debug("stream canceled", "error", err, "name", name)
			return nil
		}
		slog.Error("stream error", "error", err, "name", name)
		return err
	}
	return nil
}

func closeStream(stream grpc.ClientStream, name string) error {
	err := stream.CloseSend()
	if err != nil {
		slog.Error("failed to close client stream", "error", err, "name", name)
		return fmt.Errorf("failed to close client stream named %s: %w", name, err)
	}
	return nil
}
