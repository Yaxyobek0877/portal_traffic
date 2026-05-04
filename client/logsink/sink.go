// Package logsink tails the on-disk Portal log file and POSTs new
// lines to a server endpoint every interval. Lets us debug user
// reports ("peers don't see each other") without asking the user to
// dig through ~/.portal/logs/ and paste hundreds of lines.
//
// Wire format: gzipped plain text, one log line per newline. Server
// reads X-Client-ID (stable per-install UUID) and X-Portal-Version
// to bucket reports.
//
// The sink is best-effort. If the server is down, requests fail, or
// network drops out, we just sleep and retry next tick. We never
// retry old data — the local file is the source of truth, the
// server copy is convenience.
package logsink

import (
	"bytes"
	"compress/gzip"
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Config wires the sink. UploadURL = "" disables uploading; the
// rest of the package becomes a no-op.
type Config struct {
	// Path to the log file we're tailing. Read-only access.
	LogFile string

	// Where to POST. Empty disables. Typically derived from the
	// signaling URL by swapping wss:// → https:// and using a
	// /logs/upload path.
	UploadURL string

	// Stable per-install client identifier sent as X-Client-ID.
	// Computed once and kept under ~/.portal/client-id.
	ClientID string

	// App version, sent as X-Portal-Version.
	Version string

	// How often to drain new lines and POST. Default 30s.
	Interval time.Duration

	// HTTP client. nil = sane default with 10s timeout.
	HTTP *http.Client

	Logger *slog.Logger
}

// Sink is the running uploader.
type Sink struct {
	cfg    Config
	logger *slog.Logger

	mu       sync.Mutex
	offset   int64 // bytes already shipped from the file
	cancel   context.CancelFunc
	stopped  chan struct{}
}

// Start kicks off the background goroutine. Returns nil and a no-op
// Sink if cfg.UploadURL is empty (uploading disabled).
func Start(cfg Config) *Sink {
	if cfg.Interval <= 0 {
		cfg.Interval = 30 * time.Second
	}
	if cfg.HTTP == nil {
		cfg.HTTP = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	logger := cfg.Logger.With("component", "logsink")

	s := &Sink{
		cfg:     cfg,
		logger:  logger,
		stopped: make(chan struct{}),
	}

	if cfg.UploadURL == "" || cfg.LogFile == "" {
		logger.Info("logsink disabled", "url_set", cfg.UploadURL != "", "log_set", cfg.LogFile != "")
		close(s.stopped)
		return s
	}

	// Skip whatever's already in the file at startup so we don't
	// upload yesterday's content on every launch. Subsequent ticks
	// pick up only newly-written bytes.
	if fi, err := os.Stat(cfg.LogFile); err == nil {
		s.offset = fi.Size()
	}

	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	go s.loop(ctx)
	logger.Info("logsink started", "url", cfg.UploadURL, "client_id", cfg.ClientID, "interval", cfg.Interval)
	return s
}

// Stop signals the background goroutine to exit and waits briefly
// for it to drain.
func (s *Sink) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
	select {
	case <-s.stopped:
	case <-time.After(2 * time.Second):
	}
}

func (s *Sink) loop(ctx context.Context) {
	defer close(s.stopped)
	t := time.NewTicker(s.cfg.Interval)
	defer t.Stop()
	for {
		select {
		case <-ctx.Done():
			// One last drain on shutdown so the very tail of the
			// session doesn't get lost.
			_ = s.drainOnce(context.Background())
			return
		case <-t.C:
			if err := s.drainOnce(ctx); err != nil {
				// Best-effort: log at debug, retry next tick.
				s.logger.Debug("logsink drain failed", "err", err)
			}
		}
	}
}

// drainOnce reads everything appended to the log file since the last
// successful drain and POSTs it. If the file got rotated (size
// shrank below our offset), we reset offset to 0 and pick up from
// the beginning of the new file.
func (s *Sink) drainOnce(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.cfg.LogFile)
	if err != nil {
		return fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return fmt.Errorf("stat: %w", err)
	}
	size := fi.Size()
	if size < s.offset {
		// Rotation or truncation. Restart from 0.
		s.offset = 0
	}
	if size == s.offset {
		return nil
	}

	if _, err := f.Seek(s.offset, io.SeekStart); err != nil {
		return fmt.Errorf("seek: %w", err)
	}
	body, err := io.ReadAll(f)
	if err != nil {
		return fmt.Errorf("read: %w", err)
	}
	if len(body) == 0 {
		return nil
	}

	// Gzip in memory — log diff per interval is tiny (typically a
	// few KB), buffering is fine.
	var gz bytes.Buffer
	gw := gzip.NewWriter(&gz)
	if _, err := gw.Write(body); err != nil {
		return fmt.Errorf("gzip: %w", err)
	}
	if err := gw.Close(); err != nil {
		return fmt.Errorf("gzip close: %w", err)
	}

	if err := s.post(ctx, gz.Bytes()); err != nil {
		return err
	}
	s.offset = size
	s.logger.Debug("logsink uploaded", "bytes", len(body), "compressed", gz.Len())
	return nil
}

