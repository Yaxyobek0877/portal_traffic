// Package signaling is the WebSocket client that talks to the Portal
// signaling server. It dials the server, sends typed outbound messages,
// and dispatches inbound messages onto a single Events channel as typed
// values. Higher layers (mesh.Manager) consume Events and decide what
// to do with each one.
//
// The client is goroutine-safe for sends. Reads are serialised by a
// single read goroutine so consumers don't race over the socket.
package signaling

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"portal_traffic/shared/protocol"
)

// Event is the discriminated union of every server-pushed message we
// surface to consumers. Exactly one field is non-nil per event.
//
// We model this as a struct of pointers rather than an interface union
// because Go interfaces here would require a type-switch at every
// consumer; the struct shape keeps the consumer code obvious.
type Event struct {
	Created    *protocol.PortalCreated
	Joined     *protocol.PortalJoined
	PeerJoined *protocol.PortalPeerJoined
	PeerLeft   *protocol.PortalPeerLeft
	JoinReq    *protocol.PortalJoinRequest
	Kicked     *protocol.PortalKicked
	Locked     *protocol.PortalLocked
	Closed     *protocol.PortalClosed
	Offer      *protocol.WebRTCOffer
	Answer     *protocol.WebRTCAnswer
	ICE        *protocol.WebRTCICE
	Error      *protocol.Error

	// Raw is the original frame; kept so consumers can log or replay
	// without re-marshaling.
	Raw []byte
}

// Client is the typed WebSocket client.
type Client struct {
	url    string
	logger *slog.Logger

	conn   *websocket.Conn
	connMu sync.Mutex // protects writes to conn

	events chan Event

	closed    atomic.Bool
	closeOnce sync.Once
	done      chan struct{}
}

// Dial connects to the signaling server and returns a ready Client.
// Read pump is started immediately; consume Events to make progress.
func Dial(ctx context.Context, url string, logger *slog.Logger) (*Client, error) {
	if logger == nil {
		logger = slog.Default()
	}
	dialer := websocket.Dialer{
		HandshakeTimeout: 10 * time.Second,
	}
	conn, _, err := dialer.DialContext(ctx, url, nil)
	if err != nil {
		return nil, fmt.Errorf("signaling dial %s: %w", url, err)
	}
	c := &Client{
		url:    url,
		logger: logger.With("component", "signaling"),
		conn:   conn,
		events: make(chan Event, 64),
		done:   make(chan struct{}),
	}
	go c.readPump()
	return c, nil
}

// Events returns the receive-only event channel. The channel closes
// when the connection drops.
func (c *Client) Events() <-chan Event { return c.events }

// Done is closed when the client has fully shut down.
func (c *Client) Done() <-chan struct{} { return c.done }

// URL returns the configured signaling URL (handy for diagnostics).
func (c *Client) URL() string { return c.url }

// Close terminates the connection. Idempotent.
func (c *Client) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.closed.Store(true)
		c.connMu.Lock()
		defer c.connMu.Unlock()
		// Best-effort close frame; ignore errors since the conn may already be torn.
		_ = c.conn.WriteControl(
			websocket.CloseMessage,
			websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""),
			time.Now().Add(time.Second),
		)
		err = c.conn.Close()
	})
	return err
}

// ----------------------------------------------------------------------------
// Outbound (typed senders)
// ----------------------------------------------------------------------------

// CreatePortal sends portal.create. Returns the marshaling error only;
// the server response arrives as a Created event.
func (c *Client) CreatePortal(nick string, publicNick bool, capacity int) error {
	return c.send(protocol.PortalCreate{
		Type:       protocol.TypePortalCreate,
		Nickname:   nick,
		PublicNick: publicNick,
		Capacity:   capacity,
	})
}

// JoinPortal sends portal.join.
func (c *Client) JoinPortal(portalID, code, nick string) error {
	return c.send(protocol.PortalJoin{
		Type:     protocol.TypePortalJoin,
		PortalID: portalID,
		Code:     code,
		Nickname: nick,
	})
}

// JoinByNick sends portal.join_by_nick.
func (c *Client) JoinByNick(target, my, requestID string) error {
	return c.send(protocol.PortalJoinByNick{
		Type:           protocol.TypePortalJoinByNick,
		TargetNickname: target,
		MyNickname:     my,
		RequestID:      requestID,
	})
}

// AcceptJoin / DeclineJoin reply to an inbound JoinReq event.
func (c *Client) AcceptJoin(requestID string) error {
	return c.send(protocol.PortalJoinResponse{Type: protocol.TypePortalJoinResponse, RequestID: requestID, Accept: true})
}

