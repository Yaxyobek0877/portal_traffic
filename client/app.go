// Package main — Wails app glue. Holds an *App which the frontend
// calls into. Methods on *App are auto-bound to JavaScript by Wails;
// they appear under window.go.main.App.* on the frontend side.
//
// The App is intentionally thin: it owns a *mesh.Manager and a
// *proxy.Forwarder, translates frontend calls into mesh operations,
// and pumps mesh events back out to the frontend via runtime.EventsEmit.
package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	stdruntime "runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"portal_traffic_client/crashreport"
	"portal_traffic_client/lanscan"
	"portal_traffic_client/logsink"
	"portal_traffic_client/mesh"
	"portal_traffic_client/nat"
	"portal_traffic_client/proxy"
	"portal_traffic_client/signaling"
	"portal_traffic_client/storage"
	"portal_traffic_client/transfer"
	"portal_traffic_client/updater"
)

// PortalView is the JSON projection of the local portal context for
// the frontend.
//
// SessionID is the local id of the session that owns this portal —
// stable across the session's lifetime and used by the frontend
// store to route the returned PortalView into the right per-session
// slot. CurrentPortal can return a PortalView with empty SessionID
// when no session is foreground; that's fine because the frontend
// only reads PortalView.SessionID after a successful CreatePortal /
// JoinPortal which always carries a session.
type PortalView struct {
	SessionID string `json:"sessionId,omitempty"`
	PortalID  string `json:"portalId"`
	Code      string `json:"code"`
	OwnerID   string `json:"ownerId"`
	OwnPeerID string `json:"ownPeerId"`
	OwnVIP    string `json:"ownVip"`
	IsOwner   bool   `json:"isOwner"`
}

// PeerView is the JSON projection of a remote peer.
type PeerView struct {
	// SessionID is the local id of the portal this peer belongs to.
	// Always set on peer:* events; empty when an older code path
	// constructs a PeerView without context (kept optional for
	// backwards compatibility with the JSON shape).
	SessionID string `json:"sessionId,omitempty"`

	PeerID    string         `json:"peerId"`
	Nickname  string         `json:"nickname"`
	VirtualIP string         `json:"virtualIp"`
	IsOwner   bool           `json:"isOwner"`
	State     string         `json:"state"` // "connecting" | "connected" | "failed" | "closed"
	RTTMs     float64        `json:"rttMs"` // 0 if no pong yet
	BytesSent int64          `json:"bytesSent"`
	BytesRecv int64          `json:"bytesRecv"`
	Services  []ServiceView  `json:"services"`

	// Transport tells the user whether traffic on this peer is going
	// directly (host/srflx — peer-to-peer) or via a TURN relay. Empty
	// strings until ICE has nominated a pair.
	Transport       string `json:"transport"`        // "direct" | "relay" | "" (unknown)
	TransportLocal  string `json:"transportLocal"`   // raw ICE type — "host" | "srflx" | "prflx" | "relay"
	TransportRemote string `json:"transportRemote"`  // raw ICE type from the other side

	// Selected-pair addresses ("192.168.1.53:54538"). Lets the UI
	// answer "is this LAN or internet?" — both 192.168.x.x means LAN.
	PathLocalAddr  string `json:"pathLocalAddr"`
	PathRemoteAddr string `json:"pathRemoteAddr"`
}

// ServiceView is a peer's announced service.
type ServiceView struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Port     int    `json:"port"`
	Target   string `json:"target,omitempty"`
	// Health is "ok" | "down" | "unknown" — populated only for
	// services exposed by the local user; remote peers' services
	// always come back "" (we can't probe their LAN from here).
	Health      string `json:"health,omitempty"`
	HealthError string `json:"healthError,omitempty"`
	// Paused is true for services the user has explicitly paused —
	// the row stays in the UI so they can un-pause, but no peer can
	// dial it and it isn't announced to the mesh.
	Paused bool `json:"paused,omitempty"`
}

// ChatMessage is what the frontend logs in the chat panel.
type ChatMessage struct {
	SessionID string    `json:"sessionId,omitempty"`
	From      string    `json:"from"` // peer id
	Nickname  string    `json:"nickname"`
	Text      string    `json:"text"`
	At        time.Time `json:"at"`
	IsLocal   bool      `json:"isLocal"`
}

// SignalingStatus reports the current connectivity state.
type SignalingStatus struct {
	URL       string `json:"url"`
	Connected bool   `json:"connected"`
	Error     string `json:"error,omitempty"`
}

// TurnConfig is the JSON shape the frontend uses to read/write TURN
// credentials. Empty URL means "no TURN configured".
type TurnConfig struct {
	URL        string `json:"url"`
	Username   string `json:"username"`
	Credential string `json:"credential"`
}

// LocalListener describes a port the OS reports as listening, with
// its protocol (TCP or UDP). UDP entries matter for game traffic —
// CS2 / Valorant / most multiplayer FPS run their game stream on UDP,
// so a TCP-only detector would silently miss them.
type LocalListener struct {
	Port     int    `json:"port"`
	Protocol string `json:"protocol"` // "tcp" | "udp"
	Process  string `json:"process"`
	PID      int    `json:"pid"`
	Local    string `json:"local"`
}

// App is the Wails-bound singleton.
type App struct {
	ctx    context.Context
	logger *slog.Logger

	mu  sync.RWMutex
	url string

	// Multi-portal session state. Replaces the old single
	// mesh/fwd/xfer/nick fields — see client/sessions.go for the full
	// shape and protocol.
	//
	// sessions is keyed by portalSession.localID. activeID is the
	// localID of the foreground session (the one whose Peers/services
	// the UI screen reflects). "" when the user is on Welcome / has
	// no active session. Locking: sessionsMu guards both the map and
	// activeID — never take the session's own mu while holding
	// sessionsMu (the session's pump goroutines try to grab the
	// reverse order on every event).
	sessionsMu sync.RWMutex
	sessions   map[string]*portalSession
	activeID   string

	// nick is the last nickname the user signed up / signed in with.
	// New sessions default to it (CreatePortal/JoinPortal can override).
	nick string

	// store persists settings, portal history, contacts.
	store *storage.Store

	// natResult is the cached startup NAT classification.
	natResult nat.Result

	// cfTurnCache caches Cloudflare TURN credentials between
	// CreatePortal/JoinPortal calls. See cloudflareturn.go.
	cfTurnCache cloudflareTurnCache

	// crashCatcher writes Go panics to ~/.portal/crashes/. Set by main()
	// before Startup runs.
	crashCatcher *crashreport.Catcher

	// updateCache stores the most recent CheckForUpdate result so the UI
	// can render it without forcing a network call on every render.
	updateMu       sync.Mutex
	updateCache    updater.Result
	updateCacheTTL time.Time

	// logSink is the background goroutine that ships log lines to the
	// signaling host's /logs/upload endpoint. nil if upload is disabled
	// or its config didn't resolve. See client/logsink.
	logSink *logsink.Sink

	// exposedHealth caches the latest TCP-probe outcome for every port
	// we've exposed — keyed by port. Updated by the background health
	// loop (App.runHealthLoop) every 30s; consumed by LocalServices to
	// decorate each ServiceView with a green/yellow/red indicator the
	// UI can render. Lets the user spot "camera offline" before a peer
	// has to dial and find out the hard way.
	healthMu      sync.Mutex
	exposedHealth map[serviceHealthKey]serviceHealth

	// approval state — see proxy.Approver. The Forwarder calls into
	// this to decide whether an inbound open request gets dialed or
	// rejected. The map caches per-(peer,port,protocol) decisions so
	// the user isn't re-prompted every time a peer reconnects to the
	// same service this session.
	approvalMu      sync.Mutex
	approvalDecided map[approvalKey]bool          // per-(peer,port,proto) sticky decision
	approvalPending map[string]chan bool          // per-requestID waiter

	// cloudAuth holds the server-account session token + cached user
	// info after a successful CloudSignIn / CloudSignUp. New mesh
	// sessions pass cloudAuth.Token in mesh.Config.CloudAuthToken so
	// the signaling server can stamp portals with the user ID, which
	// in turn makes /api/portals show this device's portals on every
	// other device the same user is signed into. Empty token = anon
	// connection (legacy behaviour).
	cloudMu   sync.RWMutex
	cloudAuth cloudAuthState
}

type cloudAuthState struct {
	Token    string `json:"token,omitempty"`
	UserID   string `json:"userId,omitempty"`
	Username string `json:"username,omitempty"`
}

type approvalKey struct {
	peerID   string
	port     int
	protocol string
}

// serviceHealthKey identifies a single exposed service for the health
// cache. Keying by (port, protocol) — not just port — keeps a TCP
// probe failure from poisoning the UDP entry on the same port (and
// vice versa). Common case: a user exposes both TCP+UDP on :80 to a
// LAN device that only answers TCP; UDP should stay 'unknown', not
// be inferred 'down' from the TCP outcome.
type serviceHealthKey struct {
	port     int
	protocol string // "tcp" | "udp"
}

type serviceHealth struct {
	target    string
	status    string // "ok" | "down" | "unknown"
	err       string
	checkedAt time.Time
}

// NewApp constructs the app singleton; main.go binds it.
func NewApp(logger *slog.Logger) *App {
	return &App{logger: logger}
}

// Startup is called once Wails has the runtime context. We use it
// to remember ctx for runtime.EventsEmit calls from any goroutine,
// open the on-disk store, and kick off a background NAT classification.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.logger.Info("portal app starting")

	// Initialise the session map up front so every read path can rely
	// on it being non-nil even before the first portal is dialed.
	a.sessionsMu.Lock()
	a.sessions = make(map[string]*portalSession)
	a.sessionsMu.Unlock()

	// Approval caches.
	a.approvalMu.Lock()
	a.approvalDecided = make(map[approvalKey]bool)
	a.approvalPending = make(map[string]chan bool)
	a.approvalMu.Unlock()

	// Open SQLite. If it fails we proceed without persistence — the
	// app still works, settings just don't survive restarts.
	store, err := storage.Open("")
	if err != nil {
		a.logger.Warn("storage open failed; running without persistence", "err", err)
	} else {
		a.store = store
		a.url = store.GetOr(storage.KeySignalingURL, "")
		a.nick = store.GetOr(storage.KeyNickname, "")
	}

	// Restore the cloud-account session token (if any) so the next
	// portal we open includes ?token=... on its WebSocket connect.
	// No network call — just reads the local file.
	a.loadCloudAuth()

	a.startLogSink()
	go a.runHealthLoop()

	// Run NAT detection in the background; UI subscribes to "nat:result".
	go func() {
		dctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		r, err := nat.Detect(dctx, nat.DefaultServers)
		if err != nil {
			a.logger.Debug("nat detect", "err", err)
			return
		}
		a.mu.Lock()
		a.natResult = r
		a.mu.Unlock()
		runtime.EventsEmit(ctx, "nat:result", r)
	}()
}

