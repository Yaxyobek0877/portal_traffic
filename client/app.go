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
	"strings"
	"sync"
	"time"

	"github.com/wailsapp/wails/v2/pkg/runtime"

	"portal_traffic_client/mesh"
	"portal_traffic_client/proxy"
	"portal_traffic_client/signaling"
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
	Services  []ServiceView  `json:"services"`
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

// App is the Wails-bound singleton.
type App struct {
	ctx    context.Context
	logger *slog.Logger

	mu       sync.RWMutex
	mesh     *mesh.Manager
	fwd      *proxy.Forwarder
	nick     string
	url      string

	// connectingTo is set when CreatePortal/JoinPortal is in flight, so
	// the UI can surface a "connecting" state without polling.
	connectingTo string
}

// NewApp constructs the app singleton; main.go binds it.
func NewApp(logger *slog.Logger) *App {
	return &App{logger: logger}
}

// Startup is called once Wails has the runtime context. We use it
// to remember ctx for runtime.EventsEmit calls from any goroutine.
func (a *App) Startup(ctx context.Context) {
	a.ctx = ctx
	a.logger.Info("portal app starting")
}

// Shutdown cleans up the mesh and proxy on quit.
func (a *App) Shutdown(ctx context.Context) {
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

// ----------------------------------------------------------------------------
// Bound methods (called from the frontend)
// ----------------------------------------------------------------------------

// SignalingURL returns the configured signaling URL (effectively the
// project default unless overridden by env).
func (a *App) SignalingURL() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	if a.url != "" {
		return a.url
	}
	return signaling.DefaultURL
}

// SetSignalingURL changes the URL we'll connect to on next create/join.
// Has no effect on a session already in flight.
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
	a.mu.Unlock()
	return nil
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
	return a.waitPortalReady(8 * time.Second)
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
	return a.waitPortalReady(8 * time.Second)
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
// peerID's exposed remotePort. localPort 0 means OS-pick. Returns the
// resolved local addr (e.g. "127.0.0.1:51234") so the UI can show it.
func (a *App) DialService(peerID string, remotePort, localPort int) (string, error) {
	a.mu.RLock()
	f := a.fwd
	a.mu.RUnlock()
	if f == nil {
		return "", errors.New("portal yo'q")
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
	a.mu.Lock()
	a.nick = nickname
	a.mesh = mesh.New(mesh.Config{
		SignalingURL:      url,
		Nickname:          nickname,
		HeartbeatInterval: 3 * time.Second,
		Logger:            a.logger,
	})
	a.fwd = proxy.New(a.mesh, a.logger)
	a.mesh.SetProxyHandler(a.fwd)
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

// waitPortalReady blocks until the mesh emits PortalReady, then
// returns the projection.
func (a *App) waitPortalReady(timeout time.Duration) (PortalView, error) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		a.mu.RLock()
		m := a.mesh
		a.mu.RUnlock()
		if m == nil {
			return PortalView{}, errors.New("mesh tear down")
		}
		if pi := m.Portal(); pi != nil {
			return portalToView(pi), nil
		}
		time.Sleep(50 * time.Millisecond)
	}
	return PortalView{}, errors.New("portal javobi kelmadi (timeout)")
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
		runtime.EventsEmit(a.ctx, "portal:ready", *ev.Portal)

	case mesh.EventPeerJoining:
		runtime.EventsEmit(a.ctx, "peer:joining", peerToView(ev.Peer))

	case mesh.EventPeerReady:
		runtime.EventsEmit(a.ctx, "peer:ready", peerToView(ev.Peer))

	case mesh.EventPeerRTT:
		runtime.EventsEmit(a.ctx, "peer:rtt", peerToView(ev.Peer))

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
	return PeerView{
		PeerID:    p.ID,
		Nickname:  p.Nickname,
		VirtualIP: p.VirtualIP,
		IsOwner:   p.IsOwner,
		State:     state,
		RTTMs:     float64(rtt.Microseconds()) / 1000.0,
		Services:  svc,
	}
}
