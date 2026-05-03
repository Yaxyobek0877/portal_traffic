// Package mesh orchestrates the full peer-to-peer mesh from a single
// client's perspective. It owns the signaling.Client, drives the
// WebRTC handshake against every other peer in the portal, and runs
// the heartbeat over the `control` data channel.
//
// The Manager exposes a high-level event stream so consumers (the
// Wails app, the CLI test harness, future tests) can react to peers
// joining and leaving without learning the WebRTC state machine.
package mesh

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/pion/webrtc/v4"
	"portal_traffic/shared/protocol"
	"portal_traffic_client/crypt"
	"portal_traffic_client/peer"
	"portal_traffic_client/signaling"
)

// splitTurnURLs accepts a free-form list of URLs (separated by
// newlines, commas, or spaces) and returns those that look like turn:
// or turns: schemes. Empty input yields an empty slice.
func splitTurnURLs(s string) []string {
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

// Config holds the runtime parameters for a Manager.
type Config struct {
	SignalingURL      string
	Nickname          string
	PublicNick        bool
	ICEServers        []webrtc.ICEServer
	HeartbeatInterval time.Duration
	Logger            *slog.Logger

	// Optional TURN server. If TurnURL is non-empty it's appended
	// to ICEServers along with the credentials. Required for
	// peers behind symmetric NAT or CGNAT — without TURN those
	// connections silently fail to establish.
	TurnURL        string
	TurnUsername   string
	TurnCredential string
}

// Default ICE servers — public STUN only.
//
// We tried bundling a public TURN fallback (Open Relay Project) but
// their service no longer accepts the legacy openrelayproject creds,
// so peers got `had_relay=false` and silently failed instead of being
// helped. STUN-only is the honest baseline: peers on cone NAT
// connect, peers on symmetric NAT need a TURN configured in
// Settings → TURN. The doc NAT-VA-TURN.md walks the user through
// Cloudflare TURN setup (free / pay-as-you-go).
var DefaultICEServers = []webrtc.ICEServer{
	{URLs: []string{"stun:stun.l.google.com:19302"}},
	{URLs: []string{"stun:stun.cloudflare.com:3478"}},
}

// Peer is the mesh's view of another participant in the portal.
//
// Concurrency: the Manager-level mu only protects membership in the
// peers map. Per-peer mutable state (rtt, outstanding ping timestamps,
// service registry) is protected by Peer.mu, which is independent of
// the manager's lock and may be held without it. This is what avoids
// the "write under RLock" data race we'd otherwise have when heartbeat
// goroutines and message-routing goroutines both touch outstanding.
type Peer struct {
	ID        string
	Nickname  string
	VirtualIP string
	IsOwner   bool

	conn *peer.Connection

	mu sync.Mutex

	// last observed round-trip time on the control channel.
	rtt          time.Duration
	rttUpdatedAt time.Time

	// outstanding ping timestamps awaiting pong, keyed by send-time
	// unix-milli. Pruned in heartbeat tick to avoid leaking on packet loss.
	outstanding map[int64]time.Time

	// services exposed by this remote peer (announced over the control
	// channel). Keyed by port. Owned by the proxy layer.
	services map[int]ServiceAnnounce

	// bytesSent / bytesRecv are atomic counters tracked across all four
	// data channels — the diagnostic view shows them as cumulative
	// since this peer joined the portal.
	bytesSent atomic.Int64
	bytesRecv atomic.Int64
}

// ServiceAnnounce mirrors protocol.ServiceExpose for the proxy layer.
type ServiceAnnounce struct {
	Name     string
	Protocol string
	Port     int
}

// MeshEvent is what consumers see. PortalReady fires once the local
// PortalCreated/PortalJoined response has landed; PeerReady fires
// once a given peer has reached "all data channels open".
type MeshEvent struct {
	Type     MeshEventType
	Peer     *Peer
	Portal   *PortalInfo
	Err      error
	ChatText string // populated for EventChat
}

type MeshEventType int

const (
	EventPortalReady MeshEventType = iota
	EventPeerJoining
	EventPeerReady
	EventPeerRTT
	EventPeerLeft
	EventPortalClosed
	EventError
	EventChat
	EventServiceAnnounce
	EventReconnecting   // signaling dropped; we'll retry
	EventReconnected    // signaling came back and (joiner) re-joined OK
	EventReconnectGiveUp // exceeded retry budget OR portal gone
	EventPeerTransport  // ICE selected-pair changed (direct vs relay)
)

func (t MeshEventType) String() string {
	return [...]string{
		"portal_ready", "peer_joining", "peer_ready",
		"peer_rtt", "peer_left", "portal_closed", "error",
		"chat", "service_announce",
		"reconnecting", "reconnected", "reconnect_give_up",
		"peer_transport",
	}[t]
}

// PortalInfo summarises the local portal context for consumers.
type PortalInfo struct {
	PortalID  string
	Code      string
	OwnerID   string
	OwnPeerID string
	OwnVIP    string
	IsOwner   bool
}

// Manager coordinates the local peer's view of one portal.
type Manager struct {
	cfg    Config
	logger *slog.Logger

	sig *signaling.Client

	mu        sync.RWMutex
	peers     map[string]*Peer // keyed by remote peer ID
	portal    *PortalInfo
	myPeerID  string
	myVIP     string

	// portalKey is derived from the portal access code via PBKDF2 once
	// we've successfully created or joined. nil before that. App-layer
	// chat / proxy / transfer payloads are sealed with this key.
	portalKey *crypt.Key

	// localServices is what *we* expose into the mesh. Keyed by port.
	// Owned by the proxy layer; the announcement is broadcast over the
	// control channel whenever it changes.
	localServices map[int]ServiceAnnounce

	// reconnect bookkeeping. Stored at create/join time so we can re-do
	// the same operation if signaling drops.
	wasOwner    bool
	joinedID    string // empty when wasOwner
	joinedCode  string // empty when wasOwner
	wantPortal  bool   // true once we've successfully entered a portal

	// initialErr is fed when the server responds with an error to our
	// initial portal.create / portal.join, before m.portal is set.
	// CreatePortal / JoinPortal pick this up so the calling
	// frontend method returns a clean error rather than relying on
	// timeout. Buffered so handleSignalingEvent never blocks; once
	// portal is established we stop pushing here and surface errors
	// as EventError instead.
	initialErr chan error

	// proxyHandler is set by the proxy.Forwarder when the user wires
	// it in. nil means proxy frames are dropped on the floor.
	proxyHandler ProxyHandler

	// transferHandler is set by the transfer.Engine.
	transferHandler TransferHandler

	events chan MeshEvent

	closeOnce sync.Once
	closed    chan struct{}
	wg        sync.WaitGroup
}

// New constructs a Manager. Call Start (or Create / Join) to bring it up.
func New(cfg Config) *Manager {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	if cfg.HeartbeatInterval <= 0 {
		cfg.HeartbeatInterval = 5 * time.Second
	}
	if cfg.ICEServers == nil {
		cfg.ICEServers = DefaultICEServers
	}
	// Append the configured TURN server(s), if any. TurnURL may be a
	// single URL or several separated by newlines / commas / spaces —
	// we split and feed all variants to pion. Networks block ports
	// and protocols inconsistently (mobile carriers especially love
	// blocking UDP-only TURN), so listing TCP/UDP/TLS variants
	// dramatically improves connect success.
	if cfg.TurnURL != "" {
		urls := splitTurnURLs(cfg.TurnURL)
		if len(urls) > 0 {
			cfg.ICEServers = append(cfg.ICEServers, webrtc.ICEServer{
				URLs:       urls,
				Username:   cfg.TurnUsername,
				Credential: cfg.TurnCredential,
			})
		}
	}
	return &Manager{
		cfg:           cfg,
		logger:        cfg.Logger.With("component", "mesh"),
		peers:         make(map[string]*Peer),
		localServices: make(map[int]ServiceAnnounce),
		events:        make(chan MeshEvent, 64),
		closed:        make(chan struct{}),
		initialErr:    make(chan error, 1),
	}
}

// Events returns the consumer-facing event stream. Closes when the
// manager shuts down.
func (m *Manager) Events() <-chan MeshEvent { return m.events }

// InitialError returns a channel that receives at most one error if
// the server rejects our initial portal.create / portal.join. Once
// the portal is established, errors flow as EventError instead.
func (m *Manager) InitialError() <-chan error { return m.initialErr }

// Done closes when the manager has fully shut down.
func (m *Manager) Done() <-chan struct{} { return m.closed }

// Portal returns the current portal info, or nil if not yet ready.
func (m *Manager) Portal() *PortalInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.portal
}