// startLogSink wires up the background log uploader if it's enabled
// (default-on, opt-out via KeyLogUpload="0"). Resolves the upload URL
// from the configured signaling URL by default, can be overridden via
// KeyLogUploadURL.
func (a *App) startLogSink() {
	enabled := "1"
	uploadURL := ""
	signalingURL := ""
	if a.store != nil {
		enabled = a.store.GetOr(storage.KeyLogUpload, "1")
		uploadURL = a.store.GetOr(storage.KeyLogUploadURL, "")
		signalingURL = a.store.GetOr(storage.KeySignalingURL, "")
	}
	if enabled == "0" {
		a.logger.Info("logsink: disabled by setting")
		return
	}
	if uploadURL == "" {
		uploadURL = logsink.DeriveUploadURL(signalingURL)
	}
	if uploadURL == "" {
		uploadURL = logsink.DeriveUploadURL(signaling.DefaultURL)
	}
	if uploadURL == "" {
		a.logger.Info("logsink: no upload URL resolvable, skipping")
		return
	}

	clientID, err := logsink.LoadOrMintClientID()
	if err != nil {
		a.logger.Warn("logsink: client id mint failed", "err", err)
		return
	}

	a.mu.Lock()
	if a.logSink != nil {
		// Already running (Startup called twice somehow).
		a.mu.Unlock()
		return
	}
	a.logSink = logsink.Start(logsink.Config{
		LogFile:   LogFile,
		UploadURL: uploadURL,
		ClientID:  clientID,
		Version:   Version,
		Logger:    a.logger,
	})
	a.mu.Unlock()
}

// Shutdown cleans up every session, the SQLite store, and the
// log-uploader.
func (a *App) Shutdown(ctx context.Context) {
	a.mu.Lock()
	store := a.store
	sink := a.logSink
	a.store = nil
	a.logSink = nil
	a.mu.Unlock()
	if sink != nil {
		// Triggers one final drain so the very last lines (incl. the
		// "shutdown" record itself once it lands) get uploaded.
		sink.Stop()
	}

	// Snapshot every session under the sessions lock, then close
	// outside the lock so a slow mesh.Leave doesn't stall the
	// shutdown of unrelated subsystems.
	a.sessionsMu.Lock()
	all := a.sessions
	a.sessions = make(map[string]*portalSession)
	a.activeID = ""
	a.sessionsMu.Unlock()
	for _, s := range all {
		a.closeSession(s)
	}

	if store != nil {
		_ = store.Close()
	}
}

// ----------------------------------------------------------------------------
// Bound methods (called from the frontend)
// ----------------------------------------------------------------------------

// SignalingURL returns the configured signaling URL (effectively the
// project default unless overridden in settings).
func (a *App) SignalingURL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.url != "" {
		return a.url
	}
	return signaling.DefaultURL
}

// NATInfo returns the cached NAT classification result. Empty Type if
// detection hasn't finished yet — the UI also subscribes to nat:result
// for live updates.
func (a *App) NATInfo() nat.Result {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.natResult
}

// SaveDir reports where received files land. UI displays it next to
// the file-transfer affordance. Every session shares the same save
// dir (transfer.NewEngine defaults to it), so the active session's
// engine — or the default — is fine to read from.
func (a *App) SaveDir() string {
	if s := a.activeSession(); s != nil && s.xfer != nil {
		return s.xfer.SaveDir()
	}
	for _, s := range a.allSessions() {
		if s.xfer != nil {
			return s.xfer.SaveDir()
		}
	}
	// Fallback so the UI can show something pre-portal.
	fallback, _ := os.UserHomeDir()
	return filepath.Join(fallback, "Downloads", "Portal")
}

// SetSignalingURL changes the URL we'll connect to on next create/join.
// Persists immediately. Has no effect on a session already in flight.
func (a *App) SetSignalingURL(url string) error {
	url = strings.TrimSpace(url)
	if url == "" {
		return errors.New("URL bo'sh bo'lmasligi kerak")
	}
	if !strings.HasPrefix(url, "ws://") && !strings.HasPrefix(url, "wss://") {
		return errors.New("URL ws:// yoki wss:// bilan boshlanishi kerak")
	}
	a.mu.Lock()
	a.url = url
	store := a.store
	a.mu.Unlock()
	if store != nil {
		_ = store.PutSetting(storage.KeySignalingURL, url)
	}
	return nil
}

// GetTurnConfig returns the persisted TURN credentials. Empty URL
// means "not configured" — without TURN, peers behind symmetric NAT
// can't connect.
func (a *App) GetTurnConfig() TurnConfig {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return TurnConfig{}
	}
	return TurnConfig{
		URL:        store.GetOr(storage.KeyTurnURL, ""),
		Username:   store.GetOr(storage.KeyTurnUsername, ""),
		Credential: store.GetOr(storage.KeyTurnCredential, ""),
	}
}

// SetTurnConfig persists TURN credentials. Empty URL clears the
// configuration; whitespace-only fields are treated as empty. Takes
// effect on the next CreatePortal / JoinPortal call.
func (a *App) SetTurnConfig(c TurnConfig) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage mavjud emas")
	}
	c.URL = strings.TrimSpace(c.URL)
	c.Username = strings.TrimSpace(c.Username)
	c.Credential = strings.TrimSpace(c.Credential)

	if c.URL != "" {
		// Accept one or more URLs separated by newlines/commas/spaces.
		// Each must start with turn:/turns:.
		anyValid := false
		for _, raw := range strings.FieldsFunc(c.URL, func(r rune) bool {
			return r == ',' || r == '\n' || r == ' ' || r == '\t' || r == ';'
		}) {
			raw = strings.TrimSpace(raw)
			if raw == "" {
				continue
			}
			if !strings.HasPrefix(raw, "turn:") && !strings.HasPrefix(raw, "turns:") {
				return errors.New(
					"har bir URL turn: yoki turns: bilan boshlanishi kerak (topildi: " + raw + ")")
			}
			anyValid = true
		}
		if !anyValid {
			return errors.New("kamida bitta turn:// URL bering")
		}
	}
	_ = store.PutSetting(storage.KeyTurnURL, c.URL)
	_ = store.PutSetting(storage.KeyTurnUsername, c.Username)
	_ = store.PutSetting(storage.KeyTurnCredential, c.Credential)
	return nil
}

// ----------------------------------------------------------------------------
// File transfer
// ----------------------------------------------------------------------------

// SendFile asks the OS to pick a file via Wails dialog and starts a
// transfer to peerID. Returns the assigned xfer_id (string for the JS
// boundary). Routes to whichever portal contains the peer.
func (a *App) SendFile(peerID string) (string, error) {
	s := a.sessionWithPeer(peerID)
	if s == nil || s.xfer == nil {
		return "", errors.New("portal yo'q")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Yuborish uchun fayl tanlang",
	})
	if err != nil || path == "" {
		return "", err
	}
	xferID, err := s.xfer.SendFile(peerID, path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", xferID), nil
}

// SendFilePath sends a file by absolute path (used by drag-and-drop;
// the UI gets the path from Wails' OnFileDrop event).
func (a *App) SendFilePath(peerID, path string) (string, error) {
	s := a.sessionWithPeer(peerID)
	if s == nil || s.xfer == nil {
		return "", errors.New("portal yo'q")
	}
	if path == "" {
		return "", errors.New("fayl yo'li bo'sh")
	}
	xferID, err := s.xfer.SendFile(peerID, path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", xferID), nil
}

// OpenSaveDir opens the OS file manager at the receive folder.
func (a *App) OpenSaveDir() error {
	dir := a.SaveDir()
	_ = os.MkdirAll(dir, 0o755)
	runtime.BrowserOpenURL(a.ctx, "file://"+dir)
	return nil
}

// ----------------------------------------------------------------------------
// Persistence — recent portals
// ----------------------------------------------------------------------------

// RecentPortals returns the latest N portal_history rows. Always
// returns a non-nil slice so JSON serialises as `[]` not `null` —
// the frontend treats history as an array and `.length` on null
// throws.
func (a *App) RecentPortals(n int) []storage.HistoryEntry {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	out := []storage.HistoryEntry{}
	if store == nil {
		return out
	}
	if n <= 0 {
		n = 10
	}
	rows, err := store.RecentHistory(n)
	if err != nil || rows == nil {
		return out
	}
	return rows
}

// ClearHistory wipes portal history.
func (a *App) ClearHistory() error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	return store.ClearHistory()
}

// RenamePortal sets a user-friendly label on a portal_history row.
// Empty label clears the existing one (the UI then falls back to the
// portal ID). Returns an error if the row doesn't exist so the caller
// can refresh — likely the user removed it from another window.
func (a *App) RenamePortal(historyID int64, label string) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage mavjud emas")
	}
	label = strings.TrimSpace(label)
	if len([]rune(label)) > 60 {
		// 60 chars is plenty for a friendly room name and prevents
		// pathological inputs from breaking the recent-portals layout.
		return errors.New("nom juda uzun (max 60 ta belgi)")
	}
	return store.SetHistoryLabel(historyID, label)
}

// RemoveRecentPortal deletes a single history row. Used by the X
// button on a recent-portals entry. No error if the id is gone.
func (a *App) RemoveRecentPortal(historyID int64) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	return store.DeleteHistory(historyID)
}

// CreatePortal dials signaling and creates a new portal in the
// foreground (becomes the active session). Returns the new portal
// info once the server has confirmed.
func (a *App) CreatePortal(nickname string, publicNick bool) (PortalView, error) {
	return a.createPortal(nickname, true /*makeActive*/)
}

// BackgroundCreatePortal creates a new portal but leaves the
// foreground UI on whatever it was on. Used by the dashboard's
// "Fonda ulash" affordance — the user gets a portal connection up
// and running for someone to join, while staying on Welcome (or in
// a different portal they're already foregrounded into).
func (a *App) BackgroundCreatePortal(nickname string) (PortalView, error) {
	return a.createPortal(nickname, false /*makeActive*/)
}

func (a *App) createPortal(nickname string, makeActive bool) (PortalView, error) {
	// Stamp the device label onto the mesh nickname so multi-device
	// users (same account, several boxes) are distinguishable in the
	// room. The combined form ('texuz.uy') goes to the mesh layer;
	// persistEnter receives the RAW form ('texuz') so resume calls
	// re-stamp with the CURRENT device label rather than re-stamping
	// an already-stamped input (which would land us at 'texuz.uy.uy').
	rawNick := nickname
	nickname = a.nicknameWithDevice(nickname)
	s, err := a.bringUpSession(nickname, makeActive, true /*isOwner*/)
	if err != nil {
		return PortalView{}, err
	}
	if err := s.mesh.CreatePortal(a.ctx); err != nil {
		a.dropSession(s)
		return PortalView{}, err
	}
	pv, err := a.waitPortalReady(s, 8*time.Second)
	if err != nil {
		a.dropSession(s)
		return pv, err
	}
	pv.SessionID = s.localID
	s.setView(pv)
	a.persistEnter(pv, rawNick, true)
	if makeActive && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "portal:switched",
			map[string]string{"sessionId": s.localID, "portalId": pv.PortalID})
	}
	return pv, nil
}

// JoinPortal dials signaling and joins by ID + code in the
// foreground.
func (a *App) JoinPortal(nickname, portalID, code string) (PortalView, error) {
	return a.joinPortal(nickname, portalID, code, true /*makeActive*/)
}

// BackgroundJoinPortal joins without taking foreground. The session
// runs alongside whichever portal the user is currently in (or the
// Welcome dashboard).
func (a *App) BackgroundJoinPortal(nickname, portalID, code string) (PortalView, error) {
	return a.joinPortal(nickname, portalID, code, false /*makeActive*/)
}

