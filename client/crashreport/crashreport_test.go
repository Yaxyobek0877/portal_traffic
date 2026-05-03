package crashreport

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"testing"
	"time"
)

func TestCapture_WritesFile(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, "0.4.0-test")

	path := c.Capture("simulated panic", debug.Stack())
	if path == "" {
		t.Fatal("Capture returned empty path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var r Report
	if err := json.Unmarshal(data, &r); err != nil {
		t.Fatalf("decode: %v\nraw: %s", err, data)
	}
	if r.PortalVersion != "0.4.0-test" {
		t.Errorf("PortalVersion = %q, want 0.4.0-test", r.PortalVersion)
	}
	if r.PanicMessage != "simulated panic" {
		t.Errorf("PanicMessage = %q", r.PanicMessage)
	}
	if r.Stack == "" {
		t.Errorf("Stack empty")
	}
	if r.GoroutineCount < 1 {
		t.Errorf("GoroutineCount = %d, expect ≥ 1", r.GoroutineCount)
	}
}

func TestCapture_NilCatcherIsSafe(t *testing.T) {
	var c *Catcher
	// Should not panic.
	if path := c.Capture("anything", debug.Stack()); path != "" {
		t.Errorf("nil catcher returned path %q, want empty", path)
	}
	// List on nil catcher should also be safe.
	if reports, err := c.List(); reports != nil || err != nil {
		t.Errorf("nil List = (%v, %v), want (nil, nil)", reports, err)
	}
}

func TestList_NewestFirst(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, "0.4.0")

	// Write two reports; second should appear first.
	c.Capture("first", debug.Stack())
	time.Sleep(1100 * time.Millisecond) // IDs use second precision
	c.Capture("second", debug.Stack())

	reports, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 2 {
		t.Fatalf("len(reports) = %d, want 2", len(reports))
	}
	if reports[0].PanicMessage != "second" {
		t.Errorf("newest = %q, want \"second\"", reports[0].PanicMessage)
	}
}

func TestList_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, "0.4.0")
	reports, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 0 {
		t.Errorf("empty dir: len = %d, want 0", len(reports))
	}
}

func TestList_IgnoresUnrelatedFiles(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, "0.4.0")
	c.Capture("real", debug.Stack())
	// Drop a non-crash file in the same dir — must be ignored.
	_ = os.WriteFile(filepath.Join(dir, "README.txt"), []byte("hello"), 0o600)
	_ = os.WriteFile(filepath.Join(dir, "crash-not-json.json"), []byte("{garbage"), 0o600)

	reports, err := c.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(reports) != 1 {
		t.Errorf("len = %d, want 1 (README + malformed file should be ignored)", len(reports))
	}
}

func TestClear(t *testing.T) {
	dir := t.TempDir()
	c := New(dir, "0.4.0")
	c.Capture("one", debug.Stack())
	c.Capture("two", debug.Stack())
	if err := c.Clear(); err != nil {
		t.Fatal(err)
	}
	reports, _ := c.List()
	if len(reports) != 0 {
		t.Errorf("after Clear: len = %d, want 0", len(reports))
	}
}

func TestSanitiseStack_RemovesHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil || home == "" {
		t.Skip("no user home directory")
	}
	stack := "goroutine 1:\n" + home + "/projects/foo.go:42 +0x12\n"
	got := sanitiseStack(stack)
	if strings.Contains(got, home) {
		t.Errorf("sanitiseStack did not redact home dir: %q", got)
	}
	if !strings.Contains(got, "$HOME") {
		t.Errorf("sanitiseStack did not insert $HOME: %q", got)
	}
}

func TestSanitisePanic_TrimsLong(t *testing.T) {
	long := strings.Repeat("a", 4096)
	got := sanitisePanic(long)
	if !strings.HasSuffix(got, "(truncated)") {
		t.Errorf("expected truncated suffix")
	}
	if len(got) > 1100 {
		t.Errorf("got len %d, want ≤ ~1100", len(got))
	}
}
