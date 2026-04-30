// Package protocol defines the wire format used between Portal clients and
// the signaling server, and between peers over the `control` WebRTC data
// channel. All messages are JSON-encoded with a top-level "type" discriminator.
//
// The server-facing message set covers portal lifecycle (create/join/leave),
// nickname directory operations, and WebRTC handshake relay (offer/answer/ice).
// After WebRTC connections are established, peers exchange messages directly
// over data channels; those types are also defined here for shared use.
package protocol

import (
	"encoding/json"
	"errors"
)

// ----------------------------------------------------------------------------
// Message type constants
// ----------------------------------------------------------------------------

const (
	// Client → Server
	TypePortalCreate         = "portal.create"
	TypePortalJoin           = "portal.join"
	TypePortalJoinByNick     = "portal.join_by_nick"
	TypePortalJoinResponse   = "portal.join_response" // owner accepts/declines a nick-based join request
	TypePortalLeave          = "portal.leave"
	TypePortalKick           = "portal.kick"
	TypePortalLock           = "portal.lock"
	TypeNickSetVisibility    = "nick.set_visibility"

	// Server → Client (responses / pushes)
	TypePortalCreated      = "portal.created"
	TypePortalJoined       = "portal.joined"
	TypePortalPeerJoined   = "portal.peer_joined"
	TypePortalPeerLeft     = "portal.peer_left"
	TypePortalJoinRequest  = "portal.join_request"
	TypePortalKicked       = "portal.kicked"
	TypePortalLocked       = "portal.locked"
	TypePortalClosed       = "portal.closed"
	TypeError              = "error"

	// Bidirectional, relayed by server between two peers
	TypeWebRTCOffer  = "webrtc.offer"
	TypeWebRTCAnswer = "webrtc.answer"
	TypeWebRTCICE    = "webrtc.ice"

	// P2P-only (sent over the `control` data channel after WebRTC is up).
	// Documented here so client and any future debug tooling share definitions.
	TypePing                = "ping"
	TypePong                = "pong"
	TypePresence            = "presence"
	TypeServiceExpose       = "service.expose"
	TypeServiceUnexpose     = "service.unexpose"
	TypeServiceListRequest  = "service.list_request"
	TypeServiceListResponse = "service.list_response"
)

// ----------------------------------------------------------------------------
// Error codes
// ----------------------------------------------------------------------------

const (
	ErrInvalidMessage    = "INVALID_MESSAGE"
	ErrPortalNotFound    = "PORTAL_NOT_FOUND"
	ErrPortalCodeWrong   = "PORTAL_CODE_WRONG"
	ErrPortalFull        = "PORTAL_FULL"
	ErrPortalLocked      = "PORTAL_LOCKED"
	ErrNicknameInvalid   = "NICKNAME_INVALID"
	ErrNicknameTaken     = "NICKNAME_TAKEN"
	ErrNicknameNotFound  = "NICKNAME_NOT_FOUND"
	ErrNicknameNotPublic = "NICKNAME_NOT_PUBLIC"
	ErrNotInPortal       = "NOT_IN_PORTAL"
	ErrAlreadyInPortal   = "ALREADY_IN_PORTAL"
	ErrPeerNotFound      = "PEER_NOT_FOUND"
	ErrNotOwner          = "NOT_OWNER"
	ErrRateLimited       = "RATE_LIMITED"
	ErrInternal          = "INTERNAL"
)

// ----------------------------------------------------------------------------
// Shared types
// ----------------------------------------------------------------------------

// PeerInfo is sent in PortalJoined (existing peer list) and PortalPeerJoined.
// It contains everything a peer needs to identify another peer and start the
// WebRTC handshake with them.
type PeerInfo struct {
	PeerID    string `json:"peer_id"`
	Nickname  string `json:"nickname"`
	VirtualIP string `json:"virtual_ip"`
	IsOwner   bool   `json:"is_owner"`
}

// NickVisibility values for nickname directory entries.
const (
	NickHidden       = "hidden"        // not in directory at all
	NickFriendsOnly  = "friends_only"  // visible only to known contacts (client-side enforced)
	NickPublic       = "public"        // anyone can look up and request a join
)