func (c *Client) DeclineJoin(requestID string) error {
	return c.send(protocol.PortalJoinResponse{Type: protocol.TypePortalJoinResponse, RequestID: requestID, Accept: false})
}

// Leave sends portal.leave.
func (c *Client) Leave() error {
	return c.send(protocol.PortalLeave{Type: protocol.TypePortalLeave})
}

// Kick sends portal.kick (owner only — server enforces).
func (c *Client) Kick(peerID string) error {
	return c.send(protocol.PortalKick{Type: protocol.TypePortalKick, PeerID: peerID})
}

// Lock sends portal.lock.
func (c *Client) Lock(locked bool) error {
	return c.send(protocol.PortalLock{Type: protocol.TypePortalLock, Locked: locked})
}

// SendOffer sends a WebRTC offer to a peer.
func (c *Client) SendOffer(to, sdp string) error {
	return c.send(protocol.WebRTCOffer{Type: protocol.TypeWebRTCOffer, To: to, SDP: sdp})
}

// SendAnswer sends a WebRTC answer.
func (c *Client) SendAnswer(to, sdp string) error {
	return c.send(protocol.WebRTCAnswer{Type: protocol.TypeWebRTCAnswer, To: to, SDP: sdp})
}

// SendICE sends a single ICE candidate. mid and mline may be omitted.
func (c *Client) SendICE(to, candidate, mid string, mline *int) error {
	return c.send(protocol.WebRTCICE{
		Type:          protocol.TypeWebRTCICE,
		To:            to,
		Candidate:     candidate,
		SDPMid:        mid,
		SDPMLineIndex: mline,
	})
}

// send marshals v and writes it to the socket. Concurrent senders are
// serialised by connMu; the websocket library does not allow concurrent
// writers.
func (c *Client) send(v any) error {
	if c.closed.Load() {
		return errors.New("signaling: client closed")
	}
	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Errorf("signaling marshal: %w", err)
	}
	c.connMu.Lock()
	defer c.connMu.Unlock()
	_ = c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return c.conn.WriteMessage(websocket.TextMessage, b)
}

// ----------------------------------------------------------------------------
// Inbound
// ----------------------------------------------------------------------------

// readPump owns the read side of the socket. It runs until the
// connection drops, then closes Events and Done.
func (c *Client) readPump() {
	defer func() {
		close(c.events)
		close(c.done)
	}()
	for {
		_, raw, err := c.conn.ReadMessage()
		if err != nil {
			if !c.closed.Load() {
				c.logger.Debug("signaling read error", "err", err)
			}
			return
		}
		ev, err := decodeEvent(raw)
		if err != nil {
			c.logger.Warn("signaling decode error", "err", err, "raw", string(raw))
			continue
		}
		// Non-blocking send if buffer is full would lose events; better to
		// block briefly so consumers stay in sync. If they really stall,
		// they can hold us up — that's correct backpressure.
		c.events <- ev
	}
}

// decodeEvent inspects the type field and unmarshals into the matching
// payload struct, returning a populated Event.
func decodeEvent(raw []byte) (Event, error) {
	t, err := protocol.TypeOf(raw)
	if err != nil {
		return Event{Raw: raw}, err
	}
	ev := Event{Raw: raw}
	switch t {
	case protocol.TypePortalCreated:
		v := &protocol.PortalCreated{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Created = v
	case protocol.TypePortalJoined:
		v := &protocol.PortalJoined{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Joined = v
	case protocol.TypePortalPeerJoined:
		v := &protocol.PortalPeerJoined{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.PeerJoined = v
	case protocol.TypePortalPeerLeft:
		v := &protocol.PortalPeerLeft{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.PeerLeft = v
	case protocol.TypePortalJoinRequest:
		v := &protocol.PortalJoinRequest{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.JoinReq = v
	case protocol.TypePortalKicked:
		v := &protocol.PortalKicked{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Kicked = v
	case protocol.TypePortalLocked:
		v := &protocol.PortalLocked{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Locked = v
	case protocol.TypePortalClosed:
		v := &protocol.PortalClosed{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Closed = v
	case protocol.TypeWebRTCOffer:
		v := &protocol.WebRTCOffer{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Offer = v
	case protocol.TypeWebRTCAnswer:
		v := &protocol.WebRTCAnswer{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Answer = v
	case protocol.TypeWebRTCICE:
		v := &protocol.WebRTCICE{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.ICE = v
	case protocol.TypeError:
		v := &protocol.Error{}
		if err := json.Unmarshal(raw, v); err != nil {
			return ev, err
		}
		ev.Error = v
	default:
		return ev, fmt.Errorf("signaling: unknown type %q", t)
	}
	return ev, nil
}
