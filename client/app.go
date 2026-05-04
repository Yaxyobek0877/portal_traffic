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
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"portal_traffic_client/crashreport"
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
type PortalView struct {
	PortalID  string `json:"portalId"`
	Code      string `json:"code"`
	OwnerID   string `json:"ownerId"`
	OwnPeerID string `json:"ownPeerId"`
	OwnVIP    string `json:"ownVip"`
	IsOwner   bool   `json:"isOwner"`
}

// PeerView is the JSON projection of a remote peer.
type PeerView struct {
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
}

// ChatMessage is what the frontend logs in the chat panel.
type ChatMessage struct {
	From     string    `json:"from"`     // peer id
	Nickname string    `json:"nickname"`
	Text     string    `json:"text"`
	At       time.Time `json:"at"`
	IsLocal  bool      `json:"isLocal"`
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

// LocalListener describes a TCP port the OS reports as listening.
type LocalListener struct {
	Port    int    `json:"port"`
	Process string `json:"process"`
	PID     int    `json:"pid"`
	Local   string `json:"local"`
}

// App is the Wails-bound singleton.
type App struct {
	ctx    context.Context
	logger *slog.Logger

	mu       sync.RWMutex
	mesh     *mesh.Manager
	fwd      *proxy.Forwarder
	xfer     *transfer.Engine
	nick     string
	url      string

	// connectingTo is set when CreatePortal/JoinPortal is in flight, so
	// the UI can surface a "connecting" state without polling.
	connectingTo string

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

	a.startLogSink()

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

// Shutdown cleans up the mesh, proxy, transfer engine, and SQLite.
func (a *App) Shutdown(ctx context.Context) {
	a.mu.Lock()
	m := a.mesh
	f := a.fwd
	store := a.store
	sink := a.logSink
	a.mesh = nil
	a.fwd = nil
	a.xfer = nil
	a.store = nil
	a.logSink = nil
	a.mu.Unlock()
	if sink != nil {
		// Triggers one final drain so the very last lines (incl. the
		// "shutdown" record itself once it lands) get uploaded.
		sink.Stop()
	}
	if f != nil {
		_ = f.Close()
	}
	if m != nil {
		_ = m.Leave()
		m.Close()
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
// the file-transfer affordance.
func (a *App) SaveDir() string {
	a.mu.RLock()
	x := a.xfer
	a.mu.RUnlock()
	if x == nil {
		// Fallback so the UI can show something pre-portal.
		fallback, _ := os.UserHomeDir()
		return filepath.Join(fallback, "Downloads", "Portal")
	}
	return x.SaveDir()
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
// boundary). Progress is reported via the "transfer:progress" event.
func (a *App) SendFile(peerID string) (string, error) {
	a.mu.RLock()
	x := a.xfer
	a.mu.RUnlock()
	if x == nil {
		return "", errors.New("portal yo'q")
	}
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Yuborish uchun fayl tanlang",
	})
	if err != nil || path == "" {
		return "", err
	}
	xferID, err := x.SendFile(peerID, path)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", xferID), nil
}

// SendFilePath sends a file by absolute path (used by drag-and-drop;
// the UI gets the path from Wails' OnFileDrop event).
func (a *App) SendFilePath(peerID, path string) (string, error) {
	a.mu.RLock()
	x := a.xfer
	a.mu.RUnlock()
	if x == nil {
		return "", errors.New("portal yo'q")
	}
	if path == "" {
		return "", errors.New("fayl yo'li bo'sh")
	}
	xferID, err := x.SendFile(peerID, path)
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

// CreatePortal dials signaling and creates a new portal. Returns the
// new portal info synchronously once the server has confirmed.
func (a *App) CreatePortal(nickname string, publicNick bool) (PortalView, error) {
	if err := a.bringUpMesh(nickname); err != nil {
		return PortalView{}, err
	}
	a.setConnecting(nickname)
	defer a.setConnecting("")

	if err := a.mesh.CreatePortal(a.ctx); err != nil {
		a.tearDown()
		return PortalView{}, err
	}
	pv, err := a.waitPortalReady(8 * time.Second)
	if err != nil {
		return pv, err
	}
	a.persistEnter(pv, nickname, true)
	return pv, nil
}

// JoinPortal dials signaling and joins by ID + code.
func (a *App) JoinPortal(nickname, portalID, code string) (PortalView, error) {
	if portalID == "" || code == "" {
		return PortalView{}, errors.New("portal ID va kod bo'sh bo'lmasligi kerak")
	}
	if err := a.bringUpMesh(nickname); err != nil {
		return PortalView{}, err
	}
	a.setConnecting(nickname)
	defer a.setConnecting("")

	if err := a.mesh.JoinPortal(a.ctx, portalID, code); err != nil {
		a.tearDown()
		return PortalView{}, err
	}
	pv, err := a.waitPortalReady(8 * time.Second)
	if err != nil {
		return pv, err
	}
	// JoinPortal doesn't echo the code back from the server, so we
	// persist what the user typed.
	pv.Code = code
	a.persistEnter(pv, nickname, false)
	pv.Code = "" // don't surface the code on the joiner UI side
	return pv, nil
}

// persistEnter records this portal in history and saves the nickname
// for next launch. Best-effort; failures are logged at debug level.
func (a *App) persistEnter(pv PortalView, nickname string, isOwner bool) {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
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
}

// Leave detaches from the current portal but keeps the app running.
func (a *App) Leave() error {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return nil
	}
	a.tearDown()
	return nil
}

// CurrentPortal returns the current portal info, or zero-value if
// not in a portal.
func (a *App) CurrentPortal() PortalView {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return PortalView{}
	}
	return portalToView(m.Portal())
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
// returns the receiver-measured throughput. Blocks; the UI should
// call this from a worker context (Wails handles that automatically).
func (a *App) MeasureBandwidth(peerID string) (BandwidthResult, error) {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return BandwidthResult{}, errors.New("portal yo'q")
	}
	r, err := m.MeasureBandwidth(peerID)
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

// Peers returns a snapshot of currently tracked peers.
func (a *App) Peers() []PeerView {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return []PeerView{}
	}
	peers := m.Peers()
	out := make([]PeerView, 0, len(peers))
	for _, p := range peers {
		out = append(out, peerToView(p))
	}
	return out
}

// SendChat broadcasts a chat message to every peer. Returns peer count.
func (a *App) SendChat(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return 0
	}
	n := m.SendChat(text)
	// Echo locally so the sender sees their own message immediately.
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, "chat", ChatMessage{
			From:     m.MyPeerID(),
			Nickname: a.nickname(),
			Text:     text,
			At:       time.Now(),
			IsLocal:  true,
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

// LocalListeners enumerates TCP ports the OS reports as listening.
// Used by the Services panel to offer one-click "expose" for the
// services already running on the user's machine.
//
// Implementation: shell out to `lsof -nP -iTCP -sTCP:LISTEN -F pcPLn`
// (macOS) or fall back to `ss -lntp` (Linux). Both produce parseable
// output. Errors are returned as an empty list so the UI stays
// usable on platforms where neither is available (Windows, sandboxed
// environments).
func (a *App) LocalListeners() []LocalListener {
	out := []LocalListener{}
	cmd, parser := localListenersCommand()
	if cmd == nil {
		return out
	}
	stdout, err := cmd.Output()
	if err != nil {
		a.logger.Debug("local listeners enumerate failed", "err", err)
		return out
	}
	rows := parser(string(stdout))
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
	seen := map[int]bool{}
	for _, r := range rows {
		if r.Port < 1024 || systemNoise[r.Process] {
			continue
		}
		if seen[r.Port] {
			continue
		}
		seen[r.Port] = true
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Port < out[j].Port })
	return out
}

// LocalServices returns the services we are currently exposing.
func (a *App) LocalServices() []ServiceView {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return []ServiceView{}
	}
	out := []ServiceView{}
	for _, s := range m.LocalServices() {
		out = append(out, ServiceView{Name: s.Name, Protocol: s.Protocol, Port: s.Port})
	}
	return out
}

// ExposeService registers a local TCP port and announces it to peers.
func (a *App) ExposeService(name string, port int) error {
	a.mu.RLock()
	m := a.mesh
	f := a.fwd
	a.mu.RUnlock()
	if m == nil || f == nil {
		return errors.New("portal yo'q")
	}
	if name == "" {
		name = fmt.Sprintf("tcp:%d", port)
	}
	f.Expose(port)
	return m.AnnounceService(name, "tcp", port)
}

// UnexposeService removes a previously exposed port.
func (a *App) UnexposeService(port int) error {
	a.mu.RLock()
	m := a.mesh
	f := a.fwd
	a.mu.RUnlock()
	if m == nil || f == nil {
		return errors.New("portal yo'q")
	}
	f.Unexpose(port)
	return m.UnannounceService(port)
}

// DialService opens a local TCP listener that pumps connections to
// peerID's exposed remotePort. localPort 0 means "match remotePort if
// free, otherwise let the OS pick" — this way the local alias mirrors
// the remote address (127.0.0.1:5000 for a remote :5000) which is what
// users expect when they exposed e.g. a Minecraft server on :25565.
// Returns the resolved local addr ("127.0.0.1:5000") so the UI shows it.
func (a *App) DialService(peerID string, remotePort, localPort int) (string, error) {
	a.mu.RLock()
	f := a.fwd
	a.mu.RUnlock()
	if f == nil {
		return "", errors.New("portal yo'q")
	}
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
}

// ----------------------------------------------------------------------------
// Internals
// ----------------------------------------------------------------------------

// bringUpMesh allocates a fresh mesh.Manager + proxy.Forwarder and
// starts the event-pump goroutine. Idempotent: tearing down first if
// a previous session existed.
func (a *App) bringUpMesh(nickname string) error {
	nickname = strings.TrimSpace(nickname)
	if nickname == "" {
		return errors.New("taxallus bo'sh bo'lmasligi kerak")
	}
	a.tearDown()

	url := a.SignalingURL()
	turn := a.GetTurnConfig()
	// Try Cloudflare TURN first; if creds work the call returns ICE
	// servers (STUN defaults + Cloudflare relay) and we use them
	// directly. nil means "fall back to mesh defaults plus the manual
	// TurnURL field if any".
	cfICE := a.resolveICEServers()
	a.mu.Lock()
	a.nick = nickname
	a.mesh = mesh.New(mesh.Config{
		SignalingURL:      url,
		Nickname:          nickname,
		HeartbeatInterval: 3 * time.Second,
		Logger:            a.logger,
		ICEServers:        cfICE, // nil → mesh.DefaultICEServers
		TurnURL:           turn.URL,
		TurnUsername:      turn.Username,
		TurnCredential:    turn.Credential,
	})
	a.fwd = proxy.New(a.mesh, a.logger)
	a.mesh.SetProxyHandler(a.fwd)

	// Transfer engine — emits ProgressEvents straight to the frontend
	// via Wails events. Uses ~/Downloads/Portal as save dir by default.
	ctx := a.ctx
	a.xfer = transfer.NewEngine(a.mesh, "", a.logger, func(ev transfer.ProgressEvent) {
		if ctx != nil {
			runtime.EventsEmit(ctx, "transfer:progress", ev)
		}
	})
	a.mesh.SetTransferHandler(a.xfer)
	a.mu.Unlock()

	go a.pumpEvents()
	return nil
}

func (a *App) tearDown() {
	a.mu.Lock()
	m := a.mesh
	f := a.fwd
	a.mesh = nil
	a.fwd = nil
	a.mu.Unlock()
	if f != nil {
		_ = f.Close()
	}
	if m != nil {
		_ = m.Leave()
		m.Close()
	}
}

func (a *App) setConnecting(s string) {
	a.mu.Lock()
	a.connectingTo = s
	a.mu.Unlock()
}

func (a *App) nickname() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.nick
}

