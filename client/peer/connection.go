// Package peer wraps a single pion/webrtc/v4 PeerConnection and the
// four multiplexed data channels Portal uses (`control`, `chat`,
// `transfer`, `proxy`). Higher layers (mesh.Manager) drive the
// signaling handshake; this package owns the local WebRTC state
// machine and surfaces channel-level events.
package peer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
)

// Channel names. These must match between offerer and answerer; we use
// negotiated channels with fixed IDs so opening order doesn't matter.
const (
	ChanControl  = "control"
	ChanChat     = "chat"
	ChanTransfer = "transfer"
	ChanProxy    = "proxy"
)

// channelDef describes one of the four standard data channels: its
// label, fixed negotiated ID, and reliability profile.
type channelDef struct {
	label      string
	id         uint16
	ordered    bool
	maxRetransmits *uint16 // nil = reliable
}

// channelDefs is the canonical list. Order isn't significant beyond
// allocating IDs deterministically (control=1, chat=2, transfer=3,
// proxy=4). Negotiated channels with explicit IDs let both sides
// agree without one side relying on `ondatachannel` callbacks.
var channelDefs = []channelDef{
	{ChanControl, 1, true, nil},
	{ChanChat, 2, true, nil},
	{ChanTransfer, 3, true, nil},
	{ChanProxy, 4, false, ptrU16(0)}, // unordered, no retransmits → datagram-ish
}

func ptrU16(v uint16) *uint16 { return &v }

// Role distinguishes the two sides of the handshake. The "offerer"
// generates the SDP offer; the "answerer" responds. In a mesh, each
// peer pair has exactly one offerer (we pick by lexicographic peer
// ID order — see mesh.Manager).
type Role int

const (
	RoleOfferer Role = iota
	RoleAnswerer
)

// Config holds the WebRTC ICE configuration plus identity information.
type Config struct {
	// LocalPeerID is our own peer ID (assigned by the signaling server).
	LocalPeerID string
	// RemotePeerID is who we're talking to.
	RemotePeerID string
	// Role: RoleOfferer or RoleAnswerer.
	Role Role
	// ICEServers is the STUN/TURN configuration. Required.
	ICEServers []webrtc.ICEServer
	// Logger; if nil, slog.Default is used.
	Logger *slog.Logger
}

// Message is a parsed inbound payload routed by channel name. Binary
// payloads are surfaced as Data; JSON text frames are decoded into a
// generic map plus the original Raw so callers can re-marshal into
// typed structs without paying for a second JSON pass.
type Message struct {
	Channel string
	Text    bool
	Raw     []byte
	JSON    map[string]any // populated only when Text and JSON-shaped
}

// State reports the current connection state (running, failed, closed,
// connecting). Maps roughly to webrtc.PeerConnectionState.
type State int

const (
	StateConnecting State = iota
	StateConnected
	StateFailed
	StateClosed
)

func (s State) String() string {
	return [...]string{"connecting", "connected", "failed", "closed"}[s]
}

// candidateSummary tracks which ICE candidate types we ever saw so
// the post-failure diagnostic can say "we had host+srflx but no
// relay" instead of just "ICE failed".
type candidateSummary struct {
	mu    sync.Mutex
	types map[string]int
}

func (s *candidateSummary) add(t string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.types == nil {
		s.types = make(map[string]int)
	}
	s.types[t]++
}

func (s *candidateSummary) snapshot() map[string]int {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]int, len(s.types))
	for k, v := range s.types {
		out[k] = v
	}
	return out
}