func (a *App) joinPortal(nickname, portalID, code string, makeActive bool) (PortalView, error) {
	if portalID == "" || code == "" {
		return PortalView{}, errors.New("portal ID va kod bo'sh bo'lmasligi kerak")
	}
	// Same device-stamping as createPortal — see comment there.
	rawNick := nickname
	nickname = a.nicknameWithDevice(nickname)
	s, err := a.bringUpSession(nickname, makeActive, false /*isOwner*/)
	if err != nil {
		return PortalView{}, err
	}
	// Stash the typed code into the session view up front. The mesh
	// won't echo it back from the server, so we need to remember it
	// for the eventual EventPortalReady so the cached view (used by
	// CurrentPortal / persisted history) carries it. relayEvent's
	// PortalReady branch preserves a non-empty code if already set.
	s.mu.Lock()
	s.view.Code = code
	s.mu.Unlock()
	if err := s.mesh.JoinPortal(a.ctx, portalID, code); err != nil {
		a.dropSession(s)
		return PortalView{}, err
	}
	pv, err := a.waitPortalReady(s, 8*time.Second)
	if err != nil {
		a.dropSession(s)
		return pv, err
	}
	pv.SessionID = s.localID
	pv.Code = code
	a.persistEnter(pv, rawNick, false)
	pv.Code = "" // don't surface the code on the joiner UI side
	if makeActive && a.ctx != nil {
		runtime.EventsEmit(a.ctx, "portal:switched",
			map[string]string{"sessionId": s.localID, "portalId": pv.PortalID})
	}
	return pv, nil
}

// dropSession deletes a session from the map and tears it down.
// Used when create/join fails so a stillborn session doesn't linger
// in the active-portals strip.
func (a *App) dropSession(s *portalSession) {
	if s == nil {
		return
	}
	a.sessionsMu.Lock()
	delete(a.sessions, s.localID)
	if a.activeID == s.localID {
		a.activeID = ""
	}
	a.sessionsMu.Unlock()
	a.closeSession(s)
}

// persistEnter records this portal in history, saves the nickname
// for next launch, and writes an active-session row so the portal
// can be auto-restored after an app restart. Best-effort; failures
// are logged at debug level.
func (a *App) persistEnter(pv PortalView, nickname string, isOwner bool) {
	a.mu.Lock()
	a.nick = nickname
	store := a.store
	a.mu.Unlock()
	if store == nil {
		return
	}
	_ = store.PutSetting(storage.KeyNickname, nickname)
	_ = store.AddHistory(storage.HistoryEntry{
		PortalID: pv.PortalID,
		Code:     pv.Code,
		Nickname: nickname,
		IsOwner:  isOwner,
	})
	// active_sessions: row keyed on portal_id so the resume path on
	// next launch knows what to dial. Owner rows that recreate with
	// a fresh portal_id leave the old row behind here, but the next
	// resume call only walks rows that match a live PortalID — and
	// LeaveAllPortals from sign-out clears the table anyway.
	_ = store.UpsertActiveSession(storage.ActiveSessionRow{
		PortalID: pv.PortalID,
		Code:     pv.Code,
		Nickname: nickname,
		IsOwner:  isOwner,
	})
}

// Leave detaches from the active portal. Other background sessions
// keep running. Returns nil if there's nothing active.
func (a *App) Leave() error {
	s := a.activeSession()
	if s == nil {
		return nil
	}
	return a.LeavePortal(s.localID)
}

// LeavePortal tears down a specific session by its localID. Emits
// portal:closed so the UI can drop it from its map. Used by the
// active-connections strip's X button.
//
// Deletes the matching active_sessions row — explicit X means
// 'don't bring this back next launch'. The portal_history row
// stays untouched so the dashboard's Recent strip still shows the
// portal as a rejoin candidate.
func (a *App) LeavePortal(sessionID string) error {
	a.sessionsMu.Lock()
	s := a.sessions[sessionID]
	delete(a.sessions, sessionID)
	if a.activeID == sessionID {
		a.activeID = ""
	}
	a.sessionsMu.Unlock()
	if s == nil {
		return nil
	}
	pid := s.snapshotPortalID()
	a.closeSession(s)
	a.forgetActiveSession(pid)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "portal:closed", map[string]string{"sessionId": sessionID})
	}
	return nil
}

// LeaveAllPortals tears down every active session. Useful for sign-
// out flows where we don't want the meshes hanging around after the
// user vaults the app.
//
// Wipes active_sessions in the same shot — sign-out is the user
// saying 'forget I was here'.
func (a *App) LeaveAllPortals() error {
	a.sessionsMu.Lock()
	all := a.sessions
	a.sessions = make(map[string]*portalSession)
	a.activeID = ""
	a.sessionsMu.Unlock()
	for _, s := range all {
		a.closeSession(s)
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "portal:closed",
				map[string]string{"sessionId": s.localID})
		}
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store != nil {
		_ = store.ClearActiveSessions()
	}
	return nil
}

// forgetActiveSession deletes the active_sessions row keyed on the
// portal_id, if any. Best-effort — runs after closeSession on the
// LeavePortal path and on the server-side EventPortalClosed branch
// in relayEvent.
func (a *App) forgetActiveSession(portalID string) {
	if portalID == "" {
		return
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return
	}
	_ = store.DeleteActiveSession(portalID)
}

// ResumeActiveSessions walks every row in the active_sessions table
// and re-dials it as a background session. Called by the frontend
// once the user has unlocked the vault — we don't want to silently
// reconnect ahead of authentication, even though the data on disk
// is the same.
//
// Owner rows: the original portal_id is dead (server destroys an
// owner's portal on disconnect), so we BackgroundCreatePortal with
// the saved nickname. The new session gets a fresh portal_id; the
// old active_sessions row is replaced via persistEnter's UpsertActive
// which rewrites by the new portal_id. The stale row (old portal_id)
// is dropped after the create succeeds.
//
// Joiner rows: try BackgroundJoinPortal with the saved portal_id +
// code. On 'no such portal' or any other failure, drop the row —
// the portal is gone, no point retrying every launch.
//
// Concurrent: each row gets its own goroutine so a slow signaling
// connect doesn't stall others. Returns immediately; failures land
// in the log.
func (a *App) ResumeActiveSessions() error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	rows, err := store.ListActiveSessions()
	if err != nil {
		a.logger.Warn("resume: list active sessions", "err", err)
		return err
	}
	if len(rows) == 0 {
		return nil
	}
	a.logger.Info("resume: dialing saved sessions", "count", len(rows))
	for _, row := range rows {
		row := row
		go a.resumeOne(row)
	}
	return nil
}

// resumeOne handles a single active-session row. Best-effort —
// success rewrites the row with the freshly-issued portal_id,
// failure leaves the row intact so the next launch can retry.
//
// Earlier versions dropped the row on any failure, which made a
// transient server hiccup (e.g. a regex tightening that bumped
// every owner-row to NICKNAME_INVALID once) wipe the user's whole
// auto-reconnect set. Leaving rows in place on transient failure
// is the safer default: a permanently-dead portal still self-heals
// because BackgroundJoinPortal's mesh layer surfaces 'no such
// portal' once the user manually retries, and persistEnter on a
// successful re-create overwrites the stale row in place anyway.
func (a *App) resumeOne(row storage.ActiveSessionRow) {
	if row.IsOwner {
		a.logger.Info("resume: re-creating owner session",
			"old_portal_id", row.PortalID, "nickname", row.Nickname)
		pv, err := a.BackgroundCreatePortal(row.Nickname)
		if err != nil {
			a.logger.Warn("resume: owner re-create failed; keeping row for retry",
				"old_portal_id", row.PortalID, "err", err)
			return
		}
		// Success: persistEnter has already inserted a row keyed on
		// the new portal_id. Drop the stale row so we don't pile up
		// dead pointers across reboots.
		if pv.PortalID != row.PortalID {
			a.forgetActiveSession(row.PortalID)
		}
		return
	}
	if row.Code == "" {
		a.logger.Warn("resume: skipping joiner row with empty code",
			"portal_id", row.PortalID)
		a.forgetActiveSession(row.PortalID)
		return
	}
	a.logger.Info("resume: rejoining session",
		"portal_id", row.PortalID, "nickname", row.Nickname)
	if _, err := a.BackgroundJoinPortal(row.Nickname, row.PortalID, row.Code); err != nil {
		// Only drop the row when the portal is provably gone — keeping
		// it on transient errors lets the next launch retry without
		// the user having to remember their friend's id+code.
		permanent := false
		if errMsg := err.Error(); errMsg != "" {
			lower := strings.ToLower(errMsg)
			if strings.Contains(lower, "no such portal") ||
				strings.Contains(lower, "portal_full") ||
				strings.Contains(lower, "portal_locked") ||
				strings.Contains(lower, "code does not match") {
				permanent = true
			}
		}
		if permanent {
			a.logger.Warn("resume: portal permanently gone; dropping row",
				"portal_id", row.PortalID, "err", err)
			a.forgetActiveSession(row.PortalID)
		} else {
			a.logger.Warn("resume: join failed; keeping row for retry",
				"portal_id", row.PortalID, "err", err)
		}
	}
}

// SwitchPortal makes a different session foreground. The frontend's
// store re-reads peers / chat / services from the new session and
// re-renders the Portal screen. Returns ErrNotFound if the
// sessionID isn't in the map.
func (a *App) SwitchPortal(sessionID string) error {
	a.sessionsMu.Lock()
	s, ok := a.sessions[sessionID]
	if !ok {
		a.sessionsMu.Unlock()
		return errors.New("session topilmadi")
	}
	a.activeID = sessionID
	view := s.view
	a.sessionsMu.Unlock()
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "portal:switched",
			map[string]string{"sessionId": sessionID, "portalId": view.PortalID})
	}
	return nil
}

// ActivePortals returns a summary of every live session so the
// Welcome dashboard can render its "Faol ulanishlar" strip without
// having to subscribe to N events.
func (a *App) ActivePortals() []PortalSummary {
	a.sessionsMu.RLock()
	all := make([]*portalSession, 0, len(a.sessions))
	for _, s := range a.sessions {
		all = append(all, s)
	}
	active := a.activeID
	a.sessionsMu.RUnlock()
	out := make([]PortalSummary, 0, len(all))
	for _, s := range all {
		out = append(out, s.summarize(s.localID == active))
	}
	// Stable order: active first, then by nickname so the strip
	// doesn't reshuffle every render.
	sort.Slice(out, func(i, j int) bool {
		if out[i].IsActive != out[j].IsActive {
			return out[i].IsActive
		}
		if out[i].Nickname != out[j].Nickname {
			return out[i].Nickname < out[j].Nickname
		}
		return out[i].SessionID < out[j].SessionID
	})
	return out
}

// ActiveSessionID returns the foreground session's localID, or ""
// when none. Lets the frontend store decide which session's data
// to render right after a refresh / cold restart.
func (a *App) ActiveSessionID() string {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()
	return a.activeID
}

// CurrentPortal returns the active session's portal info, or zero-
// value if none.
func (a *App) CurrentPortal() PortalView {
	s := a.activeSession()
	if s == nil || s.mesh == nil {
		return PortalView{}
	}
	v := portalToView(s.mesh.Portal())
	v.SessionID = s.localID
	return v
}