// Peers returns a snapshot of currently tracked peers.
func (m *Manager) Peers() []*Peer {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		out = append(out, p)
	}
	return out
}

// ----------------------------------------------------------------------------
// Lifecycle
// ----------------------------------------------------------------------------

// CreatePortal connects to the signaling server, creates a portal,
// and starts the mesh runtime.
func (m *Manager) CreatePortal(ctx context.Context) error {
	if err := m.dial(ctx); err != nil {
		return err
	}
	m.mu.Lock()
	m.wasOwner = true
	m.wantPortal = true
	m.mu.Unlock()

	if err := m.sig.CreatePortal(m.cfg.Nickname, m.cfg.PublicNick, 0); err != nil {
		return err
	}
	m.startRuntime()
	return nil
}

// JoinPortal connects and joins an existing portal by ID + code.
func (m *Manager) JoinPortal(ctx context.Context, portalID, code string) error {
	if err := m.dial(ctx); err != nil {
		return err
	}
	// Pre-derive the encryption key now while we have the code in hand;
	// the server's PortalJoined response doesn't echo the code.
	k := crypt.Derive(code)
	m.mu.Lock()
	m.portalKey = &k
	m.wasOwner = false
	m.joinedID = portalID
	m.joinedCode = code
	m.wantPortal = true
	m.mu.Unlock()

	if err := m.sig.JoinPortal(portalID, code, m.cfg.Nickname); err != nil {
		return err
	}
	m.startRuntime()
	return nil
}

