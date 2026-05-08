// Per-install device label.
//
// Same account on phone + laptop + work desktop should still be
// distinguishable in the room. Storing a small free-form 'device
// name' on each install and prefixing it onto the mesh nickname
// ('texuz · uy') is the simplest user-visible answer that doesn't
// need any signaling-server changes.
//
// Default is platform-derived ('mac', 'win 64', 'linux') so a fresh
// install reads as something rather than blank; the user can rename
// to 'uy' / 'ish' / whatever in Settings → Profile.

package main

import (
	"runtime"
	"strings"

	"portal_traffic_client/storage"
)

// CurrentDeviceName returns the saved device label, or a platform
// default when nothing is set. Bound to JS via Wails so the frontend
// can render the field on Settings → Profile and pre-fill the input
// box on first edit.
func (a *App) CurrentDeviceName() string {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return defaultDeviceName()
	}
	if v, _ := store.GetSetting(storage.KeyDeviceName); v != "" {
		return v
	}
	return defaultDeviceName()
}

// SetDeviceName persists the user-typed label. Empty string clears
// the override (next read falls back to defaultDeviceName).
//
// Trims whitespace and caps at 32 runes — long names break the
// nickname combinator on narrow PeerCard widths.
func (a *App) SetDeviceName(name string) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	clean := strings.TrimSpace(name)
	if r := []rune(clean); len(r) > 32 {
		clean = string(r[:32])
	}
	return store.PutSetting(storage.KeyDeviceName, clean)
}

// defaultDeviceName picks a sensible per-OS label so a fresh install
// has something readable in the room before the user customises.
// 'win 64' / 'win 32' show the bitness because users on a single
// account commonly mix old and new machines and the bitness is the
// cheapest discriminator.
func defaultDeviceName() string {
	switch runtime.GOOS {
	case "darwin":
		return "mac"
	case "windows":
		if runtime.GOARCH == "amd64" || runtime.GOARCH == "arm64" {
			return "win 64"
		}
		return "win 32"
	case "linux":
		return "linux"
	default:
		return runtime.GOOS
	}
}

// deviceTagSeparator is the boundary between username and device
// label in the combined mesh nickname. The deployed signaling
// server's validator (server/nickname.go validNickname) only
// allows `[a-zA-Z0-9_\-.]` and rejects the rest with
// NICKNAME_INVALID. We use '.' because it's the most readable of
// the three (`texuz.uy` reads clearly as a hierarchical handle)
// and the period is unlikely to be ambiguous with mid-username
// punctuation.
const deviceTagSeparator = "."

// deviceTagSanitize strips characters the signaling server
// rejects from a free-form device label: whitespace, punctuation,
// Unicode glyphs. A user typing 'uy mac' or 'work — main' should
// still get something on the wire that won't be bounced as
// NICKNAME_INVALID; we keep ASCII letters, digits, underscores
// and hyphens, drop the rest.
func deviceTagSanitize(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch {
		case r >= 'a' && r <= 'z',
			r >= 'A' && r <= 'Z',
			r >= '0' && r <= '9',
			r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	return b.String()
}

// nicknameWithDevice combines the auth nickname with the device
// label using deviceTagSeparator. Used by createPortal /
// joinPortal so the mesh peer announcement carries both. Empty
// device name falls through unchanged. Idempotency note: this
// helper does NOT try to detect 'already stamped' input — the
// caller is expected to pass the RAW nickname (just the username,
// no separator) every time. createPortal does this by holding the
// raw form locally and only computing the combined form for the
// mesh layer; persistEnter saves the raw form too, so resume calls
// land here with the right input again.
func (a *App) nicknameWithDevice(nick string) string {
	dev := deviceTagSanitize(a.CurrentDeviceName())
	if dev == "" {
		return nick
	}
	return nick + deviceTagSeparator + dev
}