// waitPortalReady blocks until the mesh emits PortalReady, the
// signaling server responds with an error, the deadline expires,
// or we get torn down. Errors come back fast (the server typically
// rejects bad portal IDs in ~50ms) so the user isn't left staring
// at a "Ulanmoqda..." spinner for the full timeout window.
func (a *App) waitPortalReady(timeout time.Duration) (PortalView, error) {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return PortalView{}, errors.New("mesh tear down")
	}

	deadline := time.After(timeout)
	tick := time.NewTicker(50 * time.Millisecond)
	defer tick.Stop()
	for {
		if pi := m.Portal(); pi != nil {
			return portalToView(pi), nil
		}
		select {
		case err := <-m.InitialError():
			a.tearDown()
			return PortalView{}, err
		case <-deadline:
			a.tearDown()
			return PortalView{}, errors.New("portal javobi kelmadi (timeout)")
		case <-tick.C:
		}
	}
}

// pumpEvents fans mesh events out to the frontend via Wails event bus.
// Runs until the mesh.Manager closes.
func (a *App) pumpEvents() {
	a.mu.RLock()
	m := a.mesh
	a.mu.RUnlock()
	if m == nil {
		return
	}
	for {
		select {
		case <-m.Done():
			return
		case ev, ok := <-m.Events():
			if !ok {
				return
			}
			a.relayEvent(ev)
		}
	}
}