func (m *Manager) dial(ctx context.Context) error {
	c, err := signaling.Dial(ctx, m.cfg.SignalingURL, m.logger)
	if err != nil {
		return err
	}
	m.sig = c
	return nil
}

func (m *Manager) startRuntime() {
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.run()
	}()
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.heartbeatLoop()
	}()
}

// Close terminates all peer connections and the signaling link.
// Safe to call multiple times.
func (m *Manager) Close() {
	m.closeOnce.Do(func() {
		if m.sig != nil {
			_ = m.sig.Close()
		}
		m.mu.Lock()
		for _, p := range m.peers {
			if p.conn != nil {
				_ = p.conn.Close()
			}
		}
		m.peers = nil
		m.mu.Unlock()
		close(m.closed)
		// events isn't closed (we leave it open and rely on Done()) so
		// stragglers from peer goroutines never panic on a closed chan.
	})
	m.wg.Wait()
}

// emit publishes an event without blocking. If the buffer is full, the
// event is dropped and a warning logged — events are observation, not
// authoritative state.
func (m *Manager) emit(ev MeshEvent) {
	select {
	case <-m.closed:
	case m.events <- ev:
	default:
		m.logger.Warn("mesh event dropped; consumer slow", "type", ev.Type.String())
	}
}

// ----------------------------------------------------------------------------
// Main runtime — drains signaling events
// ----------------------------------------------------------------------------

func (m *Manager) run() {
	for {
		select {
		case <-m.closed:
			return
		case ev, ok := <-m.sig.Events():
			if !ok {
				if m.shouldReconnect() {
					if !m.attemptReconnect() {
						m.emit(MeshEvent{Type: EventReconnectGiveUp})
						return
					}
					continue
				}
				m.emit(MeshEvent{Type: EventPortalClosed, Err: errors.New("signaling closed")})
				return
			}
			m.handleSignalingEvent(ev)
		}
	}
}

func (m *Manager) shouldReconnect() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Reconnect is only useful if the user is in a portal and we've
	// stopped on our own. If the user explicitly Leave()'d, wantPortal
	// is false and we don't retry.
	return m.wantPortal
}

