package pcp

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/crypto/chacha20poly1305"
)

// PairSession is one peer's view of an encrypted channel between
// itself and exactly one other peer. Each peer in the portal mesh
// holds N PairSessions (one per other peer).
//
// Once both sides have run Derive with matching inputs, frames can
// be sealed with Seal and opened with Open. The underlying counters
// guard against replay; per-direction keys ensure that even if one
// direction's tag is forged, the other direction stays sealed.
type PairSession struct {
	// k_send / k_recv are the directional ChaCha20-Poly1305 keys.
	// 32 bytes each.
	kSend [32]byte
	kRecv [32]byte

	// Nonce prefix is the first 16 bytes; combined with the 8-byte
	// counter it produces a 24-byte XChaCha20-Poly1305 nonce.
	noncePrefixSend [16]byte
	noncePrefixRecv [16]byte

	// sendCounter is the next counter value to use when sealing.
	// Starts at 0; monotonically increasing, atomic.
	sendCounter atomic.Uint64

	// recvWindow tracks accepted counters to reject replays. We use a
	// simple "highest accepted" cursor: incoming counter must be >
	// recvHighest. Pure ordered streams (which our control/chat/proxy
	// channels are, by virtue of running over reliable WebRTC data
	// channels) need nothing fancier.
	//
	// recvSeenAny disambiguates the initial state: before the first
	// accepted frame, recvHighest is 0, but we still need to accept a
	// frame with counter==0. After acceptance, replays of counter==0
	// must be rejected.
	recvMu      sync.Mutex
	recvSeenAny bool
	recvHighest uint64
}

// DeriveSession constructs a PairSession from:
//
//   - sharedSecret: result of X25519(myEphPriv, peerEphPub)
//   - portalID: the 6-digit (or longer) portal id, used as HKDF salt
//   - myIdPub, peerIdPub: Ed25519 long-term public keys for both peers
//
// Both peers pass the same shared secret + portal id; the lexicographic
// ordering of the identity public keys determines who is "lo" and "hi"
// without any explicit role negotiation.
func DeriveSession(sharedSecret []byte, portalID string, myIdPub, peerIdPub []byte) (*PairSession, error) {
	if len(sharedSecret) == 0 {
		return nil, errors.New("pcp: shared secret empty")
	}
	if len(myIdPub) != PublicKeySize || len(peerIdPub) != PublicKeySize {
		return nil, fmt.Errorf("pcp: identity pubkeys must be %d bytes each", PublicKeySize)
	}

	loPK, hiPK := myIdPub, peerIdPub
	myIsLo := bytes.Compare(myIdPub, peerIdPub) < 0
	if !myIsLo {
		loPK, hiPK = peerIdPub, myIdPub
	}

	// HKDF-Extract: master = HKDF(salt=portalID, ikm=sharedSecret)
	// We achieve this with hkdf.New(... salt ...) but Go's stdlib API
	// only exposes Extract+Expand fused — using hkdfExpand here with
	// the portal id as part of the info string is equivalent for our
	// purposes given the input quality.
	salt := []byte("portal:" + portalID)
	prk := hkdfExtractAndExpand(sharedSecret, salt, []byte(VersionString+":master"), 32)

	// Per-direction keys.
	kLoToHi := hkdfExpandFixed(prk, []byte(VersionString+":pair:lo->hi"), 32)
	kHiToLo := hkdfExpandFixed(prk, []byte(VersionString+":pair:hi->lo"), 32)

	// Per-direction nonce prefixes.
	pLoToHi := hkdfExpandFixed(prk, []byte(VersionString+":nonce:lo->hi:"+string(loPK)+"->"+string(hiPK)), 16)
	pHiToLo := hkdfExpandFixed(prk, []byte(VersionString+":nonce:hi->lo:"+string(hiPK)+"->"+string(loPK)), 16)

	s := &PairSession{}
	if myIsLo {
		copy(s.kSend[:], kLoToHi)
		copy(s.kRecv[:], kHiToLo)
		copy(s.noncePrefixSend[:], pLoToHi)
		copy(s.noncePrefixRecv[:], pHiToLo)
	} else {
		copy(s.kSend[:], kHiToLo)
		copy(s.kRecv[:], kLoToHi)
		copy(s.noncePrefixSend[:], pHiToLo)
		copy(s.noncePrefixRecv[:], pLoToHi)
	}
	return s, nil
}

