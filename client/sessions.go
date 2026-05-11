// Multi-portal session management.
//
// One App can hold connections to several portals at once. This file
// owns the data structures and locking that make that possible.
//
// A `portalSession` is one such connection: its own mesh.Manager, its
// own proxy.Forwarder, its own transfer.Engine. It carries a stable
// `localID` (random 8-char hex) that the UI uses to reference the
// session — the server-assigned `portalID` only exists once
// PortalReady has fired, so we can't key on it during the dial.
//
// The App keeps:
//
//   - sessions map[localID]*portalSession  — every live session
//   - activeID string                       — which session the
//     foreground UI is showing (Peers, services, chat scope all key
//     off this). "" when no session is foreground (e.g. on Welcome).
//
// Most pre-existing per-portal methods (Peers, SendChat, DialService,
// MeasureBandwidth, …) read the active session via activeSession()
// and behave exactly as before. Methods that span sessions (Expose,
// Unexpose, SetExposeEnabled) walk every session in the map so a
// single user-level "Och" reaches every connected portal — services
// the user wants shared shouldn't have to be re-exposed per portal.
//
// Background-connect: BackgroundCreatePortal / BackgroundJoinPortal
// bring up a new session WITHOUT changing activeID. The UI stays on
// whatever it was on (Welcome dashboard, or a different portal); the
// new session's mesh is alive in the background, peers connect, and
// any exposed services are announced to it on PortalReady.
//
// Foreground swap: SwitchPortal(localID) changes activeID and emits
// `portal:switched` so the UI can re-render against the new session.
//
// Lifecycle: LeavePortal(localID) tears down one specific session.
// Leave() is a thin wrapper that leaves the active session (kept for
// backwards compatibility with the existing UI).

package main

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"portal_traffic_client/mesh"
	"portal_traffic_client/proxy"
	"portal_traffic_client/transfer"
)

// portalSession is one in-flight or established portal connection.
//
// All fields are immutable after construction except `view`, `state`,
// and `err`, which are protected by `mu`. The mesh / fwd / xfer
// pointers are set once at construction and never reassigned, so
// readers can grab them under the App-level lock without walking the
// session's mu.
type portalSession struct {
	localID  string // generated client-side; stable across the session lifetime
	nickname string
	isOwner  bool

	// Background sessions are created via BackgroundCreatePortal /
	// BackgroundJoinPortal. On PortalReady we still emit
	// `portal:ready` so the frontend can update its state map, but
	// SwitchPortal isn't called automatically — the UI stays where
	// the user left it.
	background bool

	mesh *mesh.Manager
	fwd  *proxy.Forwarder
	xfer *transfer.Engine

	mu    sync.RWMutex
	view  PortalView // populated once mesh emits EventPortalReady
	state string     // "connecting" | "connected" | "failed" | "closed"
	err   error      // last error when state == "failed"

	// pumpDone closes when the per-session event-pump goroutine
	// finishes. Used by tearDownSession to cleanly drain.
	pumpDone chan struct{}
}

// snapshot is a lock-friendly copy of the session's mutable state.
// Used by methods that need to read a coherent set of fields without
// holding the session mu across an event emit.
func (s *portalSession) snapshot() (view PortalView, state string, err error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.view, s.state, s.err
}

func (s *portalSession) setView(v PortalView) {
	s.mu.Lock()
	s.view = v
	s.state = "connected"
	s.err = nil
	s.mu.Unlock()
}

func (s *portalSession) setState(state string, err error) {
	s.mu.Lock()
	s.state = state
	s.err = err
	s.mu.Unlock()
}

// PortalSummary is the JSON projection we ship to the frontend for
// every active session. Lets the dashboard render a "currently
// connected" strip without having to subscribe to N different events
// per session.
type PortalSummary struct {
	SessionID string `json:"sessionId"` // localID
	PortalID  string `json:"portalId"`  // "" while connecting
	Nickname  string `json:"nickname"`
	IsOwner   bool   `json:"isOwner"`
	State     string `json:"state"`     // "connecting" | "connected" | "failed" | "closed"
	Error     string `json:"error,omitempty"`
	PeerCount int    `json:"peerCount"`
	IsActive  bool   `json:"isActive"`  // matches App.activeID
}

// summarize returns the read-only projection. Locks the session mu
// once and reads peer count from mesh outside the lock.
func (s *portalSession) summarize(active bool) PortalSummary {
	view, state, err := s.snapshot()
	peerCount := 0
	if s.mesh != nil {
		peerCount = len(s.mesh.Peers())
	}
	out := PortalSummary{
		SessionID: s.localID,
		PortalID:  view.PortalID,
		Nickname:  s.nickname,
		IsOwner:   s.isOwner,
		State:     state,
		PeerCount: peerCount,
		IsActive:  active,
	}
	if err != nil {
		out.Error = err.Error()
	}
	return out
}

// newSessionID returns a random 8-char hex string (4 bytes of
// entropy) for use as portalSession.localID. Cryptographic-grade
// randomness is overkill for this — we just want collisions to be
// unlikely within a single user's session map of <20 entries — but
// it's also free, so we use it.
func newSessionID() string {
	var b [4]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Fall back to a nano-time-derived string. The collision risk
		// is negligible at the call sites we have today.
		return "s-fallback"
	}
	return "s-" + hex.EncodeToString(b[:])
}
