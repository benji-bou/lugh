package helper

import (
	"log/slog"
	"os"
)

func SetLog(level slog.Level, showSource bool) {
	opts := &slog.HandlerOptions{
		Level:     level,
		AddSource: showSource,
	}
	handler := slog.NewTextHandler(os.Stderr, opts)
	slog.SetDefault(slog.New(handler))
}
