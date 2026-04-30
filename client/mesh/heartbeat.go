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

// sendPings emits a ping on every peer's control channel and tracks
// the send time so we can compute RTT when the pong returns.
//
// Locking: m.mu (RLock) is held only long enough to snapshot the peer
// pointers; per-peer state is then mutated under each Peer's own mu.
// This is what makes the function safe against concurrent writers (us,
// pruneStalePings, handleControl).
func (m *Manager) sendPings() {
	now := time.Now()
	ts := now.UnixMilli()
	payload, _ := json.Marshal(protocol.Ping{Type: protocol.TypePing, TS: ts})

	m.mu.RLock()
	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		peers = append(peers, p)
	}
	m.mu.RUnlock()

	for _, p := range peers {
		if p.conn == nil || !p.conn.ChannelOpen(peer.ChanControl) {
			continue
		}
		if err := p.conn.SendText(peer.ChanControl, string(payload)); err != nil {
			m.logger.Debug("ping send failed", "peer", p.ID, "err", err)
			continue
		}
		p.mu.Lock()
		p.outstanding[ts] = now
		p.mu.Unlock()
	}
}

func (m *Manager) pruneStalePings() {
	cutoff := time.Now().Add(-3 * m.cfg.HeartbeatInterval)
	m.mu.RLock()
	peers := make([]*Peer, 0, len(m.peers))
	for _, p := range m.peers {
		peers = append(peers, p)
	}
	m.mu.RUnlock()

	for _, p := range peers {
		p.mu.Lock()
		for ts, sentAt := range p.outstanding {
			if sentAt.Before(cutoff) {
				delete(p.outstanding, ts)
			}
		}
		p.mu.Unlock()
	}
}

// routeMessage dispatches an inbound peer message based on the channel
// it arrived on.
func (m *Manager) routeMessage(p *Peer, msg peer.Message) {
	p.bytesRecv.Add(int64(len(msg.Raw)))
	switch msg.Channel {
	case peer.ChanControl:
		m.handleControl(p, msg)
	case peer.ChanChat:
		m.handleChat(p, msg)
	case peer.ChanTransfer:
		m.handleTransfer(p, msg)
	case peer.ChanProxy:
		m.handleProxy(p, msg)
	}
}

func (m *Manager) handleChat(p *Peer, msg peer.Message) {
	// Decrypt if encryption was negotiated for this portal.
	plain, ok := m.openEnvelope(msg.Raw, msg.Text)
	if !ok {
		m.logger.Warn("chat decrypt failed", "from", p.ID)
		return
	}
	m.logger.Debug("chat", "from", p.ID, "text", string(plain))
	m.emit(MeshEvent{Type: EventChat, Peer: p, ChatText: string(plain)})
}

func (m *Manager) handleControl(p *Peer, msg peer.Message) {
	if msg.JSON == nil {
		return
	}
	t, _ := msg.JSON["type"].(string)
	switch t {
	case protocol.TypePing:
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

		p.mu.Lock()
		sent, ok := p.outstanding[ts]
		if ok {
			delete(p.outstanding, ts)
		}
		if ok {
			p.rtt = time.Since(sent)
			p.rttUpdatedAt = time.Now()
		}
		p.mu.Unlock()

		if !ok {
			return
		}
		m.emit(MeshEvent{Type: EventPeerRTT, Peer: p})

	case protocol.TypeServiceExpose:
		name, _ := msg.JSON["name"].(string)
		proto, _ := msg.JSON["protocol"].(string)
		portF, _ := msg.JSON["port"].(float64)
		port := int(portF)
		if port <= 0 || port > 65535 || (proto != "tcp" && proto != "udp") {
			return
		}
		p.mu.Lock()
		p.services[port] = ServiceAnnounce{Name: name, Protocol: proto, Port: port}
		p.mu.Unlock()
		m.emit(MeshEvent{Type: EventServiceAnnounce, Peer: p})

	case protocol.TypeServiceUnexpose:
		portF, _ := msg.JSON["port"].(float64)
		port := int(portF)
		p.mu.Lock()
		delete(p.services, port)
		p.mu.Unlock()
		m.emit(MeshEvent{Type: EventServiceAnnounce, Peer: p})
	}
}

// RTT returns the most recently measured round-trip time for a peer,
// or 0 if no pong has been received yet.
func (p *Peer) RTT() time.Duration {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.rtt
}

// Services returns a snapshot of the services this peer is exposing.
func (p *Peer) Services() []ServiceAnnounce {
	p.mu.Lock()
	defer p.mu.Unlock()
	out := make([]ServiceAnnounce, 0, len(p.services))
	for _, s := range p.services {
		out = append(out, s)
	}
	return out
}

// BytesSent returns the cumulative encrypted bytes we've written to
// this peer across all data channels since they joined.
func (p *Peer) BytesSent() int64 { return p.bytesSent.Load() }

// BytesRecv returns the cumulative encrypted bytes we've received
// from this peer.
func (p *Peer) BytesRecv() int64 { return p.bytesRecv.Load() }