// BandwidthResult mirrors mesh.BandwidthResult for the JSON wire to
// the frontend. Receiver-measured Mbps is the honest number.
type BandwidthResult struct {
	PeerID     string  `json:"peerId"`
	Mbps       float64 `json:"mbps"`
	BytesSent  int64   `json:"bytesSent"`
	DurationMs float64 `json:"durationMs"`
}

// MeasureBandwidth runs a 3s active probe with the named peer and
// returns the receiver-measured throughput. Probes whichever portal
// the peer belongs to, picking the active session by default.
func (a *App) MeasureBandwidth(peerID string) (BandwidthResult, error) {
	s := a.sessionWithPeer(peerID)
	if s == nil {
		return BandwidthResult{}, errors.New("portal yo'q")
	}
	r, err := s.mesh.MeasureBandwidth(peerID)
	if err != nil {
		return BandwidthResult{}, err
	}
	return BandwidthResult{
		PeerID:     r.PeerID,
		Mbps:       r.Mbps,
		BytesSent:  r.BytesSent,
		DurationMs: r.DurationMs,
	}, nil
}

// Peers returns a snapshot of the active portal's peers. Older UI
// code calls this on every tick; new multi-portal-aware code should
// prefer SessionPeers(localID).
func (a *App) Peers() []PeerView {
	s := a.activeSession()
	if s == nil || s.mesh == nil {
		return []PeerView{}
	}
	return peersForSession(s)
}

// SessionPeers returns the peers of a specific session by localID.
// Empty slice if the session doesn't exist or has no mesh yet.
func (a *App) SessionPeers(sessionID string) []PeerView {
	s := a.sessionByLocalID(sessionID)
	if s == nil || s.mesh == nil {
		return []PeerView{}
	}
	return peersForSession(s)
}

func peersForSession(s *portalSession) []PeerView {
	peers := s.mesh.Peers()
	out := make([]PeerView, 0, len(peers))
	for _, p := range peers {
		v := peerToView(p)
		v.SessionID = s.localID
		out = append(out, v)
	}
	return out
}

// sessionWithPeer returns the session that currently has a peer
// with the given peerID. Used by methods that take a peer-id from
// the UI but don't get a session-id (DialService, MeasureBandwidth,
// SendFile). Falls back to the active session if no match — older
// callers don't always carry a session-id argument.
func (a *App) sessionWithPeer(peerID string) *portalSession {
	for _, s := range a.allSessions() {
		if s.mesh == nil {
			continue
		}
		for _, p := range s.mesh.Peers() {
			if p.ID == peerID {
				return s
			}
		}
	}
	return a.activeSession()
}

// SendChat broadcasts a chat message to every peer. Returns peer count.
func (a *App) SendChat(text string) int {
	return a.SendChatTo(a.ActiveSessionID(), text)
}

// SendChatTo broadcasts a chat message to a specific session's
// peers. Returns peer count. Called by the new multi-portal aware
// UI when the user types in a chat panel attached to a specific
// session.
func (a *App) SendChatTo(sessionID, text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	if sessionID == "" {
		sessionID = a.ActiveSessionID()
	}
	s := a.sessionByLocalID(sessionID)
	if s == nil || s.mesh == nil {
		return 0
	}
	n := s.mesh.SendChat(text)
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "chat", ChatMessage{
			SessionID: s.localID,
			From:      s.mesh.MyPeerID(),
			Nickname:  s.nickname,
			Text:      text,
			At:        time.Now(),
			IsLocal:   true,
		})
	}
	return n
}

// ----------------------------------------------------------------------------
// Service / proxy bindings
// ----------------------------------------------------------------------------

// TurnTestResult is what TestTurn returns.
type TurnTestResult struct {
	OK          bool     `json:"ok"`
	Message     string   `json:"message"`
	Types       []string `json:"types"`       // candidate types we saw
	HadRelay    bool     `json:"hadRelay"`
	GatherMs    int64    `json:"gatherMs"`
	URLs        []string `json:"urls"`
}

// TestTurn spins up a temporary RTCPeerConnection with the currently
// configured TURN servers and reports which kinds of ICE candidates
// it gathers within a 5-second window. If the result includes a
// "relay" candidate, TURN is reachable and the peer mesh should
// succeed even on Symmetric NAT.
//
// This is the single most informative diagnostic for "why aren't my
// peers connecting?" because it sidesteps the rest of the WebRTC
// state machine and tests just the TURN reachability.
func (a *App) TestTurn() TurnTestResult {
	t := a.GetTurnConfig()
	res := TurnTestResult{}

	iceServers := []webrtc.ICEServer{
		{URLs: []string{"stun:stun.l.google.com:19302"}},
	}
	if t.URL != "" {
		urls := splitTurnURLsForTest(t.URL)
		res.URLs = urls
		if len(urls) > 0 {
			iceServers = append(iceServers, webrtc.ICEServer{
				URLs:       urls,
				Username:   t.Username,
				Credential: t.Credential,
			})
		}
	} else {
		res.Message = "TURN sozlanmagan — Settings → TURN bo'limidan sozlang"
		return res
	}

	api := webrtc.NewAPI()
	pc, err := api.NewPeerConnection(webrtc.Configuration{ICEServers: iceServers})
	if err != nil {
		res.Message = "PeerConnection yaratib bo'lmadi: " + err.Error()
		return res
	}
	defer pc.Close()

	// We need at least one transceiver/data channel to trigger
	// candidate gathering. A throwaway data channel does the job.
	if _, err := pc.CreateDataChannel("test", nil); err != nil {
		res.Message = "test channel: " + err.Error()
		return res
	}

	seen := map[string]int{}
	var seenMu sync.Mutex
	pc.OnICECandidate(func(cand *webrtc.ICECandidate) {
		if cand == nil {
			return
		}
		seenMu.Lock()
		seen[cand.Typ.String()]++
		seenMu.Unlock()
		a.logger.Info("turn test candidate",
			"type", cand.Typ.String(),
			"addr", cand.Address,
			"port", cand.Port,
		)
	})

	offer, err := pc.CreateOffer(nil)
	if err != nil {
		res.Message = "CreateOffer: " + err.Error()
		return res
	}
	if err := pc.SetLocalDescription(offer); err != nil {
		res.Message = "SetLocalDescription: " + err.Error()
		return res
	}

	deadline := time.After(6 * time.Second)
	start := time.Now()
loop:
	for {
		select {
		case <-deadline:
			break loop
		case <-time.After(150 * time.Millisecond):
			seenMu.Lock()
			if seen["relay"] > 0 {
				seenMu.Unlock()
				break loop
			}
			seenMu.Unlock()
		}
	}
	res.GatherMs = time.Since(start).Milliseconds()

	seenMu.Lock()
	for k := range seen {
		res.Types = append(res.Types, k)
	}
	res.HadRelay = seen["relay"] > 0
	seenMu.Unlock()
	sort.Strings(res.Types)

	res.OK = res.HadRelay
	switch {
	case res.HadRelay:
		res.Message = "TURN ishlamoqda — ulanish hosil bo'lishi kerak ✓"
	case len(res.Types) == 0:
		res.Message = "Bironta candidate yig'ilmadi — tarmoq bloklayotgan bo'lishi mumkin"
	default:
		res.Message = "TURN dan relay candidate kelmadi — credentials noto'g'ri yoki TURN serveri yiqilgan"
	}
	a.logger.Info("turn test", "ok", res.OK, "types", res.Types, "hadRelay", res.HadRelay)
	return res
}

// splitTurnURLsForTest mirrors mesh.splitTurnURLs but is duplicated
// here to avoid importing the mesh package's internal helper.
func splitTurnURLsForTest(s string) []string {
	out := []string{}
	for _, raw := range strings.FieldsFunc(s, func(r rune) bool {
		return r == ',' || r == '\n' || r == ' ' || r == '\t' || r == ';'
	}) {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if !strings.HasPrefix(raw, "turn:") && !strings.HasPrefix(raw, "turns:") {
			continue
		}
		out = append(out, raw)
	}
	return out
}

// AppVersion returns the running Portal version (the const declared in
// main.go). Surfaced in Settings → About.
func (a *App) AppVersion() string { return Version }

// InstallUpdate downloads the asset from the most recent
// CheckForUpdate result, stages it next to a swap script, runs the
// swap script in detached mode, and quits the running app so the
// script can replace the binary cleanly. The user sees the dock
// icon disappear for ~1 second and then the new version come up.
//
// Returns an error string when staging fails (network, archive
// corrupt, no asset for the running OS); empty string on success
// (the app will be terminated by the time the JS-side caller's
// promise resolves, so seeing an empty error means 'we're going
// down, restart imminent'.).
func (a *App) InstallUpdate() string {
	res := a.CheckForUpdate(false)
	if !res.Available {
		return "no update available"
	}
	if res.AssetForOS == "" {
		return "no download URL for this platform"
	}

	// Determine the swap target. On macOS we replace the .app bundle
	// (the inner Mach-O has parents we can't blow away while running);
	// on Windows / Linux we replace the binary file directly.
	exe, err := os.Executable()
	if err != nil {
		return "locate self: " + err.Error()
	}
	target := exe
	if stdruntime.GOOS == "darwin" {
		target = macAppPathFromExecutable(exe)
	}

	pending, err := updater.PrepareInstall(context.Background(), res.AssetForOS, target, nil)
	if err != nil {
		a.logger.Warn("update: prepare install failed", "err", err)
		return err.Error()
	}
	a.logger.Info("update: applying",
		"target", pending.CurrentPath, "staged", pending.StagedPath)

	// Hand the swap script the goroutine it needs to outlive us, then
	// quit the Wails app. The brief sleep before quit gives the JS
	// side time to update its UI ('Restarting...').
	if err := pending.Apply(); err != nil {
		a.logger.Warn("update: apply failed", "err", err)
		return err.Error()
	}
	go func() {
		time.Sleep(500 * time.Millisecond)
		runtime.Quit(a.ctx)
	}()
	return ""
}

// CheckForUpdate polls GitHub Releases for a newer version and returns
// what it found. Result.Available is true only when a strictly newer
// version is published. Errors (offline, rate-limit) come back via
// Result.Error so the UI can render them without try/catch.
//
// 24-hour in-memory cache: repeated calls within that window return
// the previous result without a network hit. Pass refresh=true to
// force a fresh check.
func (a *App) CheckForUpdate(refresh bool) updater.Result {
	a.updateMu.Lock()
	if !refresh && time.Now().Before(a.updateCacheTTL) && a.updateCache.CheckedAt.After(time.Time{}) {
		out := a.updateCache
		a.updateMu.Unlock()
		return out
	}
	a.updateMu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	res := updater.CheckForUpdate(ctx, updater.Config{CurrentVersion: Version})

	a.updateMu.Lock()
	a.updateCache = res
	a.updateCacheTTL = time.Now().Add(24 * time.Hour)
	a.updateMu.Unlock()

	a.logger.Info("update check",
		"current", res.CurrentVersion,
		"latest", res.LatestVersion,
		"available", res.Available,
		"err", res.Error,
	)
	return res
}

// OpenReleasePage opens the user's browser at the latest release page
// (or the repo's releases overview as a fallback). Returns nothing —
// the OS handles it.
func (a *App) OpenReleasePage(url string) {
	if url == "" {
		url = "https://github.com/Yaxyobek0877/portal_traffic/releases/latest"
	}
	runtime.BrowserOpenURL(a.ctx, url)
}