// attemptReconnect tries to dial signaling again with exponential
// backoff and re-issue the appropriate portal operation. Returns
// true if reconnection succeeded; false if we exhausted attempts or
// the manager was closed.
//
// For owners: re-creating the portal would change the ID/code, so we
// emit EventReconnectGiveUp instead — the UI tells the user to make
// a fresh portal and share the new code with peers (existing P2P
// connections are still up via WebRTC).
//
// For joiners: we re-issue portal.join with the same ID + code. If
// the original portal still exists on the server (other members keep
// it alive), we slot back in; otherwise the server returns
// PORTAL_NOT_FOUND and we give up.
func (m *Manager) attemptReconnect() bool {
	m.emit(MeshEvent{Type: EventReconnecting})
	m.logger.Info("signaling dropped; attempting reconnect")

	m.mu.RLock()
	wasOwner := m.wasOwner
	joinedID := m.joinedID
	joinedCode := m.joinedCode
	m.mu.RUnlock()

	if wasOwner {
		// We can't reclaim our owned portal; the server destroyed it
		// when our connection dropped. Existing peer connections are
		// still up via WebRTC, but anyone else trying to join will
		// fail. Best UX is to surface this and let the user decide.
		m.emit(MeshEvent{
			Type: EventError,
			Err:  errors.New("Tarmoq vaqtinchalik uzildi. Yangi do'stlar qo'shilishi uchun portalni qaytadan oching."),
		})
		return false
	}

	const maxAttempts = 8
	delay := time.Second
	for attempt := 0; attempt < maxAttempts; attempt++ {
		select {
		case <-m.closed:
			return false
		case <-time.After(delay):
		}

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		c, err := signaling.Dial(ctx, m.cfg.SignalingURL, m.logger)
		cancel()
		if err != nil {
			m.logger.Debug("reconnect dial failed", "attempt", attempt+1, "err", err)
			delay = capDelay(delay * 2)
			continue
		}

		// Replace the active client.
		m.mu.Lock()
		old := m.sig
		m.sig = c
		m.mu.Unlock()
		if old != nil {
			_ = old.Close()
		}

		if err := m.sig.JoinPortal(joinedID, joinedCode, m.cfg.Nickname); err != nil {
			m.logger.Debug("reconnect join failed", "attempt", attempt+1, "err", err)
			_ = c.Close()
			delay = capDelay(delay * 2)
			continue
		}

		// Success — drop existing peer entries so the joining flow
		// rebuilds them. Existing WebRTC connections to the same peers
		// will be replaced; pion handles the renegotiation.
		m.mu.Lock()
		for id, p := range m.peers {
			if p.conn != nil {
				_ = p.conn.Close()
			}
			delete(m.peers, id)
		}
		m.mu.Unlock()

		m.emit(MeshEvent{Type: EventReconnected})
		m.logger.Info("signaling reconnected", "attempt", attempt+1)
		return true
	}
	return false
}

func capDelay(d time.Duration) time.Duration {
	if d > 5*time.Minute {
		return 5 * time.Minute
	}
	return d
}

func (m *Manager) handleSignalingEvent(ev signaling.Event) {
	switch {
	case ev.Created != nil:
		m.onPortalReady(ev.Created.PortalID, ev.Created.Code, ev.Created.PeerID,
			ev.Created.VirtualIP, ev.Created.PeerID, true, nil)

	case ev.Joined != nil:
		// Owner of the existing peer list is whichever peer says is_owner.
		var ownerID string
		for _, p := range ev.Joined.Peers {
			if p.IsOwner {
				ownerID = p.PeerID
				break
			}
		}
		m.onPortalReady(ev.Joined.PortalID, "", ev.Joined.PeerID,
			ev.Joined.VirtualIP, ownerID, false, ev.Joined.Peers)

	case ev.PeerJoined != nil:
		m.onPeerJoined(ev.PeerJoined.PeerID, ev.PeerJoined.Nickname,
			ev.PeerJoined.VirtualIP)

	case ev.PeerLeft != nil:
		m.onPeerLeft(ev.PeerLeft.PeerID, ev.PeerLeft.Reason)

	case ev.Offer != nil:
		m.onOffer(ev.Offer.From, ev.Offer.SDP)

	case ev.Answer != nil:
		m.onAnswer(ev.Answer.From, ev.Answer.SDP)

	case ev.ICE != nil:
		m.onRemoteICE(ev.ICE.From, ev.ICE.Candidate, ev.ICE.SDPMid, ev.ICE.SDPMLineIndex)

	case ev.Closed != nil:
		m.emit(MeshEvent{Type: EventPortalClosed, Portal: m.Portal()})

	case ev.Error != nil:
		m.logger.Warn("signaling error", "code", ev.Error.Code, "msg", ev.Error.Message)
		err := fmt.Errorf("%s", ev.Error.Message)
		// If we don't have a portal yet, this is a response to the
		// initial create/join — route it to whoever is waiting on
		// initialErr instead of dumping a banner. Once we have a
		// portal, errors are runtime concerns and become banners.
		m.mu.RLock()
		hasPortal := m.portal != nil
		m.mu.RUnlock()
		if !hasPortal {
			select {
			case m.initialErr <- err:
			default:
			}
			return
		}
		m.emit(MeshEvent{Type: EventError, Err: err})
	}
}