func (a *App) relayEvent(ev mesh.MeshEvent) {
	if a.ctx == nil {
		return
	}
	switch ev.Type {
	case mesh.EventPortalReady:
		// Pass through portalToView so the JSON we emit matches the
		// PortalView shape the frontend subscribes with (portalId,
		// code, ownerId — lowerCamel json tags). Sending *ev.Portal
		// directly serialised the raw mesh.PortalInfo struct with Go
		// field names (PortalID, Code, OwnerID), so the subscriber's
		// setPortal() overwrote the good data Welcome had just put
		// in the store with object whose fields were all undefined —
		// the ID/KOD then rendered blank. Race was platform-dependent
		// (Windows hit it most reliably; macOS sometimes won the race).
		runtime.EventsEmit(a.ctx, "portal:ready", portalToView(ev.Portal))

	case mesh.EventPeerJoining:
		runtime.EventsEmit(a.ctx, "peer:joining", peerToView(ev.Peer))

	case mesh.EventPeerReady:
		runtime.EventsEmit(a.ctx, "peer:ready", peerToView(ev.Peer))

	case mesh.EventPeerRTT:
		runtime.EventsEmit(a.ctx, "peer:rtt", peerToView(ev.Peer))

	case mesh.EventPeerTransport:
		runtime.EventsEmit(a.ctx, "peer:transport", peerToView(ev.Peer))

	case mesh.EventPeerLeft:
		runtime.EventsEmit(a.ctx, "peer:left", peerToView(ev.Peer))

	case mesh.EventPortalClosed:
		runtime.EventsEmit(a.ctx, "portal:closed", nil)

	case mesh.EventChat:
		runtime.EventsEmit(a.ctx, "chat", ChatMessage{
			From:     ev.Peer.ID,
			Nickname: ev.Peer.Nickname,
			Text:     ev.ChatText,
			At:       time.Now(),
			IsLocal:  false,
		})

	case mesh.EventServiceAnnounce:
		runtime.EventsEmit(a.ctx, "peer:services", peerToView(ev.Peer))

	case mesh.EventError:
		msg := ""
		if ev.Err != nil {
			msg = ev.Err.Error()
		}
		runtime.EventsEmit(a.ctx, "error", msg)
	}
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