// CrashReports lists all locally-captured crash reports, newest first.
// Reports live in ~/.portal/crashes/ and contain only non-PII data
// (panic message, stack with $HOME redacted, OS, version).
func (a *App) CrashReports() []crashreport.Report {
	if a.crashCatcher == nil {
		return []crashreport.Report{}
	}
	reports, err := a.crashCatcher.List()
	if err != nil {
		a.logger.Warn("crashreport list failed", "err", err)
		return []crashreport.Report{}
	}
	if reports == nil {
		return []crashreport.Report{}
	}
	return reports
}

// OpenCrashFolder opens the OS file manager at the crashes directory.
func (a *App) OpenCrashFolder() error {
	if a.crashCatcher == nil {
		return errors.New("crash catcher unavailable")
	}
	dir := a.crashCatcher.Dir()
	_ = os.MkdirAll(dir, 0o700)
	runtime.BrowserOpenURL(a.ctx, "file://"+dir)
	return nil
}

// ClearCrashReports deletes all crash reports. UI is responsible for
// the confirmation prompt.
func (a *App) ClearCrashReports() error {
	if a.crashCatcher == nil {
		return nil
	}
	return a.crashCatcher.Clear()
}

// LogLines returns the last `n` lines of the in-memory log ring.
// Used by Settings → Diagnostika to surface what the app has been
// doing without having to dig into the on-disk file.
func (a *App) LogLines(n int) []string {
	return LogTail(n)
}

// LogFilePath returns the absolute on-disk path of the current log
// file (~/.portal/logs/portal-YYYY-MM-DD.log).
func (a *App) LogFilePath() string {
	return LogPath()
}

// OpenLogFolder opens the OS file manager at the logs directory.
func (a *App) OpenLogFolder() error {
	p := LogPath()
	if p == "" {
		return errors.New("log fayli yo'q")
	}
	dir := filepath.Dir(p)
	runtime.BrowserOpenURL(a.ctx, "file://"+dir)
	return nil
}

// ClearLogs empties the in-memory tail. The on-disk file remains so
// you can dig deeper if needed.
func (a *App) ClearLogs() {
	ClearLogTail()
}

// LocalListeners enumerates TCP and UDP ports the OS reports as
// listening. Used by the Services panel to offer one-click "expose"
// for services already running on the user's machine — including UDP
// game servers (CS2 27015, Minecraft Bedrock 19132) which a TCP-only
// sweep silently misses.
//
// Implementation: shell out per protocol — `lsof -iTCP -sTCP:LISTEN`
// + `lsof -iUDP` (macOS), or `ss -lntp` + `ss -lnup` (Linux). Errors
// are absorbed so the UI stays usable on platforms where neither is
// available (Windows, sandboxed environments).
// RiskAssessment classifies a planned (target, port) before exposing
// so the UI can prompt for confirmation on the dangerous ones.
type RiskAssessment struct {
	Level   string `json:"level"`   // "safe" | "warn" | "danger"
	Reason  string `json:"reason"`  // human-readable Uzbek
	Hint    string `json:"hint"`    // suggested mitigation
}

// AssessExposeRisk returns a non-empty Reason whenever the (target,
// port, protocol) combination smells dangerous — DB ports, router
// admin web UI, RDP, anything where a stranger landing on the
// service via a leaked portal code could cause real damage.
//
// Heuristic-driven: false positives are fine ("warn" doesn't block,
// it just nudges); false negatives are the real cost. When in doubt,
// flag.
func (a *App) AssessExposeRisk(target string, protocol string, port int) RiskAssessment {
	target = strings.TrimSpace(target)
	host := ""
	if target != "" {
		if h, _, err := net.SplitHostPort(target); err == nil {
			host = h
		}
	}

	// Effective port the proxy will dial — when target is set, that's
	// the port part of target; otherwise the mesh-side port (which
	// also doubles as the localhost target in default mode).
	effPort := port
	if target != "" {
		if _, p, err := net.SplitHostPort(target); err == nil {
			if n, err := strconv.Atoi(p); err == nil {
				effPort = n
			}
		}
	}

	// Database / cache / message-queue ports — almost always weakly
	// authenticated, often plaintext. Don't share with strangers.
	dbPorts := map[int]string{
		3306:  "MySQL/MariaDB",
		5432:  "PostgreSQL",
		27017: "MongoDB",
		6379:  "Redis",
		11211: "Memcached",
		9200:  "Elasticsearch",
		5984:  "CouchDB",
		7474:  "Neo4j",
		8086:  "InfluxDB",
		2181:  "ZooKeeper",
		9092:  "Kafka",
	}
	if name, ok := dbPorts[effPort]; ok {
		return RiskAssessment{
			Level:  "danger",
			Reason: name + " (port " + strconv.Itoa(effPort) + ") — ma'lumotlar bazasi sukut bo'yicha himoyasiz",
			Hint:   "Bu portni mesh'ga ochish — ma'lumot o'g'irlash xatosi. Faqat zarur bo'lsa, kuchli parol va shifrlangan ulanish bilan.",
		}
	}

	// Remote-administration ports.
	adminPorts := map[int]string{
		3389: "RDP — masofadan ish stoli",
		5900: "VNC — ekran almashinuvi",
		22:   "SSH",
	}
	if name, ok := adminPorts[effPort]; ok {
		return RiskAssessment{
			Level:  "warn",
			Reason: name + " — masofadan to'liq boshqaruv beradi",
			Hint:   "Faqat ishonchli kishilarga ochib bering. Strong key/parol shart.",
		}
	}

	// Router admin web UI — common gateway IPs.
	gatewayIPs := map[string]bool{
		"192.168.0.1": true, "192.168.1.1": true, "192.168.1.254": true,
		"10.0.0.1": true, "10.0.0.138": true, "10.1.1.1": true,
		"192.168.100.1": true, "192.168.2.1": true, "172.16.0.1": true,
	}
	if host != "" && gatewayIPs[host] && (effPort == 80 || effPort == 443 || effPort == 8080) {
		return RiskAssessment{
			Level:  "danger",
			Reason: "Router admin paneli (" + target + ")",
			Hint:   "Mehmon zaif parol bilan kirsa, butun tarmog'ingizni boshqaradi. Default parolni o'zgartiring va birinchi bo'lib parolni almashtiring.",
		}
	}

	// Plain HTTP web UIs on common admin ports — warn but don't block.
	adminWebPorts := map[int]bool{8080: true, 8443: true, 8000: true, 8123: true, 5000: true, 9090: true}
	if adminWebPorts[effPort] && protocol == "tcp" {
		return RiskAssessment{
			Level:  "warn",
			Reason: "Admin web UI sifatida ko'p ishlatiladigan port (" + strconv.Itoa(effPort) + ")",
			Hint:   "Servis kuchli autentifikatsiyaga egami tekshiring.",
		}
	}

	return RiskAssessment{Level: "safe"}
}

// ActivityEntry mirrors proxy.ActivityEntry for the JSON wire to the
// frontend.
type ActivityEntry struct {
	Time     time.Time `json:"time"`
	PeerID   string    `json:"peerId"`
	Nickname string    `json:"nickname,omitempty"`
	Protocol string    `json:"protocol"`
	Port     int       `json:"port"`
	Target   string    `json:"target"`
	Result   string    `json:"result"`
}

// ProxyActivity returns recent peer-dial events on services we host
// across every active session — Settings → Faollik should show
// activity in any portal you're connected to, not just the one
// currently foregrounded. Newest entries last, sorted by time.
func (a *App) ProxyActivity() []ActivityEntry {
	sessions := a.allSessions()
	if len(sessions) == 0 {
		return []ActivityEntry{}
	}
	// Build one shared nickname map by walking every session's peers.
	// Same peer-id can't appear in two sessions at once (the server
	// minted it for one portal), so the merge is collision-free.
	nicks := map[string]string{}
	for _, s := range sessions {
		if s.mesh == nil {
			continue
		}
		for _, p := range s.mesh.Peers() {
			nicks[p.ID] = p.Nickname
		}
	}
	var out []ActivityEntry
	for _, s := range sessions {
		if s.fwd == nil {
			continue
		}
		for _, e := range s.fwd.Activity() {
			out = append(out, ActivityEntry{
				Time: e.Time, PeerID: e.PeerID, Nickname: nicks[e.PeerID],
				Protocol: e.Protocol, Port: e.Port, Target: e.Target,
				Result: e.Result,
			})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Time.Before(out[j].Time) })
	if out == nil {
		out = []ActivityEntry{}
	}
	return out
}

// ScanLAN does a quick TCP probe across the host's local /24 on a
// short list of well-known service ports (RTSP cameras, HTTP web
// UIs, network printers, etc.) so the user can one-click expose LAN
// devices without typing IP:port by hand. ~5–10 seconds for a typical
// home network. Emits "lanscan:progress" events while running so the
// UI can paint a "scanning 192.168.1.123…" line instead of a blank.
func (a *App) ScanLAN() []lanscan.Discovery {
	ctx, cancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer cancel()
	out := lanscan.ScanWithProgress(ctx, func(p lanscan.Progress) {
		if a.ctx == nil {
			return
		}
		runtime.EventsEmit(a.ctx, "lanscan:progress", p)
	})
	a.logger.Info("lan scan complete", "found", len(out))
	if out == nil {
		// JSON-marshal nil slice as [] not null — UI iterates over it.
		out = []lanscan.Discovery{}
	}
	return out
}

func (a *App) LocalListeners() []LocalListener {
	out := []LocalListener{}
	probes := localListenerProbes()
	if len(probes) == 0 {
		return out
	}
	var all []LocalListener
	for _, p := range probes {
		stdout, err := p.cmd.Output()
		if err != nil {
			a.logger.Debug("local listeners enumerate failed",
				"protocol", p.protocol, "err", err)
			continue
		}
		all = append(all, p.parser(string(stdout), p.protocol)...)
	}
	// Filter out the noise: ports < 1024 are mostly system services
	// (most users won't want to expose mDNS, AirPlay, etc.); also
	// drop the well-known infra (cloudflared metrics, our own
	// signaling, our embedded webview). The user can still type any
	// port manually.
	systemNoise := map[string]bool{
		"ControlCe": true, "rapportd": true, "cloudflar": true,
		"identitys": true, "sharingd": true,
		"Portal": true, "portal-si": true, "portal-cl": true,
	}
	// Dedup key is (protocol, port) — the same port can legitimately
	// have a TCP and a UDP listener at once (Steam dedicated servers
	// are a classic example).
	type key struct {
		proto string
		port  int
	}
	seen := map[key]bool{}
	for _, r := range all {
		if r.Port < 1024 || systemNoise[r.Process] {
			continue
		}
		k := key{r.Protocol, r.Port}
		if seen[k] {
			continue
		}
		seen[k] = true
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Port != out[j].Port {
			return out[i].Port < out[j].Port
		}
		return out[i].Protocol < out[j].Protocol
	})
	return out
}