func (m *Manager) onPortalReady(portalID, code, peerID, vip, ownerID string, isOwner bool, existing []protocol.PeerInfo) {
	m.mu.Lock()
	m.myPeerID = peerID
	m.myVIP = vip
	// Owner path: server returned the code in PortalCreated; derive now.
	// Joiner path: code was set in JoinPortal already, so leave portalKey alone.
	if code != "" && m.portalKey == nil {
		k := crypt.Derive(code)
		m.portalKey = &k
	}
	m.portal = &PortalInfo{
		PortalID:  portalID,
		Code:      code,
		OwnerID:   ownerID,
		OwnPeerID: peerID,
		OwnVIP:    vip,
		IsOwner:   isOwner,
	}
	portalCopy := *m.portal
	m.mu.Unlock()

	m.emit(MeshEvent{Type: EventPortalReady, Portal: &portalCopy})

	// As a joiner, initiate handshakes with every existing peer.
	for _, p := range existing {
		m.onPeerJoinedWithRoster(p.PeerID, p.Nickname, p.VirtualIP, true)
	}
}

// onPeerJoined is called when the server pushes portal.peer_joined.
// We are the existing member in this case — wait for the offer to
// arrive (the joiner is the offerer per our handshake convention).
func (m *Manager) onPeerJoined(peerID, nick, vip string) {
	m.onPeerJoinedWithRoster(peerID, nick, vip, false)
}

// onPeerJoinedWithRoster handles both code paths (initial roster on
// join, and live peer_joined after we're already in). When we're the
// joiner discovering existing peers, we initiate; when we're an
// existing peer learning about a newcomer, we wait.
func (m *Manager) onPeerJoinedWithRoster(peerID, nick, vip string, weAreJoiner bool) {
	m.mu.Lock()
	if _, exists := m.peers[peerID]; exists {
		m.mu.Unlock()
		return
	}
	// Glare-free rule: the joiner is the offerer for every existing peer.
	// Existing peers receiving portal.peer_joined always take the answerer
	// role. This works because there is exactly one "newer" side per pair.
	role := peer.RoleAnswerer
	if weAreJoiner {
		role = peer.RoleOfferer
	}
	conn, err := peer.New(peer.Config{
		LocalPeerID:  m.myPeerID,
		RemotePeerID: peerID,
		Role:         role,
		ICEServers:   m.cfg.ICEServers,
		Logger:       m.logger,
	})
	if err != nil {
		m.mu.Unlock()
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("create peer %s: %w", peerID, err)})
		return
	}
	p := &Peer{
		ID: peerID, Nickname: nick, VirtualIP: vip,
		conn:        conn,
		outstanding: map[int64]time.Time{},
		services:    map[int]ServiceAnnounce{},
	}
	m.peers[peerID] = p
	m.mu.Unlock()

	m.emit(MeshEvent{Type: EventPeerJoining, Peer: p})

	// Spawn goroutines to forward local ICE and inbound messages.
	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		m.forwardLocalICE(p)
	}()
	go func() {
		defer m.wg.Done()
		m.handlePeerMessages(p)
	}()

	if role == peer.RoleOfferer {
		m.wg.Add(1)
		go func() {
			defer m.wg.Done()
			m.kickOffOffer(p)
		}()
	}
	// State change watcher.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.watchState(p)
	}()
	// Transport (selected ICE pair) watcher.
	m.wg.Add(1)
	go func() {
		defer m.wg.Done()
		m.watchTransport(p)
	}()
}