// Seal produces a wire-format frame containing the sealed plaintext.
// Wire format:  [counter (8 BE) || ciphertext+tag]
//
// The 8-byte counter is included in the wire output so the receiver
// can reconstruct the same nonce. The counter is also implicit in the
// nonce, so an attacker forging the counter still gets a tag failure.
func (s *PairSession) Seal(plaintext []byte) ([]byte, error) {
	counter := s.sendCounter.Add(1) - 1
	if counter >= maxFrameCounter {
		return nil, errors.New("pcp: send counter exhausted; rekey required")
	}

	aead, err := chacha20poly1305.NewX(s.kSend[:])
	if err != nil {
		return nil, fmt.Errorf("pcp: aead init: %w", err)
	}
	nonce := buildNonce(s.noncePrefixSend, counter)
	ct := aead.Seal(nil, nonce, plaintext, frameAD)

	// Wire: counter (8 BE) || ciphertext+tag
	out := make([]byte, 0, 8+len(ct))
	var counterBytes [8]byte
	binary.BigEndian.PutUint64(counterBytes[:], counter)
	out = append(out, counterBytes[:]...)
	out = append(out, ct...)
	return out, nil
}

// Open reverses Seal. Returns the plaintext, or an error if the tag
// fails or the counter has been seen before (replay).
func (s *PairSession) Open(frame []byte) ([]byte, error) {
	if len(frame) < 8+chacha20poly1305.Overhead {
		return nil, errors.New("pcp: frame too short")
	}
	counter := binary.BigEndian.Uint64(frame[:8])
	ct := frame[8:]

	// Replay check: counter must be strictly greater than the highest
	// counter we've previously accepted. (Reliable WebRTC data
	// channels guarantee in-order delivery, so we don't need a sliding
	// window.)
	s.recvMu.Lock()
	if s.recvSeenAny && counter <= s.recvHighest {
		s.recvMu.Unlock()
		return nil, errors.New("pcp: replay or out-of-order frame")
	}
	s.recvMu.Unlock()

	aead, err := chacha20poly1305.NewX(s.kRecv[:])
	if err != nil {
		return nil, fmt.Errorf("pcp: aead init: %w", err)
	}
	nonce := buildNonce(s.noncePrefixRecv, counter)
	pt, err := aead.Open(nil, nonce, ct, frameAD)
	if err != nil {
		return nil, fmt.Errorf("pcp: open: %w", err)
	}

	s.recvMu.Lock()
	s.recvSeenAny = true
	if counter > s.recvHighest {
		s.recvHighest = counter
	}
	s.recvMu.Unlock()

	return pt, nil
}

// SendCounter returns the current send counter (next counter to be
// used). Useful for diagnostics and rekey trigger logic.
func (s *PairSession) SendCounter() uint64 { return s.sendCounter.Load() }

// frameAD is the additional-authenticated-data string we mix into
// every PCP-1 frame. Including a fixed AD pins the frame to this
// protocol version, so a frame encrypted under future PCP-2 keys
// can never be misinterpreted as PCP-1.
var frameAD = []byte(VersionString + ":frame:v1")

// maxFrameCounter is the rekey threshold. After this many frames in
// one direction, a fresh DeriveSession must replace the current one.
// 2^48 ≈ 2.8 * 10^14 — never reached in any realistic Portal session,
// but bounded so we have a guarantee.
const maxFrameCounter uint64 = 1 << 48

// buildNonce concatenates the 16-byte direction prefix with the 8-byte
// counter to produce the 24-byte XChaCha20-Poly1305 nonce.
func buildNonce(prefix [16]byte, counter uint64) []byte {
	nonce := make([]byte, 24)
	copy(nonce[:16], prefix[:])
	binary.BigEndian.PutUint64(nonce[16:], counter)
	return nonce
}

// hkdfExtractAndExpand performs a one-shot HKDF (Extract + Expand)
// with explicit salt. We expose this small helper to keep the call
// sites in DeriveSession readable.
func hkdfExtractAndExpand(ikm, salt, info []byte, outLen int) []byte {
	out := make([]byte, outLen)
	r := hkdfNewWithSalt(ikm, salt, info)
	if _, err := r.Read(out); err != nil {
		// A read from hkdf only fails if you try to read more than
		// 255*HashLen bytes; we always ask for 32, which is fine.
		panic(fmt.Errorf("pcp: hkdf read: %w", err))
	}
	return out
}

// hkdfExpandFixed wraps hkdfExpand for a known-good outLen so call
// sites stay readable.
func hkdfExpandFixed(prk, info []byte, outLen int) []byte {
	r := hkdfNewWithSalt(prk, nil, info)
	out := make([]byte, outLen)
	if _, err := r.Read(out); err != nil {
		panic(fmt.Errorf("pcp: hkdf read: %w", err))
	}
	return out
}