// LocalServices returns the services we are currently exposing.
//
// We treat exposes as global — a service the user opens via the UI
// is announced into every active session's mesh, so peers in any
// portal can dial it. The list returned here therefore reflects the
// persisted exposed_services rows (the source of truth) decorated
// with health from the background prober. The active session's mesh
// snapshot is used as a sanity-check that a row really is currently
// announced; differences are extremely rare (announce ack vs
// persisted row) but the UI handles them gracefully.
func (a *App) LocalServices() []ServiceView {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()

	// Targets come from any session's forwarder — they all share the
	// same target table because ExposeService walks every session.
	// Pick the first non-nil one; fall back to empty.
	var targets map[int]string
	for _, s := range a.allSessions() {
		if s.fwd != nil {
			targets = s.fwd.ExposedSnapshot()
			break
		}
	}
	if targets == nil {
		targets = map[int]string{}
	}

	a.healthMu.Lock()
	healthCopy := make(map[serviceHealthKey]serviceHealth, len(a.exposedHealth))
	for k, v := range a.exposedHealth {
		healthCopy[k] = v
	}
	a.healthMu.Unlock()

	out := []ServiceView{}
	if store == nil {
		return out
	}
	saved, _ := store.ListExposedServices()
	for _, s := range saved {
		v := ServiceView{
			Name:     s.Name,
			Protocol: s.Protocol,
			Port:     s.Port,
		}
		if t, ok := targets[s.Port]; ok {
			v.Target = t
		} else {
			v.Target = s.Target
		}
		if !s.Enabled {
			v.Paused = true
		} else if h, ok := healthCopy[serviceHealthKey{s.Port, s.Protocol}]; ok {
			v.Health = h.status
			v.HealthError = h.err
		} else {
			v.Health = "unknown"
		}
		out = append(out, v)
	}
	return out
}

// runHealthLoop periodically TCP-probes every exposed target and
// stores the result in exposedHealth so LocalServices can hand
// the UI an up-to-date "camera reachable / camera offline"
// indicator. Cheap (one connect per service every 30s) and
// completely independent of peer activity.
func (a *App) runHealthLoop() {
	t := time.NewTicker(30 * time.Second)
	defer t.Stop()
	// Probe immediately on startup so the first user view isn't
	// stuck on "unknown" for 30 seconds.
	a.runHealthOnce()
	for {
		select {
		case <-t.C:
			a.runHealthOnce()
		}
	}
}

func (a *App) runHealthOnce() {
	// We need both the host:port target (lives on the forwarder) and
	// the protocol (lives in storage rows). Walk both: forwarder
	// snapshot tells us "is this port currently announced", storage
	// tells us "tcp or udp" so the probe can pick the right strategy.
	var f *proxy.Forwarder
	for _, s := range a.allSessions() {
		if s.fwd != nil {
			f = s.fwd
			break
		}
	}
	if f == nil {
		return
	}
	targets := f.ExposedSnapshot()
	if len(targets) == 0 {
		return
	}

	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return
	}
	saved, err := store.ListExposedServices()
	if err != nil {
		a.logger.Debug("health: list exposed", "err", err)
		return
	}

	results := make(map[serviceHealthKey]serviceHealth, len(saved))
	for _, svc := range saved {
		if !svc.Enabled {
			continue
		}
		target, ok := targets[svc.Port]
		if !ok {
			continue
		}
		key := serviceHealthKey{port: svc.Port, protocol: svc.Protocol}
		h := serviceHealth{target: target, checkedAt: time.Now()}
		switch svc.Protocol {
		case "tcp":
			conn, err := net.DialTimeout("tcp", target, 2*time.Second)
			if err != nil {
				h.status = "down"
				h.err = friendlyHealthError(target, err)
			} else {
				h.status = "ok"
				_ = conn.Close()
			}
		case "udp":
			// UDP can't be probed without sending protocol-specific
			// traffic — a "connect" succeeds even if nothing is
			// listening (no SYN/ACK handshake), so we'd false-positive
			// 'ok' on every UDP target. Mark as unknown and let the
			// UI surface a neutral indicator.
			h.status = "unknown"
			h.err = "UDP holatini avtomatik tekshirib bo'lmaydi — peer ulanganda aniqlanadi"
		default:
			h.status = "unknown"
		}
		results[key] = h
	}

	a.healthMu.Lock()
	if a.exposedHealth == nil {
		a.exposedHealth = map[serviceHealthKey]serviceHealth{}
	}
	for k, v := range results {
		a.exposedHealth[k] = v
	}
	// Drop entries for (port, protocol) tuples no longer exposed so
	// stale "down" indicators don't stick around after Unexpose.
	for k := range a.exposedHealth {
		if _, ok := results[k]; !ok {
			delete(a.exposedHealth, k)
		}
	}
	a.healthMu.Unlock()
}

// friendlyHealthError translates a raw net error into something
// localised + actionable for the UI. Generic errors (refused, no
// route, timeout) get the most-likely-cause hint appended; anything
// we don't recognise falls through to the original string so we
// don't hide useful diagnostic info from a power user.
func friendlyHealthError(target string, err error) string {
	msg := err.Error()
	low := strings.ToLower(msg)
	switch {
	case strings.Contains(low, "no route to host"),
		strings.Contains(low, "network is unreachable"):
		return target + " — qurilma o'chiq yoki tarmoqda yo'q (IP'ni tekshiring)"
	case strings.Contains(low, "connection refused"):
		return target + " — qurilma yoqilgan, lekin shu portda servis yo'q (port to'g'rimi?)"
	case strings.Contains(low, "i/o timeout"),
		strings.Contains(low, "deadline exceeded"):
		return target + " — javob bermoqda emas (qurilma yoqilganmi?)"
	case strings.Contains(low, "no such host"):
		return target + " — DNS topa olmadi (host nomi to'g'rimi?)"
	}
	return msg
}