func (m *Manager) kickOffOffer(p *Peer) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	sdp, err := p.conn.CreateOffer(ctx)
	if err != nil {
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("create offer for %s: %w", p.ID, err)})
		return
	}
	if err := m.sig.SendOffer(p.ID, sdp); err != nil {
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("send offer: %w", err)})
	}
}

func (m *Manager) onOffer(from, sdp string) {
	m.mu.RLock()
	p := m.peers[from]
	m.mu.RUnlock()
	if p == nil {
		// Could happen if peer.joined hasn't been processed yet — treat
		// the offer as a join trigger.
		m.onPeerJoinedWithRoster(from, "", "", false)
		m.mu.RLock()
		p = m.peers[from]
		m.mu.RUnlock()
		if p == nil {
			return
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	answerSDP, err := p.conn.HandleOffer(ctx, sdp)
	if err != nil {
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("handle offer from %s: %w", from, err)})
		return
	}
	if err := m.sig.SendAnswer(from, answerSDP); err != nil {
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("send answer: %w", err)})
	}
}

func (m *Manager) onAnswer(from, sdp string) {
	m.mu.RLock()
	p := m.peers[from]
	m.mu.RUnlock()
	if p == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := p.conn.HandleAnswer(ctx, sdp); err != nil {
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("handle answer from %s: %w", from, err)})
	}
}

func (m *Manager) onRemoteICE(from, cand, mid string, mline *int) {
	m.mu.RLock()
	p := m.peers[from]
	m.mu.RUnlock()
	if p == nil {
		return
	}
	if err := p.conn.AddRemoteICE(cand, mid, mline); err != nil {
		m.logger.Debug("AddRemoteICE failed", "from", from, "err", err)
	}
}

func (m *Manager) onPeerLeft(peerID, reason string) {
	m.mu.Lock()
	p := m.peers[peerID]
	delete(m.peers, peerID)
	m.mu.Unlock()
	if p != nil {
		_ = p.conn.Close()
		m.emit(MeshEvent{Type: EventPeerLeft, Peer: p})
	}
}

// ----------------------------------------------------------------------------
// Per-peer goroutines
// ----------------------------------------------------------------------------

func (m *Manager) forwardLocalICE(p *Peer) {
	for {
		select {
		case <-m.closed:
			return
		case <-p.conn.Done():
			return
		case cand := <-p.conn.LocalICE():
			mid := ""
			if cand.SDPMid != nil {
				mid = *cand.SDPMid
			}
			var mline *int
			if cand.SDPMLineIndex != nil {
				v := int(*cand.SDPMLineIndex)
				mline = &v
			}
			if err := m.sig.SendICE(p.ID, cand.Candidate, mid, mline); err != nil {
				m.logger.Debug("send ICE", "to", p.ID, "err", err)
			}
		}
	}
}

func (m *Manager) handlePeerMessages(p *Peer) {
	for {
		select {
		case <-m.closed:
			return
		case <-p.conn.Done():
			return
		case msg := <-p.conn.Messages():
			m.routeMessage(p, msg)
		}
	}
}

func (m *Manager) watchTransport(p *Peer) {
	for {
		select {
		case <-m.closed:
			return
		case <-p.conn.Done():
			return
		case <-p.conn.TransportChanges():
			m.emit(MeshEvent{Type: EventPeerTransport, Peer: p})
		}
	}
}

func (m *Manager) watchState(p *Peer) {
	announced := false
	for {
		select {
		case <-m.closed:
			return
		case <-p.conn.Done():
			return
		case <-p.conn.StateChanges():
			if !announced && p.conn.State() == peer.StateConnected && p.conn.AllChannelsOpen() {
				m.emit(MeshEvent{Type: EventPeerReady, Peer: p})
				m.replayServicesTo(p)
				announced = true
			}
		case <-time.After(200 * time.Millisecond):
			// Channels may open after the state hits Connected; poll briefly.
			if !announced && p.conn.State() == peer.StateConnected && p.conn.AllChannelsOpen() {
				m.emit(MeshEvent{Type: EventPeerReady, Peer: p})
				m.replayServicesTo(p)
				announced = true
			}
		}
	}
}