// Connection is a single peer-to-peer relationship. Each instance
// owns one webrtc.PeerConnection and four data channels.
type Connection struct {
	cfg    Config
	logger *slog.Logger

	pc       *webrtc.PeerConnection
	cands    candidateSummary

	mu       sync.Mutex
	channels map[string]*webrtc.DataChannel
	state    State
	// pending ICE candidates received before the remote description is
	// applied. We can't add them to pion until SetRemoteDescription has
	// landed, so we buffer them here.
	pendingRemoteICE []webrtc.ICECandidateInit

	// selected ICE candidate pair types ("host" | "srflx" | "prflx" |
	// "relay"). Empty until ICE picks a pair. local=="relay" or
	// remote=="relay" means traffic is going through TURN.
	//
	// We also keep the candidate addresses ("192.168.1.53:54538") so the
	// UI can answer "are we on the same Wi-Fi or going over the
	// internet?" — host pairs both being 192.168.x.x is a clear LAN tell,
	// srflx pairs are real internet traversal.
	selMu         sync.Mutex
	selLocalTyp   string
	selRemoteTyp  string
	selLocalAddr  string
	selRemoteAddr string

	// outgoing local ICE candidates: surfaced to the caller via the
	// LocalICE channel so they can be relayed through signaling.
	localICE    chan webrtc.ICECandidateInit
	messages    chan Message
	stateCh     chan State
	transportCh chan struct{} // pings on each selected-pair change

	closeOnce sync.Once
	closed    chan struct{}
}

// New creates a new Connection. The PeerConnection is started but
// the handshake hasn't begun — call CreateOffer or HandleOffer next
// depending on the role.
func New(cfg Config) (*Connection, error) {
	logger := cfg.Logger
	if logger == nil {
		logger = slog.Default()
	}
	logger = logger.With("component", "peer", "remote", cfg.RemotePeerID, "role", roleString(cfg.Role))

	api := webrtc.NewAPI()
	pc, err := api.NewPeerConnection(webrtc.Configuration{
		ICEServers: cfg.ICEServers,
	})
	if err != nil {
		return nil, fmt.Errorf("new peer connection: %w", err)
	}

	c := &Connection{
		cfg:         cfg,
		logger:      logger,
		pc:          pc,
		channels:    make(map[string]*webrtc.DataChannel, len(channelDefs)),
		state:       StateConnecting,
		localICE:    make(chan webrtc.ICECandidateInit, 32),
		messages:    make(chan Message, 64),
		stateCh:     make(chan State, 4),
		transportCh: make(chan struct{}, 4),
		closed:      make(chan struct{}),
	}

	// Surface which ICE candidate pair pion ends up using. The pair
	// types tell the user whether traffic is direct (host/srflx) or
	// relayed via TURN. SCTP/DTLS/ICE transport chain is built in
	// NewPeerConnection so accessing it here is safe.
	if sctp := pc.SCTP(); sctp != nil {
		if dtls := sctp.Transport(); dtls != nil {
			if ice := dtls.ICETransport(); ice != nil {
				ice.OnSelectedCandidatePairChange(func(pair *webrtc.ICECandidatePair) {
					if pair == nil {
						return
					}
					c.selMu.Lock()
					if pair.Local != nil {
						c.selLocalTyp = pair.Local.Typ.String()
						c.selLocalAddr = fmt.Sprintf("%s:%d", pair.Local.Address, pair.Local.Port)
					}
					if pair.Remote != nil {
						c.selRemoteTyp = pair.Remote.Typ.String()
						c.selRemoteAddr = fmt.Sprintf("%s:%d", pair.Remote.Address, pair.Remote.Port)
					}
					local, remote := c.selLocalTyp, c.selRemoteTyp
					localAddr, remoteAddr := c.selLocalAddr, c.selRemoteAddr
					c.selMu.Unlock()
					c.logger.Info("ice selected pair",
						"local", local, "local_addr", localAddr,
						"remote", remote, "remote_addr", remoteAddr,
					)
					select {
					case <-c.closed:
					case c.transportCh <- struct{}{}:
					default:
					}
				})
			}
		}
	}

	pc.OnICECandidate(func(cand *webrtc.ICECandidate) {
		var init webrtc.ICECandidateInit
		if cand != nil {
			init = cand.ToJSON()
			c.cands.add(cand.Typ.String())
			c.logger.Info("local ice candidate",
				"type", cand.Typ.String(),
				"addr", cand.Address,
				"port", cand.Port,
			)
		}
		select {
		case <-c.closed:
		case c.localICE <- init:
		}
	})

	pc.OnConnectionStateChange(func(s webrtc.PeerConnectionState) {
		c.logger.Info("peer state", "state", s.String())
		newState := translateState(s)
		c.mu.Lock()
		c.state = newState
		c.mu.Unlock()
		select {
		case <-c.closed:
		case c.stateCh <- newState:
		default:
		}
		// On failure, summarise the ICE candidates we ever saw. This
		// turns the silent "ICE failed" into an actionable diagnostic:
		//   "had host+srflx, no relay → TURN required for this NAT type"
		if newState == StateFailed {
			summary := c.cands.snapshot()
			hadRelay := summary["relay"] > 0
			hint := "TURN serveri zarur — siz Simmetrik NAT ortidasiz"
			if hadRelay {
				hint = "TURN orqali ham urinish qilindi, lekin ulanmadi (TURN balki yiqilgan)"
			}
			c.logger.Warn("peer ice failed — diagnostic",
				"candidate_types", summary,
				"had_relay", hadRelay,
				"hint", hint,
			)
		}
		if newState == StateClosed || newState == StateFailed {
			_ = c.Close()
		}
	})

	// ICE-level events at Info too — the gathering phase is where
	// most "stuck connecting" failures happen and the candidate types
	// we surface ("host", "srflx", "relay") tell you whether STUN
	// worked or you're going to need TURN.
	pc.OnICEConnectionStateChange(func(s webrtc.ICEConnectionState) {
		c.logger.Info("ice state", "state", s.String())
	})
	pc.OnICEGatheringStateChange(func(s webrtc.ICEGatheringState) {
		c.logger.Info("ice gathering", "state", s.String())
	})

	// Pre-create all four data channels with negotiated IDs. Both sides
	// do this; pion deduplicates via the ID. Without negotiation, the
	// answerer would have to wait for ondatachannel for each label.
	for _, d := range channelDefs {
		opts := &webrtc.DataChannelInit{
			Ordered:        &d.ordered,
			MaxRetransmits: d.maxRetransmits,
			Negotiated:     ptrBool(true),
			ID:             ptrU16(d.id),
		}
		dc, err := pc.CreateDataChannel(d.label, opts)
		if err != nil {
			_ = pc.Close()
			return nil, fmt.Errorf("create %s channel: %w", d.label, err)
		}
		c.attachChannel(dc)
	}

	return c, nil
}

