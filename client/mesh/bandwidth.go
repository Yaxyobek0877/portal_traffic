package mesh

// Active bandwidth probe.
//
// On the sender side: open a JSON "bw_start" frame on the control
// channel, then stream 32 KB binary chunks for a fixed duration, then
// send "bw_done" and wait for "bw_ack" carrying the receiver-measured
// bytes/elapsed. We compute Mbps from the receiver's numbers — that's
// the actual throughput; the sender's clock alone would only see the
// SCTP queue insert time, which is much faster than the wire.
//
// On the receiver side: a binary frame with the bw magic prefix is
// counted and dropped (no upper-layer dispatch). At "bw_done" we send
// the running counts back. This avoids reusing the chat / transfer
// channel (which carry app-encrypted payload) and keeps the test
// orthogonal to user data.

import (
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"portal_traffic_client/peer"
)

const (
	// Magic bytes prefixing every bandwidth-test binary chunk on the
	// control channel. Cheap discriminator vs. accidental binary frames
	// (none today; gives us headroom for future ones).
	bwMagic0 byte = 0xBB
	bwMagic1 byte = 0x00

	// 32 KB chunk: WebRTC SCTP's default max message is 64 KB; staying
	// under it avoids fragmentation. 32 KB also lines up with the
	// transfer.chunkSize the rest of the app uses for similar reasons.
	bwChunkSize = 32 * 1024

	// Cap how long we let one probe run — long enough to see steady
	// state on 1+ Gbps links, short enough not to monopolise the channel.
	bwDuration = 3 * time.Second

	// Soft cap on outstanding SCTP buffer before we sleep. pion lets you
	// queue arbitrarily, but past a few MB we're just measuring the
	// sender's RAM, not the wire.
	bwMaxBuffered = 4 * 1024 * 1024
)

// BandwidthResult is what the API returns to the UI.
type BandwidthResult struct {
	PeerID     string  `json:"peerId"`
	Mbps       float64 `json:"mbps"`
	BytesSent  int64   `json:"bytesSent"`
	DurationMs float64 `json:"durationMs"`
}

// per-side state. All access guarded by Manager.bwMu.

// bwInbound: receiver tracks how many bytes it's seen for an in-flight test.
type bwInbound struct {
	id      string
	bytesIn int64
	started time.Time
}

// bwOutboundAckCh: sender awaits a single bw_ack for the test it kicked off.
type bwOutboundAckCh chan BandwidthResult

// MeasureBandwidth runs an active probe with peerID and blocks until
// the result lands or the test times out. Returns Mbps measured by the
// receiver (the only honest number — SCTP queue inserts are faster
// than the wire).
func (m *Manager) MeasureBandwidth(peerID string) (BandwidthResult, error) {
	m.mu.RLock()
	p := m.peers[peerID]
	m.mu.RUnlock()
	if p == nil {
		return BandwidthResult{}, errors.New("peer topilmadi")
	}
	if p.conn == nil {
		return BandwidthResult{}, errors.New("peer hali ulanmagan")
	}

	id := strconv.FormatInt(time.Now().UnixNano(), 36)
	ack := make(bwOutboundAckCh, 1)
	m.bwMu.Lock()
	if m.bwOut == nil {
		m.bwOut = map[string]bwOutboundAckCh{}
	}
	m.bwOut[peerID] = ack
	m.bwMu.Unlock()
	defer func() {
		m.bwMu.Lock()
		delete(m.bwOut, peerID)
		m.bwMu.Unlock()
	}()

	startMsg, _ := json.Marshal(map[string]any{
		"type": "bw_start",
		"id":   id,
	})
	if err := p.conn.SendText(peer.ChanControl, string(startMsg)); err != nil {
		return BandwidthResult{}, fmt.Errorf("bw_start: %w", err)
	}

	chunk := make([]byte, bwChunkSize+2)
	chunk[0] = bwMagic0
	chunk[1] = bwMagic1
	if _, err := rand.Read(chunk[2:]); err != nil {
		return BandwidthResult{}, fmt.Errorf("rand: %w", err)
	}

	var bytesSent int64
	deadline := time.Now().Add(bwDuration)
	for time.Now().Before(deadline) {
		// Soft backpressure: if the channel's outbound buffer is huge,
		// pause briefly so we don't OOM ourselves with queued chunks
		// that the wire hasn't drained.
		if buf := p.conn.BufferedAmount(peer.ChanControl); buf > bwMaxBuffered {
			time.Sleep(2 * time.Millisecond)
			continue
		}
		if err := p.conn.SendBinary(peer.ChanControl, chunk); err != nil {
			return BandwidthResult{}, fmt.Errorf("bw chunk: %w", err)
		}
		bytesSent += int64(len(chunk))
	}

	doneMsg, _ := json.Marshal(map[string]any{
		"type":       "bw_done",
		"id":         id,
		"bytes_sent": bytesSent,
	})
	if err := p.conn.SendText(peer.ChanControl, string(doneMsg)); err != nil {
		return BandwidthResult{}, fmt.Errorf("bw_done: %w", err)
	}

	select {
	case res := <-ack:
		res.PeerID = peerID
		res.BytesSent = bytesSent
		return res, nil
	case <-time.After(10 * time.Second):
		return BandwidthResult{}, errors.New("bandwidth probe timeout")
	}
}

// handleBandwidthChunk is called from routeMessage for binary control
// frames matching the bw magic. We drop the bytes after counting them.
func (m *Manager) handleBandwidthChunk(peerID string, frame []byte) {
	m.bwMu.Lock()
	defer m.bwMu.Unlock()
	if m.bwIn == nil {
		return
	}
	if sess, ok := m.bwIn[peerID]; ok {
		sess.bytesIn += int64(len(frame))
	}
}

// handleBandwidthControl handles bw_start / bw_done / bw_ack on the
// control channel.
func (m *Manager) handleBandwidthControl(p *Peer, t string, msg map[string]any) {
	switch t {
	case "bw_start":
		id, _ := msg["id"].(string)
		m.bwMu.Lock()
		if m.bwIn == nil {
			m.bwIn = map[string]*bwInbound{}
		}
		m.bwIn[p.ID] = &bwInbound{id: id, started: time.Now()}
		m.bwMu.Unlock()

	case "bw_done":
		m.bwMu.Lock()
		sess := m.bwIn[p.ID]
		delete(m.bwIn, p.ID)
		m.bwMu.Unlock()
		if sess == nil {
			return
		}
		elapsed := time.Since(sess.started).Seconds()
		mbps := 0.0
		if elapsed > 0 {
			mbps = float64(sess.bytesIn) * 8 / elapsed / 1e6
		}
		ack, _ := json.Marshal(map[string]any{
			"type":       "bw_ack",
			"id":         sess.id,
			"bytes_in":   sess.bytesIn,
			"elapsed_ms": elapsed * 1000,
			"mbps":       mbps,
		})
		_ = p.conn.SendText(peer.ChanControl, string(ack))

	case "bw_ack":
		m.bwMu.Lock()
		ch := m.bwOut[p.ID]
		m.bwMu.Unlock()
		if ch == nil {
			return
		}
		bytesIn, _ := msg["bytes_in"].(float64)
		elapsed, _ := msg["elapsed_ms"].(float64)
		mbps, _ := msg["mbps"].(float64)
		select {
		case ch <- BandwidthResult{
			Mbps:       mbps,
			DurationMs: elapsed,
			BytesSent:  int64(bytesIn),
		}:
		default:
		}
	}
}

