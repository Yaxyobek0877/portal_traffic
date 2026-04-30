// File-based logging support. By default the desktop binary logs to
// stderr (which the user never sees, since `.app` bundles don't have
// a terminal). We tee everything into ~/.portal/logs/portal.log so
// the user can pull recent logs out of Settings → Diagnostika when
// something goes wrong.
package main

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// LogFile is the absolute path to the on-disk log file.
var LogFile string

// logTail keeps the last N lines in memory so the UI can display them
// without re-reading the file each time. We also still write to the
// file for offline forensics.
type logTail struct {
	mu    sync.Mutex
	lines []string
	max   int
}

var tail = &logTail{max: 500}

func (t *logTail) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	for _, line := range splitLines(string(p)) {
		if line == "" {
			continue
		}
		t.lines = append(t.lines, line)
		if len(t.lines) > t.max {
			t.lines = t.lines[len(t.lines)-t.max:]
		}
	}
	return len(p), nil
}

func (t *logTail) snapshot() []string {
	t.mu.Lock()
	defer t.mu.Unlock()
	out := make([]string, len(t.lines))
	copy(out, t.lines)
	return out
}

func splitLines(s string) []string {
	var out []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			out = append(out, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		out = append(out, s[start:])
	}
	return out
}

// initLogger wires up a slog.Logger that writes to both stderr and
// the rotating log file under ~/.portal/logs/. Returns the configured
// logger plus an opener for the on-disk path so main can keep the
// reference.
func initLogger() *slog.Logger {
	home, err := os.UserHomeDir()
	if err != nil {
		home = "."
	}
	dir := filepath.Join(home, ".portal", "logs")
	_ = os.MkdirAll(dir, 0o700)

	// One file per day so logs don't grow without bound. Old files
	// are kept (no auto-prune in v1) — easy to inspect manually.
	stamp := time.Now().Format("2006-01-02")
	LogFile = filepath.Join(dir, fmt.Sprintf("portal-%s.log", stamp))

	f, err := os.OpenFile(LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		// Worst case: fall back to stderr-only.
		return slog.New(slog.NewTextHandler(io.MultiWriter(os.Stderr, tail), &slog.HandlerOptions{
			Level: slog.LevelInfo,
		}))
	}

	w := io.MultiWriter(os.Stderr, f, tail)
	return slog.New(slog.NewTextHandler(w, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
}

// LogTail returns the last N lines kept in the in-memory ring buffer.
// max is clamped to whatever buffer size we configured.
func LogTail(n int) []string {
	all := tail.snapshot()
	if n <= 0 || n > len(all) {
		return all
	}
	return all[len(all)-n:]
}

// LogPath returns the absolute path to the current log file.
func LogPath() string { return LogFile }

// ClearLogTail empties the in-memory ring buffer (file remains).
func ClearLogTail() {
	tail.mu.Lock()
	tail.lines = nil
	tail.mu.Unlock()
}