// replayServicesTo pushes our currently-exposed services to a single
// freshly-ready peer over their control channel. AnnounceService only
// broadcasts at expose time; without this, peers that join after the
// host has already exposed a port never see it.
func (m *Manager) replayServicesTo(p *Peer) {
	m.mu.RLock()
	services := make([]ServiceAnnounce, 0, len(m.localServices))
	for _, s := range m.localServices {
		services = append(services, s)
	}
	m.mu.RUnlock()

	for _, s := range services {
		_ = p.conn.SendJSON(peer.ChanControl, map[string]any{
			"type":     protocol.TypeServiceExpose,
			"name":     s.Name,
			"protocol": s.Protocol,
			"port":     s.Port,
		})
	}
}

// ----------------------------------------------------------------------------
// Sending application data
// ----------------------------------------------------------------------------

// SendChat broadcasts a text message on the chat channel to every peer.
// Returns the number of peers it was successfully queued for. Payloads
// are sealed with the portal-derived secretbox key on top of WebRTC's
// DTLS, so even a hypothetical break of the transport layer doesn't
// reveal chat content to non-portal-members.
func (m *Manager) SendChat(text string) int {
	sealed := m.sealEnvelope([]byte(text))
	n := 0
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.peers {
		if p.conn != nil && p.conn.ChannelOpen(peer.ChanChat) {
			if err := p.conn.SendBinary(peer.ChanChat, sealed); err == nil {
				n++
			}
		}
	}
	return n
}

// SendChatTo sends a chat message to a single peer.
func (m *Manager) SendChatTo(peerID, text string) error {
	m.mu.RLock()
	p, ok := m.peers[peerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not in mesh", peerID)
	}
	sealed := m.sealEnvelope([]byte(text))
	return p.conn.SendBinary(peer.ChanChat, sealed)
}

// Leave releases the portal cleanly. Clears wantPortal so a subsequent
// signaling drop won't trigger reconnect.
func (m *Manager) Leave() error {
	m.mu.Lock()
	m.wantPortal = false
	m.mu.Unlock()
	if m.sig == nil {
		return nil
	}
	return m.sig.Leave()
}

// ----------------------------------------------------------------------------
// App-layer encryption envelope helpers
// ----------------------------------------------------------------------------

// sealEnvelope wraps a plaintext payload with secretbox using the
// portal-derived key. If no key is set yet (we're still mid-handshake),
// the plaintext is returned unchanged — chat from before the portal is
// ready is rare and shouldn't be lost. Higher layers may opt to refuse.
func (m *Manager) sealEnvelope(plaintext []byte) []byte {
	m.mu.RLock()
	k := m.portalKey
	m.mu.RUnlock()
	if k == nil {
		return plaintext
	}
	return crypt.Seal(k, plaintext)
}

// openEnvelope is the inverse. text==true frames are JSON we sent
// before encryption was wired (back-compat) — pass them through. We
// detect a sealed binary frame by length: < nonce+tag is invalid;
// anything else attempts decrypt and falls back to raw on failure so
// peers running older builds still interoperate during the transition.
func (m *Manager) openEnvelope(raw []byte, isText bool) ([]byte, bool) {
	m.mu.RLock()
	k := m.portalKey
	m.mu.RUnlock()
	if k == nil || isText {
		return raw, true
	}
	pt, err := crypt.Open(k, raw)
	if err != nil {
		return nil, false
	}
	return pt, true
}

// ----------------------------------------------------------------------------
// Proxy hooks (the actual TCP forwarder lives in client/proxy)
// ----------------------------------------------------------------------------

// ProxyHandler is the contract between the mesh and the (optional)
// proxy package. It receives every decrypted inbound frame on the
// proxy data channel along with the sender's peer ID. Returning
// quickly is desirable; do heavy work in your own goroutine.
type ProxyHandler interface {
	HandleFrame(peerID string, payload []byte)
}

// TransferHandler is the file-transfer counterpart of ProxyHandler.
// Same shape, different channel.
type TransferHandler interface {
	HandleFrame(peerID string, payload []byte)
}

// SetProxyHandler registers (or replaces) the proxy.Forwarder. Pass
// nil to detach.
func (m *Manager) SetProxyHandler(h ProxyHandler) {
	m.mu.Lock()
	m.proxyHandler = h
	m.mu.Unlock()
}

// SetTransferHandler registers (or replaces) the transfer.Engine.
func (m *Manager) SetTransferHandler(h TransferHandler) {
	m.mu.Lock()
	m.transferHandler = h
	m.mu.Unlock()
}