// ----------------------------------------------------------------------------
// Header — used for first-pass type extraction
// ----------------------------------------------------------------------------

// Header is the minimal envelope used to discriminate a message type before
// unmarshaling into a concrete struct. Use TypeOf as the canonical entry point.
type Header struct {
	Type string `json:"type"`
}

// TypeOf parses just enough JSON to extract the "type" field. Returns an
// error if the input is not a JSON object or "type" is missing.
func TypeOf(raw []byte) (string, error) {
	var h Header
	if err := json.Unmarshal(raw, &h); err != nil {
		return "", err
	}
	if h.Type == "" {
		return "", errors.New("protocol: missing type field")
	}
	return h.Type, nil
}

// ----------------------------------------------------------------------------
// Client → Server messages
// ----------------------------------------------------------------------------

// PortalCreate asks the server to allocate a new portal and admit the sender
// as its owner.
type PortalCreate struct {
	Type       string `json:"type"`
	Nickname   string `json:"nickname"`
	PublicNick bool   `json:"public_nick"`
	Capacity   int    `json:"capacity,omitempty"` // 0 = use server default (16)
}

// PortalJoin asks to join an existing portal by ID and code.
type PortalJoin struct {
	Type     string `json:"type"`
	PortalID string `json:"portal_id"`
	Code     string `json:"code"`
	Nickname string `json:"nickname"`
}

// PortalJoinByNick asks the server to forward a join request to the user with
// the given nickname. The target must have set their nickname to public; the
// target receives a PortalJoinRequest and replies with PortalJoinResponse.
type PortalJoinByNick struct {
	Type           string `json:"type"`
	TargetNickname string `json:"target_nickname"`
	MyNickname     string `json:"my_nickname"`
	RequestID      string `json:"request_id"` // correlates server response back to caller
}

// PortalJoinResponse is the owner's answer to an incoming join-by-nick request.
// The server admits or rejects the requester accordingly.
type PortalJoinResponse struct {
	Type      string `json:"type"`
	RequestID string `json:"request_id"`
	Accept    bool   `json:"accept"`
}

// PortalLeave removes the sender from their current portal.
type PortalLeave struct {
	Type string `json:"type"`
}

// PortalKick removes the named peer from the sender's portal. Owner-only.
type PortalKick struct {
	Type   string `json:"type"`
	PeerID string `json:"peer_id"`
}

// PortalLock toggles the locked state of the sender's portal. Owner-only.
// When locked, no new peers can join (even with the correct code).
type PortalLock struct {
	Type   string `json:"type"`
	Locked bool   `json:"locked"`
}

// NickSetVisibility updates the sender's nickname directory entry.
type NickSetVisibility struct {
	Type       string `json:"type"`
	Nickname   string `json:"nickname"`
	Visibility string `json:"visibility"` // NickHidden | NickFriendsOnly | NickPublic
}

// ----------------------------------------------------------------------------
// Server → Client messages
// ----------------------------------------------------------------------------

// PortalCreated confirms a successful portal.create.
type PortalCreated struct {
	Type      string `json:"type"`
	PortalID  string `json:"portal_id"`
	Code      string `json:"code"`
	PeerID    string `json:"peer_id"`
	VirtualIP string `json:"virtual_ip"`
	Capacity  int    `json:"capacity"`
}

// PortalJoined confirms a successful portal.join (or accepted join-by-nick)
// and provides the existing peer roster so the joiner can initiate WebRTC
// handshakes with each one.
type PortalJoined struct {
	Type      string     `json:"type"`
	PortalID  string     `json:"portal_id"`
	PeerID    string     `json:"peer_id"`
	VirtualIP string     `json:"virtual_ip"`
	Peers     []PeerInfo `json:"peers"`
}

// PortalPeerJoined notifies existing members that a new peer has joined.
type PortalPeerJoined struct {
	Type      string `json:"type"`
	PeerID    string `json:"peer_id"`
	Nickname  string `json:"nickname"`
	VirtualIP string `json:"virtual_ip"`
}