// ExposeService registers a port and announces it to peers.
//   - protocol: "tcp" or "udp"; "" defaults to "tcp"
//   - port:     the mesh-side port other peers will dial
//   - target:   "host:port" the proxy connects to when peers open the
//               stream. "" → 127.0.0.1:<port> (the local service).
//               Use a non-loopback target to forward through the
//               mesh to a LAN device, e.g. "192.168.1.100:554" for
//               an RTSP camera or NVR.
//
// UDP is required for game traffic — CS2 / Valorant run their
// tickrate on UDP, exposing them as TCP-only would silently fail at
// dial time.
func (a *App) ExposeService(name string, protocol string, port int, target string) error {
	if protocol == "" {
		protocol = "tcp"
	}
	if protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf("protocol noma'lum: %q (tcp yoki udp)", protocol)
	}
	target = strings.TrimSpace(target)
	if target != "" {
		// Validate "host:port" shape; reject anything else so a typo
		// doesn't silently route to a bizarre destination.
		if _, _, err := net.SplitHostPort(target); err != nil {
			return fmt.Errorf("target noma'lum (host:port kerak): %w", err)
		}
	}
	if name == "" {
		name = fmt.Sprintf("%s:%d", protocol, port)
	}

	// Persist first so a missing-portal error doesn't lose the user's
	// intent. Best-effort.
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store != nil {
		if err := store.SaveExposedService(storage.ExposedService{
			Port: port, Protocol: protocol, Name: name, Target: target, Enabled: true,
		}); err != nil {
			a.logger.Debug("save exposed service", "err", err)
		}
	}

	// Broadcast to every active session — a service the user wants
	// shared should reach peers in any portal they're connected to.
	// If there's no active session yet (user pre-exposed before
	// dialing) we still keep the persisted row, and bringUpSession
	// will replay it via restoreExposedServicesFor.
	sessions := a.allSessions()
	if len(sessions) == 0 {
		a.logger.Info("expose service (no portal yet — persisted only)",
			"name", name, "protocol", protocol, "port", port, "target", target)
		return nil
	}
	var firstErr error
	for _, s := range sessions {
		if s.fwd == nil || s.mesh == nil {
			continue
		}
		s.fwd.ExposeTarget(port, target)
		if err := s.mesh.AnnounceService(name, protocol, port); err != nil {
			a.logger.Warn("expose: announce failed",
				"session", s.localID, "port", port, "err", err)
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	a.logger.Info("expose service",
		"name", name, "protocol", protocol, "port", port,
		"target", target, "sessions", len(sessions),
	)
	return firstErr
}

// SetExposeEnabled toggles a service between "actively shared" and
// "remembered but paused" without losing its LAN-target / name. Pause
// removes it from the running mesh + storage's enabled flag flips
// off; resume re-applies the persisted entry to the running mesh.
//
// Lets users park their cameras / NVRs without re-typing IPs every
// time they want to stop sharing for a bit.
func (a *App) SetExposeEnabled(port int, protocol string, enabled bool) error {
	if protocol == "" {
		protocol = "tcp"
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage mavjud emas — pauza saqlanmaydi")
	}

	saved, err := store.ListExposedServices()
	if err != nil {
		return err
	}
	var match *storage.ExposedService
	for i := range saved {
		if saved[i].Port == port && saved[i].Protocol == protocol {
			match = &saved[i]
			break
		}
	}
	if match == nil {
		return fmt.Errorf("servis topilmadi: %s:%d", protocol, port)
	}
	match.Enabled = enabled
	if err := store.SaveExposedService(*match); err != nil {
		return err
	}

	sessions := a.allSessions()
	if !enabled {
		// Pause: stop sharing now but keep the row. Apply to every
		// session so the pause is global.
		for _, s := range sessions {
			if s.fwd != nil {
				s.fwd.Unexpose(port)
			}
			if s.mesh != nil {
				_ = s.mesh.UnannounceService(port)
			}
		}
		a.logger.Info("expose paused", "port", port, "protocol", protocol)
		return nil
	}
	// Resume: re-apply to every running mesh.
	var firstErr error
	for _, s := range sessions {
		if s.fwd == nil || s.mesh == nil {
			continue
		}
		s.fwd.ExposeTarget(port, match.Target)
		if err := s.mesh.AnnounceService(match.Name, match.Protocol, match.Port); err != nil {
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	a.logger.Info("expose resumed",
		"port", port, "protocol", protocol, "target", match.Target,
		"sessions", len(sessions))
	return firstErr
}

// UnexposeService removes a previously exposed port. Also wipes the
// persisted preference for that port so it doesn't auto-restore on
// the next session — if the user clicks the trash icon they mean it.
// Removes from every active session.
func (a *App) UnexposeService(port int) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	for _, s := range a.allSessions() {
		if s.fwd != nil {
			s.fwd.Unexpose(port)
		}
		if s.mesh != nil {
			_ = s.mesh.UnannounceService(port)
		}
	}
	if store != nil {
		// Delete both tcp and udp rows — the front-end models it as
		// "one entry per port" and we never want a stale persisted
		// row resurrecting an unwanted share.
		_ = store.DeleteExposedService(port, "tcp")
		_ = store.DeleteExposedService(port, "udp")
	}
	return nil
}

// restoreExposedServicesFor reapplies every saved-and-enabled
// ExposedService row to a single session's mesh + forwarder. Called
// from relayEvent on EventPortalReady so each fresh portal picks up
// the user's exposure set without them having to re-Och anything.
func (a *App) restoreExposedServicesFor(s *portalSession) {
	if s == nil || s.fwd == nil || s.mesh == nil {
		return
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return
	}
	saved, err := store.ListExposedServices()
	if err != nil {
		a.logger.Debug("restore exposed: list", "err", err)
		return
	}
	for _, svc := range saved {
		if !svc.Enabled {
			continue
		}
		s.fwd.ExposeTarget(svc.Port, svc.Target)
		if err := s.mesh.AnnounceService(svc.Name, svc.Protocol, svc.Port); err != nil {
			a.logger.Warn("restore exposed: re-announce failed",
				"session", s.localID, "name", svc.Name, "port", svc.Port, "err", err)
			continue
		}
		a.logger.Info("restore exposed: ok",
			"session", s.localID, "name", svc.Name, "protocol", svc.Protocol,
			"port", svc.Port, "target", svc.Target)
	}
}

// DialService opens a local listener (TCP or UDP) that pumps to the
// peer's exposed remotePort. protocol "" defaults to "tcp" so older
// callers still work. localPort 0 means "match remotePort if free,
// otherwise let the OS pick" — the local alias mirrors the remote
// address (127.0.0.1:27015 for a remote :27015) which is what users
// expect when typing it into a game's connect dialog.
//
// Returns the resolved local addr ("127.0.0.1:27015") so the UI
// shows it.
func (a *App) DialService(peerID string, protocol string, remotePort, localPort int) (string, error) {
	s := a.sessionWithPeer(peerID)
	if s == nil || s.fwd == nil {
		return "", errors.New("portal yo'q")
	}
	f := s.fwd
	if protocol == "" {
		protocol = "tcp"
	}
	switch protocol {
	case "tcp":
		if localPort == 0 {
			ln, err := f.DialPreferringPort(a.ctx, peerID, remotePort)
			if err != nil {
				return "", err
			}
			return ln.Addr().String(), nil
		}
		addr := fmt.Sprintf("127.0.0.1:%d", localPort)
		ln, err := f.Dial(a.ctx, peerID, remotePort, addr)
		if err != nil {
			return "", err
		}
		return ln.Addr().String(), nil
	case "udp":
		if localPort == 0 {
			conn, err := f.DialUDPPreferringPort(a.ctx, peerID, remotePort)
			if err != nil {
				return "", err
			}
			return conn.LocalAddr().String(), nil
		}
		addr := fmt.Sprintf("127.0.0.1:%d", localPort)
		conn, err := f.DialUDP(a.ctx, peerID, remotePort, addr)
		if err != nil {
			return "", err
		}
		return conn.LocalAddr().String(), nil
	default:
		return "", fmt.Errorf("protocol noma'lum: %q (tcp yoki udp)", protocol)
	}
}

// ----------------------------------------------------------------------------
// Internals
// ----------------------------------------------------------------------------

// bringUpMesh allocates a fresh mesh.Manager + proxy.Forwarder and
// starts the event-pump goroutine. Idempotent: tearing down first if
// a previous session existed.
// bringUpSession spins up a fresh portalSession with its own
// mesh.Manager / proxy.Forwarder / transfer.Engine. The new session
// is registered under its localID immediately so concurrent reads
// (e.g. another ExposeService call) can find it. If makeActive is
// true, it also becomes the foreground (App.activeID); otherwise the
// session runs in the background and the UI stays where it is.
//
// Returns the session pointer for the caller to drive
// CreatePortal / JoinPortal on its mesh.
func (a *App) bringUpSession(nickname string, makeActive bool, isOwner bool) (*portalSession, error) {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return nil, errors.New("taxallus bo'sh bo'lmasligi kerak")
	}

	url := a.SignalingURL()
	turn := a.GetTurnConfig()
	// Cloudflare TURN: kept for backwards-compat with existing user
	// installs that have a token configured. New users only get the
	// signaling-server-issued credentials applied via
	// mesh.applyServerICE on PortalReady; the UI no longer surfaces
	// the form so this falls through to nil for them.
	cfICE := a.resolveICEServers()

	s := &portalSession{
		localID:    newSessionID(),
		nickname:   nickname,
		isOwner:    isOwner,
		background: !makeActive,
		state:      "connecting",
		pumpDone:   make(chan struct{}),
	}
	s.mesh = mesh.New(mesh.Config{
		SignalingURL:      url,
		Nickname:          nickname,
		HeartbeatInterval: 3 * time.Second,
		Logger:            a.logger.With("session", s.localID),
		ICEServers:        cfICE, // nil → mesh.DefaultICEServers
		TurnURL:           turn.URL,
		TurnUsername:      turn.Username,
		TurnCredential:    turn.Credential,
		// CloudAuthToken: when the user is signed into a server account,
		// the mesh adds ?token=... on its WebSocket connect so the
		// signaling server stamps this portal's owner with the user
		// ID, making the dashboard list it across every device the
		// same account is signed into.
		CloudAuthToken: a.CloudAuthToken(),
	})
	s.fwd = proxy.New(s.mesh, a.logger.With("session", s.localID))
	s.mesh.SetProxyHandler(s.fwd)
	// Plug the approval gate. The Forwarder calls back into App's
	// Approve(...) for each peer dial; auto-allow ports return true
	// immediately, gated ports emit an event to the UI and block on
	// the Approve / DenyApproval RPC reply.
	s.fwd.SetApprover(approverFunc(a.approvePeerOpen))

	// Transfer engine — per-session so a download in portal A doesn't
	// collide with one in portal B (xfer ids are local to each engine).
	ctx := a.ctx
	sid := s.localID
	s.xfer = transfer.NewEngine(s.mesh, "", a.logger.With("session", sid),
		func(ev transfer.ProgressEvent) {
			if ctx == nil {
				return
			}
			// Tag transfer events with the session id so a UI showing
			// portal A doesn't mistakenly render progress that belongs
			// to portal B.
			runtime.EventsEmit(ctx, "transfer:progress",
				progressWithSession(ev, sid))
		})
	s.mesh.SetTransferHandler(s.xfer)

	a.sessionsMu.Lock()
	a.sessions[s.localID] = s
	if makeActive {
		a.activeID = s.localID
	}
	a.sessionsMu.Unlock()

	go a.pumpSession(s)
	return s, nil
}

// closeSession tears down a single session: leaves the mesh, closes
// the forwarder, and waits briefly for the event-pump goroutine to
// drain. Idempotent — calling it twice on the same session is a
// no-op for the second call.
func (a *App) closeSession(s *portalSession) {
	if s == nil {
		return
	}
	if s.fwd != nil {
		_ = s.fwd.Close()
	}
	if s.mesh != nil {
		_ = s.mesh.Leave()
		s.mesh.Close()
	}
	// Pump goroutine ends when mesh.Done is closed; wait briefly so
	// callers know it's drained before we drop the session pointer.
	if s.pumpDone != nil {
		select {
		case <-s.pumpDone:
		case <-time.After(2 * time.Second):
		}
	}
}

// activeSession returns the foreground session, or nil if none.
// Read-only callers should grab this once and check for nil.
func (a *App) activeSession() *portalSession {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()
	if a.activeID == "" {
		return nil
	}
	return a.sessions[a.activeID]
}

// sessionByLocalID looks up a session by its localID. Returns nil if
// not found. Used by the new multi-portal API methods.
func (a *App) sessionByLocalID(id string) *portalSession {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()
	return a.sessions[id]
}

// allSessions returns a slice copy of every live session. Caller can
// iterate without holding the lock, which matters for ExposeService /
// UnexposeService where we want to call into each session's mesh
// without serialising on the global session lock.
func (a *App) allSessions() []*portalSession {
	a.sessionsMu.RLock()
	defer a.sessionsMu.RUnlock()
	out := make([]*portalSession, 0, len(a.sessions))
	for _, s := range a.sessions {
		out = append(out, s)
	}
	return out
}

func (a *App) nickname() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.nick
}

// waitPortalReady blocks until the given session's mesh emits
// PortalReady, surfaces an error, or the deadline expires. The
// session's own mu carries the eventual PortalView (set by pumpSession
// on EventPortalReady), so we poll it on a short tick rather than
// duplicating the mesh-event-channel plumbing here.
func (a *App) waitPortalReady(s *portalSession, timeout time.Duration) (PortalView, error) {
	if s == nil || s.mesh == nil {
		return PortalView{}, errors.New("mesh tear down")
	}
	deadline := time.After(timeout)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if pi := s.mesh.Portal(); pi != nil {
			return portalToView(pi), nil
		}
		select {
		case err := <-s.mesh.InitialError():
			return PortalView{}, err
		case <-deadline:
			return PortalView{}, errors.New("portal javobi kelmadi (timeout)")
		case <-tick.C:
		}
	}
}

// pumpSession fans events from one mesh.Manager out to the frontend,
// tagging each with the session's localID + portalID so the JS side
// can route them to the right portal in its store. Runs until the
// mesh closes; closes pumpDone on exit so closeSession can wait on it.
func (a *App) pumpSession(s *portalSession) {
	defer close(s.pumpDone)
	if s.mesh == nil {
		return
	}
	for {
		select {
		case <-s.mesh.Done():
			return
		case ev, ok := <-s.mesh.Events():
			if !ok {
				return
			}
			a.relayEvent(s, ev)
		}
	}
}

func (a *App) relayEvent(s *portalSession, ev mesh.MeshEvent) {
	if a.ctx == nil {
		return
	}
	sid := s.localID
	switch ev.Type {
	case mesh.EventPortalReady:
		view := portalToView(ev.Portal)
		// Owner sessions know their code from the server; joiner
		// sessions don't (the server doesn't echo it back). For
		// joiners we patched the code into PortalView at JoinPortal
		// time before persistEnter; the live mesh.Portal doesn't
		// carry it, so prefer the cached view.code if non-empty.
		s.mu.Lock()
		if s.view.Code != "" {
			view.Code = s.view.Code
		}
		s.view = view
		s.state = "connected"
		s.err = nil
		s.mu.Unlock()
		runtime.EventsEmit(a.ctx, "portal:ready", portalReadyEvent(s, view))
		// Reapply persisted exposed services to the new mesh. Per
		// session — we want the same set of shares to surface in
		// every active portal. Fire-and-forget; failure shouldn't
		// block the event loop.
		go a.restoreExposedServicesFor(s)

	case mesh.EventPeerJoining:
		runtime.EventsEmit(a.ctx, "peer:joining", peerEvent(s, ev.Peer, "connecting"))

	case mesh.EventPeerReady:
		runtime.EventsEmit(a.ctx, "peer:ready", peerEvent(s, ev.Peer, "connected"))

	case mesh.EventPeerRTT:
		runtime.EventsEmit(a.ctx, "peer:rtt", peerEvent(s, ev.Peer, ""))

	case mesh.EventPeerTransport:
		runtime.EventsEmit(a.ctx, "peer:transport", peerEvent(s, ev.Peer, ""))

	case mesh.EventPeerLeft:
		runtime.EventsEmit(a.ctx, "peer:left", peerEvent(s, ev.Peer, "closed"))

	case mesh.EventPortalClosed:
		s.setState("closed", nil)
		// Drop the session from the map so the dashboard's "active
		// portals" strip stops showing it. Note: do NOT close the
		// session here — the mesh has already torn itself down. We
		// just need to forget it.
		a.sessionsMu.Lock()
		delete(a.sessions, sid)
		if a.activeID == sid {
			a.activeID = ""
		}
		a.sessionsMu.Unlock()
		// Server-initiated close means the portal is gone for good
		// (owner left / portal expired). Drop the active_sessions
		// row so the next-launch resume doesn't waste cycles re-
		// dialing a dead portal_id and seeing 'no such portal' from
		// the server.
		a.forgetActiveSession(s.snapshotPortalID())
		runtime.EventsEmit(a.ctx, "portal:closed", map[string]string{"sessionId": sid})

	case mesh.EventChat:
		runtime.EventsEmit(a.ctx, "chat", ChatMessage{
			SessionID: sid,
			From:      ev.Peer.ID,
			Nickname:  ev.Peer.Nickname,
			Text:      ev.ChatText,
			At:        time.Now(),
			IsLocal:   false,
		})

	case mesh.EventServiceAnnounce:
		runtime.EventsEmit(a.ctx, "peer:services", peerEvent(s, ev.Peer, ""))

	case mesh.EventError:
		msg := ""
		if ev.Err != nil {
			msg = ev.Err.Error()
		}
		runtime.EventsEmit(a.ctx, "error",
			map[string]string{"sessionId": sid, "message": msg})
	}
}

