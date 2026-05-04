package logsink

import (
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestSink_DrainsAppendedLines(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "portal.log")
	if err := os.WriteFile(logFile, []byte("opening line\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	var got atomic.Pointer[string]
	var clientID atomic.Pointer[string]
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("gzip reader: %v", err)
			return
		}
		raw, _ := io.ReadAll(gr)
		s := string(raw)
		got.Store(&s)
		c := r.Header.Get("X-Client-ID")
		clientID.Store(&c)
		w.WriteHeader(204)
	}))
	defer srv.Close()

	s := Start(Config{
		LogFile:   logFile,
		UploadURL: srv.URL + "/logs/upload",
		ClientID:  "test-client",
		Version:   "test",
		Interval:  10 * time.Millisecond,
	})
	defer s.Stop()

	// Append something AFTER startup so the sink picks it up.
	time.Sleep(30 * time.Millisecond)
	f, _ := os.OpenFile(logFile, os.O_APPEND|os.O_WRONLY, 0o600)
	_, _ = f.WriteString("hello after startup\n")
	_, _ = f.WriteString("second line\n")
	_ = f.Close()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if g := got.Load(); g != nil && strings.Contains(*g, "hello after startup") {
			break
		}
		time.Sleep(20 * time.Millisecond)
	}
	g := got.Load()
	if g == nil || !strings.Contains(*g, "hello after startup") {
		t.Fatalf("uploaded body never contained the appended line; got=%v", g)
	}
	if !strings.Contains(*g, "second line") {
		t.Errorf("expected both appended lines in one upload, got %q", *g)
	}
	if strings.Contains(*g, "opening line") {
		t.Errorf("opening line (pre-startup) should NOT have been uploaded: %q", *g)
	}
	if c := clientID.Load(); c == nil || *c != "test-client" {
		t.Errorf("X-Client-ID = %v, want test-client", c)
	}
}

func TestSink_DisabledNoURL(t *testing.T) {
	dir := t.TempDir()
	logFile := filepath.Join(dir, "portal.log")
	_ = os.WriteFile(logFile, []byte("x\n"), 0o600)
	s := Start(Config{
		LogFile:   logFile,
		UploadURL: "",
	})
	// Shouldn't panic; Stop should return immediately.
	s.Stop()
}

func TestDeriveUploadURL(t *testing.T) {
	cases := []struct {
		in, want string
	}{
		{"wss://signaling.1pro.uz/ws", "https://signaling.1pro.uz/logs/upload"},
		{"ws://localhost:18080/ws", "http://localhost:18080/logs/upload"},
		{"wss://example.com", "https://example.com/logs/upload"},
		{"wss://example.com/", "https://example.com/logs/upload"},
		{"https://wrong-scheme/ws", ""},
		{"", ""},
	}
	for _, c := range cases {
		got := DeriveUploadURL(c.in)
		if got != c.want {
			t.Errorf("DeriveUploadURL(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestLoadOrMintClientID_StableAcrossCalls(t *testing.T) {
	// LoadOrMintClientID writes to ~/.portal/client-id; we
	// override HOME to keep the test hermetic.
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	t.Setenv("USERPROFILE", tmp) // Windows
	id1, err := LoadOrMintClientID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id1) != 32 {
		t.Errorf("expected 32-hex id, got len=%d (%q)", len(id1), id1)
	}
	id2, err := LoadOrMintClientID()
	if err != nil {
		t.Fatal(err)
	}
	if id1 != id2 {
		t.Errorf("client ID not stable: %q vs %q", id1, id2)
	}
}
