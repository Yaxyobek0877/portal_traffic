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
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"portal_traffic/shared/protocol"
	"portal_traffic_client/peer"
	"portal_traffic_client/signaling"
)

// Config holds the runtime parameters for a Manager.
type Config struct {
	SignalingURL string
	Nickname     string
	PublicNick   bool
	ICEServers   []webrtc.ICEServer
	HeartbeatInterval time.Duration
	Logger       *slog.Logger
}

// Default ICE servers — public STUN. TURN is left to deploy-time
// configuration since most users won't need it.
var DefaultICEServers = []webrtc.ICEServer{
	{URLs: []string{"stun:stun.l.google.com:19302"}},
	{URLs: []string{"stun:stun.cloudflare.com:3478"}},
}

// Peer is the mesh's view of another participant in the portal.
// State, RTT, etc. are mutated under Manager.mu.
type Peer struct {
	ID        string
	Nickname  string
	VirtualIP string
	IsOwner   bool

	conn *peer.Connection

	// last observed round-trip time on the control channel.
	rtt          time.Duration
	rttUpdatedAt time.Time

	// outstanding ping timestamps awaiting pong, keyed by send-time
	// unix-milli. Pruned in heartbeat tick to avoid leaking on packet loss.
	outstanding map[int64]time.Time
}

// MeshEvent is what consumers see. PortalReady fires once the local
// PortalCreated/PortalJoined response has landed; PeerReady fires
// once a given peer has reached "all data channels open".
type MeshEvent struct {
	Type   MeshEventType
	Peer   *Peer
	Portal *PortalInfo
	Err    error
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
)

func (t MeshEventType) String() string {
	return [...]string{
		"portal_ready", "peer_joining", "peer_ready",
		"peer_rtt", "peer_left", "portal_closed", "error",
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
	return &Manager{
		cfg:    cfg,
		logger: cfg.Logger.With("component", "mesh"),
		peers:  make(map[string]*Peer),
		events: make(chan MeshEvent, 64),
		closed: make(chan struct{}),
	}
}

// Events returns the consumer-facing event stream. Closes when the
// manager shuts down.
func (m *Manager) Events() <-chan MeshEvent { return m.events }

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
				m.emit(MeshEvent{Type: EventPortalClosed, Err: errors.New("signaling closed")})
				return
			}
			m.handleSignalingEvent(ev)
		}
	}
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
		m.emit(MeshEvent{Type: EventError, Err: fmt.Errorf("%s: %s", ev.Error.Code, ev.Error.Message)})
	}
}

func (m *Manager) onPortalReady(portalID, code, peerID, vip, ownerID string, isOwner bool, existing []protocol.PeerInfo) {
	m.mu.Lock()
	m.myPeerID = peerID
	m.myVIP = vip
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
				announced = true
			}
		case <-time.After(200 * time.Millisecond):
			// Channels may open after the state hits Connected; poll briefly.
			if !announced && p.conn.State() == peer.StateConnected && p.conn.AllChannelsOpen() {
				m.emit(MeshEvent{Type: EventPeerReady, Peer: p})
				announced = true
			}
		}
	}
}

// ----------------------------------------------------------------------------
// Sending application data
// ----------------------------------------------------------------------------

// SendChat broadcasts a text message on the chat channel to every peer.
// Returns the number of peers it was successfully queued for.
func (m *Manager) SendChat(text string) int {
	n := 0
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.peers {
		if p.conn != nil && p.conn.ChannelOpen(peer.ChanChat) {
			if err := p.conn.SendText(peer.ChanChat, text); err == nil {
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
	return p.conn.SendText(peer.ChanChat, text)
}

// Leave releases the portal cleanly.
func (m *Manager) Leave() error {
	if m.sig == nil {
		return nil
	}
	return m.sig.Leave()
}