func ptrBool(v bool) *bool { return &v }

func roleString(r Role) string {
	if r == RoleOfferer {
		return "offerer"
	}
	return "answerer"
}

func translateState(s webrtc.PeerConnectionState) State {
	switch s {
	case webrtc.PeerConnectionStateConnected:
		return StateConnected
	case webrtc.PeerConnectionStateFailed:
		return StateFailed
	case webrtc.PeerConnectionStateClosed, webrtc.PeerConnectionStateDisconnected:
		return StateClosed
	default:
		return StateConnecting
	}
}

// attachChannel wires up the OnMessage and OnOpen handlers for a
// data channel and stores it in the channels map. Caller must hold
// c.mu or be in the constructor (single goroutine).
func (c *Connection) attachChannel(dc *webrtc.DataChannel) {
	label := dc.Label()
	c.channels[label] = dc

	dc.OnOpen(func() {
		c.logger.Debug("data channel open", "label", label)
	})
	dc.OnClose(func() {
		c.logger.Debug("data channel closed", "label", label)
	})
	dc.OnError(func(err error) {
		c.logger.Warn("data channel error", "label", label, "err", err)
	})
	dc.OnMessage(func(msg webrtc.DataChannelMessage) {
		m := Message{
			Channel: label,
			Text:    msg.IsString,
			Raw:     msg.Data,
		}
		if m.Text {
			var v map[string]any
			if err := json.Unmarshal(msg.Data, &v); err == nil {
				m.JSON = v
			}
		}
		select {
		case <-c.closed:
			return
		case c.messages <- m:
		case <-time.After(2 * time.Second):
			c.logger.Warn("dropped data channel message; consumer slow",
				"label", label, "size", len(msg.Data))
		}
	})
}