// PortalPeerLeft notifies remaining members that a peer has left.
type PortalPeerLeft struct {
	Type   string `json:"type"`
	PeerID string `json:"peer_id"`
	Reason string `json:"reason,omitempty"` // e.g. "left", "kicked", "disconnected"
}

// PortalJoinRequest is delivered to a portal owner when somebody calls
// portal.join_by_nick targeting their nickname.
type PortalJoinRequest struct {
	Type            string `json:"type"`
	RequestID       string `json:"request_id"` // owner echoes this in their response
	FromPeerID      string `json:"from_peer_id"`
	FromNickname    string `json:"from_nickname"`
	FromIPHashShort string `json:"from_ip_hash_short,omitempty"` // 6-char hex of remote IP for safety hint
}

// PortalKicked is sent to a peer who was kicked by the owner.
type PortalKicked struct {
	Type     string `json:"type"`
	PortalID string `json:"portal_id"`
}

// PortalLocked notifies all members that the lock state changed.
type PortalLocked struct {
	Type   string `json:"type"`
	Locked bool   `json:"locked"`
}

// PortalClosed notifies all members that the portal has been torn down
// (e.g. owner left and no one was promoted).
type PortalClosed struct {
	Type     string `json:"type"`
	PortalID string `json:"portal_id"`
	Reason   string `json:"reason,omitempty"`
}

// Error is the generic error response. The Code is machine-readable and
// stable; Message is human-readable and may change.
type Error struct {
	Type      string `json:"type"`
	Code      string `json:"code"`
	Message   string `json:"message"`
	RequestID string `json:"request_id,omitempty"`
}

// ----------------------------------------------------------------------------
// WebRTC handshake relay — bidirectional, server forwards verbatim
// ----------------------------------------------------------------------------

// WebRTCOffer carries a session description offer from one peer to another.
// On the wire from a client, "from" is unset and the server fills it in
// before forwarding to the target.
type WebRTCOffer struct {
	Type string `json:"type"`
	From string `json:"from,omitempty"`
	To   string `json:"to"`
	SDP  string `json:"sdp"`
}

// WebRTCAnswer carries a session description answer.
type WebRTCAnswer struct {
	Type string `json:"type"`
	From string `json:"from,omitempty"`
	To   string `json:"to"`
	SDP  string `json:"sdp"`
}

// WebRTCICE carries a single ICE candidate. Empty candidate string signals
// end-of-candidates (per the ICE spec).
type WebRTCICE struct {
	Type          string `json:"type"`
	From          string `json:"from,omitempty"`
	To            string `json:"to"`
	Candidate     string `json:"candidate"`
	SDPMid        string `json:"sdp_mid,omitempty"`
	SDPMLineIndex *int   `json:"sdp_mline_index,omitempty"`
}

// ----------------------------------------------------------------------------
// P2P data-channel messages (control channel) — used between peers, not server
// ----------------------------------------------------------------------------

type Ping struct {
	Type string `json:"type"`
	TS   int64  `json:"ts"` // unix milli
}

type Pong struct {
	Type   string `json:"type"`
	TS     int64  `json:"ts"`
	EchoTS int64  `json:"echo_ts"`
}

type Presence struct {
	Type   string `json:"type"`
	Status string `json:"status"` // active | idle | away
}

type ServiceExpose struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	Protocol string `json:"protocol"` // tcp | udp
	Port     int    `json:"port"`
}

type ServiceUnexpose struct {
	Type string `json:"type"`
	Port int    `json:"port"`
}

type ServiceListRequest struct {
	Type string `json:"type"`
}

type ServiceListResponse struct {
	Type     string          `json:"type"`
	Services []ServiceExpose `json:"services"`
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

// MustMarshal panics if marshaling fails. Use only with structs we control —
// they cannot fail to marshal, so any panic indicates a programming bug.
func MustMarshal(v any) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic("protocol: marshal failed for " + err.Error())
	}
	return b
}

// NewError constructs an Error message with the given code and message.
func NewError(code, msg, requestID string) Error {
	return Error{Type: TypeError, Code: code, Message: msg, RequestID: requestID}
}