// peerEvent builds the JS payload for peer:* events, tagging it with
// the session id so the frontend can route into its per-session
// peer map without ambiguity.
func peerEvent(s *portalSession, p *mesh.Peer, override string) map[string]any {
	pv := peerToView(p)
	if override != "" {
		pv.State = override
	}
	pv.SessionID = s.localID
	return map[string]any{
		"sessionId": s.localID,
		"portalId":  s.snapshotPortalID(),
		"peer":      pv,
	}
}

// portalReadyEvent is the per-session payload for "portal:ready".
// Older callers who subscribed expecting just the PortalView still
// get its fields at the top level; the sessionId is added for the
// new multi-portal-aware UI.
func portalReadyEvent(s *portalSession, v PortalView) map[string]any {
	return map[string]any{
		"sessionId":   s.localID,
		"portalId":    v.PortalID,
		"code":        v.Code,
		"ownerId":     v.OwnerID,
		"ownPeerId":   v.OwnPeerID,
		"ownVip":      v.OwnVIP,
		"isOwner":     v.IsOwner,
		"nickname":    s.nickname,
		"background":  s.background,
	}
}

// snapshotPortalID returns the server-assigned portal ID, or "" if
// PortalReady hasn't fired yet.
func (s *portalSession) snapshotPortalID() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.view.PortalID
}

// progressWithSession augments a transfer.ProgressEvent with the
// session it belongs to. We re-encode as a generic map because
// transfer.ProgressEvent is a struct in another package and we
// can't add fields to it without invasive changes.
func progressWithSession(ev transfer.ProgressEvent, sid string) map[string]any {
	return map[string]any{
		"sessionId": sid,
		"xferId":    ev.XferID,
		"peerId":    ev.PeerID,
		"direction": ev.Direction,
		"manifest":  ev.Manifest,
		"bytes":     ev.Bytes,
		"total":     ev.Total,
		"done":      ev.Done,
		"error":     ev.Error,
		"startedAt": ev.StartedAt,
		"updatedAt": ev.UpdatedAt,
		"savePath":  ev.SavePath,
	}
}

// ----------------------------------------------------------------------------
// Per-port approval gate
// ----------------------------------------------------------------------------

// approverFunc adapts a function literal to the proxy.Approver
// interface so we don't have to declare a separate struct just to
// hold one method.
type approverFunc func(ctx context.Context, peerID, protocol string, port int) bool

func (f approverFunc) Approve(ctx context.Context, peerID, protocol string, port int) bool {
	return f(ctx, peerID, protocol, port)
}

// approvePeerOpen is the central decision point for "should this
// peer be allowed to open this exposed port?". The Forwarder calls
// it from onOpenTCP / onOpenUDP after the port has been verified as
// exposed. Returning true → dial proceeds; false → frameOpenErr.
//
// Decision tree:
//
//   1. Look up the persisted exposed_services row. If require_approval
//      is false (auto-allow), return true immediately. This is the
//      historical behaviour and the one most users expect.
//   2. If we've already approved or denied this exact (peer, port,
//      protocol) tuple this session, replay the cached decision.
//      Avoids re-prompting the user for the same camera dial five
//      seconds after they already clicked Allow.
//   3. Otherwise emit a `service:approval-request` event to the UI
//      with a unique requestID, then block on a channel until the
//      frontend calls ApproveServiceRequest / DenyServiceRequest.
//      Context deadline (5s TCP / 3s UDP) → deny so the peer's
//      open call doesn't hang forever.
func (a *App) approvePeerOpen(ctx context.Context, peerID, protocol string, port int) bool {
	// Step 1: storage lookup.
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		// Without storage we can't tell whether the port is gated;
		// fail open (auto-allow) so the lack of persistence doesn't
		// break a working session.
		return true
	}
	saved, err := store.ListExposedServices()
	if err != nil {
		a.logger.Warn("approval: list exposed services failed", "err", err)
		return true
	}
	gated := false
	for _, svc := range saved {
		if svc.Port == port && svc.Protocol == protocol {
			gated = svc.RequireApproval
			break
		}
	}
	if !gated {
		return true
	}

	// Step 2: cached decision.
	key := approvalKey{peerID: peerID, port: port, protocol: protocol}
	a.approvalMu.Lock()
	if decided, ok := a.approvalDecided[key]; ok {
		a.approvalMu.Unlock()
		return decided
	}

	// Step 3: ask the UI.
	requestID := newApprovalID()
	wait := make(chan bool, 1)
	a.approvalPending[requestID] = wait
	a.approvalMu.Unlock()

	// Look up peer nickname for the prompt — falls back to the raw id.
	nickname := a.nicknameForPeer(peerID)
	serviceName := a.serviceNameFor(saved, port, protocol)

	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "service:approval-request", map[string]any{
			"requestId":   requestID,
			"peerId":      peerID,
			"nickname":    nickname,
			"protocol":    protocol,
			"port":        port,
			"serviceName": serviceName,
		})
	}
	a.logger.Info("approval: prompting host",
		"peer", peerID, "protocol", protocol, "port", port, "request_id", requestID)

	defer func() {
		a.approvalMu.Lock()
		delete(a.approvalPending, requestID)
		a.approvalMu.Unlock()
	}()

	select {
	case decision := <-wait:
		a.approvalMu.Lock()
		a.approvalDecided[key] = decision
		a.approvalMu.Unlock()
		a.logger.Info("approval: decided",
			"peer", peerID, "protocol", protocol, "port", port,
			"allow", decision)
		return decision
	case <-ctx.Done():
		// Timeout. We log but DON'T cache the deny — the user might
		// have just been afk and we want them to see the next prompt.
		a.logger.Warn("approval: timeout",
			"peer", peerID, "protocol", protocol, "port", port)
		// Also emit a 'cancelled' event so the UI can dismiss its
		// modal if it's still open.
		if a.ctx != nil {
			runtime.EventsEmit(a.ctx, "service:approval-cancelled",
				map[string]string{"requestId": requestID})
		}
		return false
	}
}

// ApproveServiceRequest / DenyServiceRequest are bound to the JS
// frontend and called when the user clicks Allow / Deny in the
// approval modal. The requestID matches the one emitted with the
// `service:approval-request` event.
func (a *App) ApproveServiceRequest(requestID string) {
	a.resolveApproval(requestID, true)
}

func (a *App) DenyServiceRequest(requestID string) {
	a.resolveApproval(requestID, false)
}

func (a *App) resolveApproval(requestID string, allow bool) {
	a.approvalMu.Lock()
	wait, ok := a.approvalPending[requestID]
	if ok {
		delete(a.approvalPending, requestID)
	}
	a.approvalMu.Unlock()
	if !ok {
		return
	}
	select {
	case wait <- allow:
	default:
	}
}

// ResetApprovalCache clears the per-(peer,port,protocol) sticky
// decisions so the user is re-prompted on the next dial. Useful
// after the host wants to revoke a previously-allowed peer.
func (a *App) ResetApprovalCache() {
	a.approvalMu.Lock()
	a.approvalDecided = make(map[approvalKey]bool)
	a.approvalMu.Unlock()
}

// SetServiceApproval flips the require_approval flag on an
// existing exposed_services row. Lets the user toggle 'tasdiqlab
// yoqish' from the UI without recreating the row.
func (a *App) SetServiceApproval(port int, protocol string, require bool) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage mavjud emas")
	}
	saved, err := store.ListExposedServices()
	if err != nil {
		return err
	}
	for _, svc := range saved {
		if svc.Port == port && svc.Protocol == protocol {
			svc.RequireApproval = require
			return store.SaveExposedService(svc)
		}
	}
	return fmt.Errorf("servis topilmadi: %s:%d", protocol, port)
}

// nicknameForPeer walks every active session looking for a peer
// matching peerID. Returns the peer's nickname if found, otherwise
// the empty string (the UI falls back to a shortened id).
func (a *App) nicknameForPeer(peerID string) string {
	for _, s := range a.allSessions() {
		if s.mesh == nil {
			continue
		}
		for _, p := range s.mesh.Peers() {
			if p.ID == peerID {
				return p.Nickname
			}
		}
	}
	return ""
}

func (a *App) serviceNameFor(saved []storage.ExposedService, port int, protocol string) string {
	for _, svc := range saved {
		if svc.Port == port && svc.Protocol == protocol {
			return svc.Name
		}
	}
	return fmt.Sprintf("%s:%d", protocol, port)
}

// newApprovalID returns a short, collision-resistant id for an
// approval round-trip. Reuses the same shape as session ids.
func newApprovalID() string {
	return "ap-" + newSessionID()[2:]
}

// ----------------------------------------------------------------------------
// View transforms
// ----------------------------------------------------------------------------

func portalToView(pi *mesh.PortalInfo) PortalView {
	if pi == nil {
		return PortalView{}
	}
	return PortalView{
		PortalID:  pi.PortalID,
		Code:      pi.Code,
		OwnerID:   pi.OwnerID,
		OwnPeerID: pi.OwnPeerID,
		OwnVIP:    pi.OwnVIP,
		IsOwner:   pi.IsOwner,
	}
}

func peerToView(p *mesh.Peer) PeerView {
	state := "connecting"
	rtt := p.RTT()
	if rtt > 0 {
		state = "connected"
	}
	services := p.Services()
	svc := make([]ServiceView, 0, len(services))
	for _, s := range services {
		svc = append(svc, ServiceView{Name: s.Name, Protocol: s.Protocol, Port: s.Port})
	}
	localTyp, remoteTyp := p.SelectedPair()
	localAddr, remoteAddr := p.SelectedPairAddrs()
	transport := ""
	if localTyp != "" || remoteTyp != "" {
		if localTyp == "relay" || remoteTyp == "relay" {
			transport = "relay"
		} else {
			transport = "direct"
		}
	}
	return PeerView{
		PeerID:          p.ID,
		Nickname:        p.Nickname,
		VirtualIP:       p.VirtualIP,
		IsOwner:         p.IsOwner,
		State:           state,
		RTTMs:           float64(rtt.Microseconds()) / 1000.0,
		BytesSent:       p.BytesSent(),
		BytesRecv:       p.BytesRecv(),
		Services:        svc,
		Transport:       transport,
		TransportLocal:  localTyp,
		TransportRemote: remoteTyp,
		PathLocalAddr:   localAddr,
		PathRemoteAddr:  remoteAddr,
	}
}