// ----------------------------------------------------------------------------
// Handshake API — driven by mesh.Manager
// ----------------------------------------------------------------------------

// CreateOffer generates an SDP offer. Used by RoleOfferer.
func (c *Connection) CreateOffer(ctx context.Context) (string, error) {
	if c.cfg.Role != RoleOfferer {
		return "", errors.New("peer: not offerer role")
	}
	offer, err := c.pc.CreateOffer(nil)
	if err != nil {
		return "", fmt.Errorf("create offer: %w", err)
	}
	if err := c.pc.SetLocalDescription(offer); err != nil {
		return "", fmt.Errorf("set local desc: %w", err)
	}
	return offer.SDP, nil
}

// HandleOffer applies the remote SDP offer and produces an answer.
// Used by RoleAnswerer.
func (c *Connection) HandleOffer(ctx context.Context, sdp string) (string, error) {
	if c.cfg.Role != RoleAnswerer {
		return "", errors.New("peer: not answerer role")
	}
	if err := c.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeOffer, SDP: sdp}); err != nil {
		return "", fmt.Errorf("set remote offer: %w", err)
	}
	c.flushPendingRemoteICE()

	answer, err := c.pc.CreateAnswer(nil)
	if err != nil {
		return "", fmt.Errorf("create answer: %w", err)
	}
	if err := c.pc.SetLocalDescription(answer); err != nil {
		return "", fmt.Errorf("set local answer: %w", err)
	}
	return answer.SDP, nil
}

// HandleAnswer applies the remote SDP answer. Used by RoleOfferer
// once we receive the answer over signaling.
func (c *Connection) HandleAnswer(ctx context.Context, sdp string) error {
	if c.cfg.Role != RoleOfferer {
		return errors.New("peer: not offerer role")
	}
	if err := c.pc.SetRemoteDescription(webrtc.SessionDescription{Type: webrtc.SDPTypeAnswer, SDP: sdp}); err != nil {
		return fmt.Errorf("set remote answer: %w", err)
	}
	c.flushPendingRemoteICE()
	return nil
}

// AddRemoteICE adds an ICE candidate received from the peer over
// signaling. If the remote description hasn't been set yet, the
// candidate is buffered until it is.
func (c *Connection) AddRemoteICE(candidate string, mid string, mline *int) error {
	init := webrtc.ICECandidateInit{Candidate: candidate}
	if mid != "" {
		m := mid
		init.SDPMid = &m
	}
	if mline != nil {
		ln := uint16(*mline)
		init.SDPMLineIndex = &ln
	}

	c.mu.Lock()
	defer c.mu.Unlock()
	if c.pc.RemoteDescription() == nil {
		c.pendingRemoteICE = append(c.pendingRemoteICE, init)
		return nil
	}
	return c.pc.AddICECandidate(init)
}

func (c *Connection) flushPendingRemoteICE() {
	c.mu.Lock()
	pending := c.pendingRemoteICE
	c.pendingRemoteICE = nil
	c.mu.Unlock()
	for _, p := range pending {
		if err := c.pc.AddICECandidate(p); err != nil {
			c.logger.Warn("flush remote ICE", "err", err)
		}
	}
}

// ----------------------------------------------------------------------------
// Channel API
// ----------------------------------------------------------------------------

// SendText writes a text frame to the named channel. Returns an error
// if the channel isn't open yet.
func (c *Connection) SendText(channel, text string) error {
	c.mu.Lock()
	dc, ok := c.channels[channel]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("peer: unknown channel %q", channel)
	}
	if dc.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("peer: channel %q not open (state=%s)", channel, dc.ReadyState())
	}
	return dc.SendText(text)
}

// SendJSON marshals v and sends it as a text frame.
func (c *Connection) SendJSON(channel string, v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.SendText(channel, string(b))
}