func (s *Sink) post(ctx context.Context, payload []byte) error {
	req, err := http.NewRequestWithContext(ctx, "POST", s.cfg.UploadURL, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain")
	req.Header.Set("Content-Encoding", "gzip")
	if s.cfg.ClientID != "" {
		req.Header.Set("X-Client-ID", s.cfg.ClientID)
	}
	if s.cfg.Version != "" {
		req.Header.Set("X-Portal-Version", s.cfg.Version)
	}

	resp, err := s.cfg.HTTP.Do(req)
	if err != nil {
		return fmt.Errorf("post: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		_, _ = io.Copy(io.Discard, resp.Body)
		return fmt.Errorf("server replied %d", resp.StatusCode)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	return nil
}

// LoadOrMintClientID returns a stable UUID-v4-like 32-hex-char id,
// generating one and writing it to ~/.portal/client-id on first call.
// We don't use a real UUID library to avoid pulling in a dep — random
// bytes are sufficient for what's effectively an opaque bucket key.
func LoadOrMintClientID() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(home, ".portal")
	_ = os.MkdirAll(dir, 0o700)
	path := filepath.Join(dir, "client-id")

	if data, err := os.ReadFile(path); err == nil {
		id := strings.TrimSpace(string(data))
		if len(id) == 32 {
			return id, nil
		}
	}

	var raw [16]byte
	if _, err := rand.Read(raw[:]); err != nil {
		return "", err
	}
	id := fmt.Sprintf("%x", raw[:])
	if err := os.WriteFile(path, []byte(id), 0o600); err != nil {
		return "", err
	}
	return id, nil
}

// DeriveUploadURL turns a wss:// signaling URL into a sensible default
// log-upload URL on the same host. Returns "" if it can't.
func DeriveUploadURL(signalingURL string) string {
	if signalingURL == "" {
		return ""
	}
	var prefix string
	var rest string
	switch {
	case strings.HasPrefix(signalingURL, "wss://"):
		prefix = "https://"
		rest = signalingURL[len("wss://"):]
	case strings.HasPrefix(signalingURL, "ws://"):
		prefix = "http://"
		rest = signalingURL[len("ws://"):]
	default:
		return ""
	}
	// Trim any path/query the signaling URL had (e.g. /ws) so we
	// land on the bare host before appending /logs/upload.
	if i := strings.IndexAny(rest, "/?"); i >= 0 {
		rest = rest[:i]
	}
	if rest == "" {
		return ""
	}
	return prefix + rest + "/logs/upload"
}

// ErrDisabled is returned by helper paths if uploading was opted-out.
var ErrDisabled = errors.New("log upload disabled")
