package modifiers

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/benji-bou/SecPipeline/helper"
	"github.com/google/martian/v3"
	"github.com/google/martian/v3/har"
	"github.com/google/martian/v3/parse"
)

var DefaultWriter io.Writer

func init() {
	parse.Register("output", func(b []byte) (*parse.Result, error) {
		return parse.NewResult(NewLogger(nil), []parse.ModifierType{"request", "response"})
	})
}

type WritterLogger struct {
	mu      sync.Mutex
	entries map[string]*har.Entry
	w       io.Writer
	logBody bool
}

func NewLogger(w io.Writer, opt ...helper.Option[WritterLogger]) *WritterLogger {
	if w == nil {
		if DefaultWriter != nil {
			w = DefaultWriter
		} else {
			w = os.Stdout
		}
	}
	return helper.ConfigurePtr(&WritterLogger{w: w, entries: make(map[string]*har.Entry), logBody: true}, opt...)
}

// ModifyRequest logs requests.
func (l *WritterLogger) ModifyRequest(req *http.Request) error {
	slog.Info("receive request", "url", req.URL, "modifier", "WritterLogger", "plugin", "MartianProxy")
	ctx := martian.NewContext(req)
	id := ctx.ID()
	hreq, err := har.NewRequest(req, l.logBody)
	if err != nil {
		errF := fmt.Errorf("failed to convert request to har format: %w", err)
		slog.Error("receive request modification failed", "url", req.URL, "modifier", "WritterLogger", "plugin", "MartianProxy", "error", errF)
		return errF
	}
	entry := &har.Entry{
		ID:              id,
		StartedDateTime: time.Now().UTC(),
		Request:         hreq,
		Cache:           &har.Cache{},
		Timings:         &har.Timings{},
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	if _, exists := l.entries[id]; exists {
		return fmt.Errorf("duplicate request ID: %s", id)
	}

	l.entries[id] = entry

	return l.writeEntries(entry)
}

func (l *WritterLogger) ModifyResponse(res *http.Response) error {
	slog.Info("receive response", "url", res.Request.URL, "modifier", "WritterLogger", "plugin", "MartianProxy")
	id := martian.NewContext(res.Request).ID()
	hres, err := har.NewResponse(res, l.logBody)
	if err != nil {
		return err
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	entry, isOK := l.entries[id]
	if !isOK {
		return fmt.Errorf("no corresponding entry for response ID: %v", id)
	}
	delete(l.entries, id)
	entry.Response = hres
	entry.Time = time.Since(entry.StartedDateTime).Nanoseconds() / 1000000
	return l.writeEntries(entry)
}

func (l *WritterLogger) writeEntries(entry *har.Entry) error {
	rawHar, err := json.Marshal(entry)
	if err != nil {
		return err
	}
	// slog.Debug("write new entry", "entry", entry, "object", "WritterLogger", "plugin", "MartianProxy")
	_, err = l.w.Write(rawHar)
	if err != nil {
		slog.Error("write entry", "error", err, "object", "WritterLogger", "plugin", "MartianProxy")
		return err
	}
	return nil
}
