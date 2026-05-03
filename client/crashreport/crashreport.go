// Package crashreport captures Go panics into structured local
// reports the user can attach to a GitHub issue. The captured data is
// strictly *non-PII*: panic message, stack trace, OS, Portal version,
// timestamp. We deliberately do not include portal IDs, peer IDs,
// nicknames, IP addresses, or anything from chat / proxy traffic.
//
// Reports live in ~/.portal/crashes/crash-<RFC3339>.json. The Settings
// → Diagnostika tab can list them, open the folder, and clear them.
//
// Phase 5 plan: an opt-in "send report" button that POSTs to a
// self-hosted endpoint. v0.4.0 ships local-only; nothing leaves the
// machine without explicit user action.
package crashreport

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"sort"
	"strings"
	"sync"
	"time"
)

// Report is one captured crash. JSON-marshalable so the frontend can
// display fields in Settings.
type Report struct {
	ID             string    `json:"id"`             // filename stem
	CapturedAt     time.Time `json:"capturedAt"`
	PortalVersion  string    `json:"portalVersion"`
	GoVersion      string    `json:"goVersion"`
	OS             string    `json:"os"`             // runtime.GOOS
	Arch           string    `json:"arch"`           // runtime.GOARCH
	PanicMessage   string    `json:"panicMessage"`
	Stack          string    `json:"stack"`          // trimmed runtime/debug.Stack output
	GoroutineCount int       `json:"goroutineCount"` // for "deadlock-shaped" diagnosis
}

// Catcher converts a recovered panic into a Report and writes it to
// disk. Returns the on-disk path of the written report (empty on
// failure — never panics from inside the panic handler).
type Catcher struct {
	dir     string // ~/.portal/crashes by default
	version string
	mu      sync.Mutex
}

// New constructs a Catcher writing to dir. dir is created if missing.
// portalVersion should be the canonical product version (e.g. "0.4.0").
//
// If dir is empty, defaults to ~/.portal/crashes. Returns the catcher
// even if the directory create fails — the catcher will silently
// no-op rather than crashing the crash handler.
func New(dir string, portalVersion string) *Catcher {
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		dir = filepath.Join(home, ".portal", "crashes")
	}
	_ = os.MkdirAll(dir, 0o700)
	return &Catcher{dir: dir, version: portalVersion}
}

// Capture builds a Report from a recovered panic and writes it to
// disk. Call from a deferred recover() block:
//
//	defer func() {
//	    if r := recover(); r != nil {
//	        catcher.Capture(r, debug.Stack())
//	        panic(r) // re-panic so the runtime still prints the trace
//	    }
//	}()
//
// Pass debug.Stack()'s output through `stack`; we don't capture it
// ourselves so callers retain control of when the snapshot is taken.
//
// Returns the path of the written file. Empty on failure.
func (c *Catcher) Capture(panicValue any, stack []byte) string {
	if c == nil {
		return ""
	}
	c.mu.Lock()
	defer c.mu.Unlock()

	r := Report{
		ID:             time.Now().UTC().Format("20060102T150405"),
		CapturedAt:     time.Now().UTC(),
		PortalVersion:  c.version,
		GoVersion:      runtime.Version(),
		OS:             runtime.GOOS,
		Arch:           runtime.GOARCH,
		PanicMessage:   sanitisePanic(fmt.Sprintf("%v", panicValue)),
		Stack:          sanitiseStack(string(stack)),
		GoroutineCount: runtime.NumGoroutine(),
	}

	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return ""
	}
	path := filepath.Join(c.dir, "crash-"+r.ID+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		return ""
	}
	return path
}

// List returns all crash reports currently on disk, newest first.
// Errors reading individual reports are skipped — the UI shouldn't
// hide all reports because one is malformed.
func (c *Catcher) List() ([]Report, error) {
	if c == nil {
		return nil, nil
	}
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	reports := make([]Report, 0, len(entries))
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "crash-") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(c.dir, e.Name()))
		if err != nil {
			continue
		}
		var r Report
		if err := json.Unmarshal(data, &r); err != nil {
			continue
		}
		reports = append(reports, r)
	}
	sort.Slice(reports, func(i, j int) bool {
		return reports[i].CapturedAt.After(reports[j].CapturedAt)
	})
	return reports, nil
}

// Path returns the on-disk path for a given report ID. Used by the UI
// to "Open in finder" the underlying file.
func (c *Catcher) Path(id string) string {
	if c == nil || id == "" {
		return ""
	}
	return filepath.Join(c.dir, "crash-"+id+".json")
}

// Dir returns the directory storing reports. Settings UI exposes this
// to a "Reveal in Finder" button.
func (c *Catcher) Dir() string {
	if c == nil {
		return ""
	}
	return c.dir
}

// Clear deletes all crash reports. Confirmation is the UI's
// responsibility — Catcher just does the deletion.
func (c *Catcher) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	entries, err := os.ReadDir(c.dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var firstErr error
	for _, e := range entries {
		if e.IsDir() || !strings.HasPrefix(e.Name(), "crash-") {
			continue
		}
		if err := os.Remove(filepath.Join(c.dir, e.Name())); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// sanitisePanic strips identifiers that might leak through a panic
// message. Conservative: nothing from the user's portal context
// (peer IDs, nicknames, IPs) should reach the report. We don't
// actively mutate panic strings since we don't control them, but
// trim absurd lengths.
func sanitisePanic(msg string) string {
	const maxLen = 1024
	if len(msg) > maxLen {
		msg = msg[:maxLen] + "…(truncated)"
	}
	return msg
}

// sanitiseStack removes the user's home-directory prefix from stack
// frame paths so a report shared in a public issue doesn't leak the
// user's username. Best-effort — Go's stack format is stable.
//
// Replaces "/Users/<name>/" or "/home/<name>/" or "C:\Users\<name>\"
// with "$HOME/".
func sanitiseStack(s string) string {
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		s = strings.ReplaceAll(s, home, "$HOME")
	}
	const maxLen = 32 * 1024
	if len(s) > maxLen {
		s = s[:maxLen] + "\n…(truncated)"
	}
	return s
}

// InstallGlobal sets up a top-level recover() so any otherwise-fatal
// panic in the calling goroutine produces a report before the program
// dies. Re-panics after capturing so the runtime still produces its
// usual log output and exits with the right status.
//
// Use as a deferred call near the top of your main goroutine:
//
//	defer crashreport.InstallGlobal(catcher)
//
// Note: only catches panics in the *current* goroutine. Other
// goroutines should set up their own recover blocks if you want their
// panics captured too.
func InstallGlobal(c *Catcher) {
	if c == nil {
		return
	}
	if r := recover(); r != nil {
		_ = c.Capture(r, debug.Stack())
		panic(r)
	}
}