func (m *Manager) handleProxy(p *Peer, msg peer.Message) {
	plain, ok := m.openEnvelope(msg.Raw, msg.Text)
	if !ok {
		m.logger.Warn("proxy frame decrypt failed", "from", p.ID)
		return
	}
	m.mu.RLock()
	h := m.proxyHandler
	m.mu.RUnlock()
	if h == nil {
		// Nothing exposes / dials proxy on our side; drop silently.
		return
	}
	h.HandleFrame(p.ID, plain)
}

func (m *Manager) handleTransfer(p *Peer, msg peer.Message) {
	plain, ok := m.openEnvelope(msg.Raw, msg.Text)
	if !ok {
		m.logger.Warn("transfer frame decrypt failed", "from", p.ID)
		return
	}
	m.mu.RLock()
	h := m.transferHandler
	m.mu.RUnlock()
	if h == nil {
		return
	}
	h.HandleFrame(p.ID, plain)
}

// SendProxyFrame is what the proxy package calls when it has a frame
// destined for a particular peer. The payload is sealed before it
// hits the data channel.
func (m *Manager) SendProxyFrame(peerID string, payload []byte) error {
	return m.sendOnChannel(peerID, peer.ChanProxy, payload)
}

// SendTransferFrame is the file-transfer counterpart of SendProxyFrame.
// Used by the transfer.Engine.
func (m *Manager) SendTransferFrame(peerID string, payload []byte) error {
	return m.sendOnChannel(peerID, peer.ChanTransfer, payload)
}

func (m *Manager) sendOnChannel(peerID, ch string, payload []byte) error {
	m.mu.RLock()
	p, ok := m.peers[peerID]
	m.mu.RUnlock()
	if !ok {
		return fmt.Errorf("peer %s not in mesh", peerID)
	}
	if !p.conn.ChannelOpen(ch) {
		return fmt.Errorf("%s channel not open with %s", ch, peerID)
	}
	sealed := m.sealEnvelope(payload)
	if err := p.conn.SendBinary(ch, sealed); err != nil {
		return err
	}
	p.bytesSent.Add(int64(len(sealed)))
	return nil
}

// MyPeerID returns our own server-assigned peer ID. Useful for the
// proxy layer when it needs to report bidirectional streams.
func (m *Manager) MyPeerID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.myPeerID
}

// ----------------------------------------------------------------------------
// Service announce — the local side of expose/unexpose
// ----------------------------------------------------------------------------

// AnnounceService records a locally exposed service and broadcasts it
// to every connected peer over the control channel. Idempotent.
func (m *Manager) AnnounceService(name, proto string, port int) error {
	if proto != "tcp" && proto != "udp" {
		return fmt.Errorf("unknown protocol %q", proto)
	}
	if port <= 0 || port > 65535 {
		return fmt.Errorf("port out of range")
	}
	m.mu.Lock()
	m.localServices[port] = ServiceAnnounce{Name: name, Protocol: proto, Port: port}
	m.mu.Unlock()
	return m.broadcastControl(map[string]any{
		"type":     protocol.TypeServiceExpose,
		"name":     name,
		"protocol": proto,
		"port":     port,
	})
}

// UnannounceService removes a previously exposed service.
func (m *Manager) UnannounceService(port int) error {
	m.mu.Lock()
	delete(m.localServices, port)
	m.mu.Unlock()
	return m.broadcastControl(map[string]any{
		"type": protocol.TypeServiceUnexpose,
		"port": port,
	})
}

// LocalServices returns a snapshot of services we're currently exposing.
func (m *Manager) LocalServices() []ServiceAnnounce {
	m.mu.RLock()
	defer m.mu.RUnlock()
	out := make([]ServiceAnnounce, 0, len(m.localServices))
	for _, s := range m.localServices {
		out = append(out, s)
	}
	return out
}

func (m *Manager) broadcastControl(payload map[string]any) error {
	m.mu.RLock()
	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		peers = append(peers, p)
	}
	m.mu.RUnlock()
	for _, p := range peers {
		if p.conn != nil && p.conn.ChannelOpen(peer.ChanControl) {
			_ = p.conn.SendJSON(peer.ChanControl, payload)
		}
	}
	return nil
}
