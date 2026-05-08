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
// label in the combined mesh nickname. Originally a middle-dot ('·')
// for visual distinction, but the deployed signaling server rejects
// it (NICKNAME_INVALID) because its regex restricts nicknames to
// ASCII alphanumeric + a small set of punctuation. The at-sign is
// universally allowed in identifier-style validators and reads
// cleanly: 'texuz@uy', 'texuz@win64'. The frontend's PeerCard /
// PeerServicesGrid render the part before '@' as the username and
// the part after as a small dimmed device tag, so visually it
// still looks like 'texuz · uy'.
const deviceTagSeparator = "@"

// nicknameWithDevice combines the auth nickname with the device
// label using deviceTagSeparator. Used by createPortal / joinPortal
// so the mesh peer announcement carries both. Empty device name
// falls through unchanged. Whitespace inside the device label is
// stripped because some servers reject it; the rendered UI restores
// readability via splitNicknameAndDevice (see frontend's format
// helpers).
func (a *App) nicknameWithDevice(nick string) string {
	dev := strings.TrimSpace(a.CurrentDeviceName())
	if dev == "" {
		return nick
	}
	if strings.Contains(nick, deviceTagSeparator) {
		// Caller already added a device tag; don't double up.
		return nick
	}
	// Strip whitespace from the device tag so 'win 64' becomes
	// 'win64' on the wire — matches the server's typical ident
	// regex without losing meaning.
	dev = strings.ReplaceAll(dev, " ", "")
	return nick + deviceTagSeparator + dev
}