// SendBinary writes a binary frame to the named channel.
func (c *Connection) SendBinary(channel string, data []byte) error {
	c.mu.Lock()
	dc, ok := c.channels[channel]
	c.mu.Unlock()
	if !ok {
		return fmt.Errorf("peer: unknown channel %q", channel)
	}
	if dc.ReadyState() != webrtc.DataChannelStateOpen {
		return fmt.Errorf("peer: channel %q not open", channel)
	}
	return dc.Send(data)
}

// BufferedAmount returns the SCTP outbound buffer for the named
// channel — bytes pion has accepted from us but hasn't yet pushed
// onto the wire. Used by the bandwidth probe to apply backpressure;
// returns 0 if the channel doesn't exist.
func (c *Connection) BufferedAmount(name string) uint64 {
	c.mu.Lock()
	dc, ok := c.channels[name]
	c.mu.Unlock()
	if !ok {
		return 0
	}
	return dc.BufferedAmount()
}

// ChannelOpen reports whether the named channel is open for sending.
func (c *Connection) ChannelOpen(name string) bool {
	c.mu.Lock()
	dc, ok := c.channels[name]
	c.mu.Unlock()
	if !ok {
		return false
	}
	return dc.ReadyState() == webrtc.DataChannelStateOpen
}

// AllChannelsOpen returns true once every channel has reached the
// "open" state.
func (c *Connection) AllChannelsOpen() bool {
	for _, d := range channelDefs {
		if !c.ChannelOpen(d.label) {
			return false
		}
	}
	return true
}

// ----------------------------------------------------------------------------
// Lifecycle / observation
// ----------------------------------------------------------------------------

// LocalICE returns local ICE candidates as they are gathered. The
// caller forwards them to the remote over signaling. Closes when the
// connection closes.
func (c *Connection) LocalICE() <-chan webrtc.ICECandidateInit { return c.localICE }

// Messages returns inbound data-channel messages. Closes when the
// connection closes.
func (c *Connection) Messages() <-chan Message { return c.messages }

// StateChanges yields each new state. The current state is also
// retrievable via State().
func (c *Connection) StateChanges() <-chan State { return c.stateCh }

// State returns the most recent observed connection state.
func (c *Connection) State() State {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.state
}

// SelectedPair returns the ICE candidate types pion picked for the
// active path. Empty strings until ICE has nominated a pair. Either
// side being "relay" means traffic is going through a TURN server.
func (c *Connection) SelectedPair() (local, remote string) {
	c.selMu.Lock()
	defer c.selMu.Unlock()
	return c.selLocalTyp, c.selRemoteTyp
}

// SelectedPairAddrs returns the actual host:port of each side of the
// selected ICE pair (e.g. "192.168.1.53:54538"). Empty strings until
// ICE has nominated a pair.
func (c *Connection) SelectedPairAddrs() (local, remote string) {
	c.selMu.Lock()
	defer c.selMu.Unlock()
	return c.selLocalAddr, c.selRemoteAddr
}

// TransportChanges fires whenever the ICE selected-pair changes —
// initially when ICE first nominates a pair, and again on any ICE
// restart that reroutes (e.g. TURN kicks in mid-session).
func (c *Connection) TransportChanges() <-chan struct{} { return c.transportCh }

// RemotePeerID returns who this connection is talking to.
func (c *Connection) RemotePeerID() string { return c.cfg.RemotePeerID }

// Done returns a channel that closes when the Connection is shut
// down. Use this as a termination signal in select loops; the
// per-event channels (LocalICE, Messages, StateChanges) are
// intentionally never closed to avoid races with pion callbacks
// that may still fire during teardown.
func (c *Connection) Done() <-chan struct{} { return c.closed }

// Close tears down the peer connection and all data channels.
// Idempotent.
func (c *Connection) Close() error {
	var err error
	c.closeOnce.Do(func() {
		close(c.closed)
		err = c.pc.Close()
	})
	return err
}
