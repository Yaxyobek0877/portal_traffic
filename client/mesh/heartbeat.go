package mesh

import (
	"encoding/json"
	"time"

	"portal_traffic/shared/protocol"
	"portal_traffic_client/peer"
)

// heartbeatLoop runs in its own goroutine and sends a `ping` to every
// peer's control channel every Config.HeartbeatInterval. Pong responses
// are matched in routeMessage and update peer.rtt.
//
// Stale outstanding entries (>3× interval old) are pruned to avoid
// leaking memory if a peer drops without sending FINs.
func (m *Manager) heartbeatLoop() {
	t := time.NewTicker(m.cfg.HeartbeatInterval)
	defer t.Stop()
	for {
		select {
		case <-m.closed:
			return
		case <-t.C:
			m.sendPings()
			m.pruneStalePings()
		}
	}
}

func (m *Manager) sendPings() {
	now := time.Now()
	ts := now.UnixMilli()
	payload, _ := json.Marshal(protocol.Ping{Type: protocol.TypePing, TS: ts})

	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.peers {
		if p.conn == nil || !p.conn.ChannelOpen(peer.ChanControl) {
			continue
		}
		if err := p.conn.SendText(peer.ChanControl, string(payload)); err != nil {
			m.logger.Debug("ping send failed", "peer", p.ID, "err", err)
			continue
		}
		// Track outstanding under the peer's own lock proxy — we already
		// hold m.mu (RLock) which keeps the map stable; mutate the peer
		// map element directly. Concurrent heartbeats are serialised by
		// the ticker so we don't race with ourselves. If routeMessage
		// runs concurrently it operates on the same map; access is safe
		// because we never resize p.outstanding from multiple goroutines
		// (heartbeat writes; routeMessage deletes).
		p.outstanding[ts] = now
	}
}

func (m *Manager) pruneStalePings() {
	cutoff := time.Now().Add(-3 * m.cfg.HeartbeatInterval)
	m.mu.RLock()
	defer m.mu.RUnlock()
	for _, p := range m.peers {
		for ts, sentAt := range p.outstanding {
			if sentAt.Before(cutoff) {
				delete(p.outstanding, ts)
			}
		}
	}
}

// routeMessage dispatches an inbound peer message based on the channel
// it arrived on. Currently only the control channel does anything
// machine-meaningful here; chat / transfer / proxy bubble up via
// future hooks (Phase 4+).
func (m *Manager) routeMessage(p *Peer, msg peer.Message) {
	switch msg.Channel {
	case peer.ChanControl:
		m.handleControl(p, msg)
	case peer.ChanChat:
		m.logger.Debug("chat msg", "from", p.ID, "text", string(msg.Raw))
	}
}

func (m *Manager) handleControl(p *Peer, msg peer.Message) {
	if msg.JSON == nil {
		return
	}
	t, _ := msg.JSON["type"].(string)
	switch t {
	case protocol.TypePing:
		// Echo back as pong.
		echo, _ := msg.JSON["ts"].(float64)
		pong, _ := json.Marshal(protocol.Pong{
			Type:   protocol.TypePong,
			TS:     time.Now().UnixMilli(),
			EchoTS: int64(echo),
		})
		_ = p.conn.SendText(peer.ChanControl, string(pong))

	case protocol.TypePong:
		echo, _ := msg.JSON["echo_ts"].(float64)
		ts := int64(echo)
		m.mu.Lock()
		sent, ok := p.outstanding[ts]
		if ok {
			delete(p.outstanding, ts)
		}
		m.mu.Unlock()
		if !ok {
			return
		}
		rtt := time.Since(sent)
		m.mu.Lock()
		p.rtt = rtt
		p.rttUpdatedAt = time.Now()
		m.mu.Unlock()
		m.emit(MeshEvent{Type: EventPeerRTT, Peer: p})
	}
}

// RTT returns the most recently measured round-trip time for a peer,
// or 0 if no pong has been received yet.
func (p *Peer) RTT() time.Duration { return p.rtt }
