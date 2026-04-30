// Package proxy is the LAN-style TCP forwarder that turns Portal's
// `proxy` data channel into a stream multiplexer. It lets one peer
// "expose" a local TCP service (e.g. localhost:25565 for a Minecraft
// server) and another peer dial that service through the mesh as if
// it were on the same LAN.
//
// Threat model: the mesh is already encrypted by both DTLS and the
// portal-derived secretbox layer (see client/crypt). What this
// package adds is *streaming semantics* — chunking, accept/dial,
// connection lifecycle.
//
// Frame format on the proxy data channel (binary):
//
//	| type (1 byte) | stream_id (4 bytes BE) | payload... |
//
// Frame types:
//
//	0x01 OPEN     payload = "tcp:<port>" or "udp:<port>"  — initiate
//	0x02 OPEN_OK  payload = ""                             — accepted
//	0x03 OPEN_ERR payload = "<error message>"              — rejected
//	0x04 DATA     payload = bytes from one direction
//	0x05 CLOSE    payload = ""                             — half-close
//
// Stream IDs are picked by the dialer side; the host echoes them.
// Concurrent streams over a single proxy channel are independent.
//
// Limitations (v1):
//   - TCP only. UDP framing follows the same shape but isn't wired up
//     here yet (port 25565-style game servers are mostly TCP anyway).
//   - No flow-control beyond what the WebRTC data channel itself
//     provides. A truly fast sender to a slow consumer can buffer
//     unbounded; pion's underlying buffer eventually backpressures.
//   - One-shot streams: no reconnect-with-same-id semantics.
package proxy

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// Frame types — these are stable wire constants.
const (
	frameOpen    byte = 0x01
	frameOpenOK  byte = 0x02
	frameOpenErr byte = 0x03
	frameData    byte = 0x04
	frameClose   byte = 0x05
)

// Mesh is the subset of mesh.Manager that the proxy needs. We define
// it as an interface so the proxy package can be unit-tested with a
// fake mesh and so the import direction stays mesh → proxy never
// proxy → mesh internals.
type Mesh interface {
	SendProxyFrame(peerID string, payload []byte) error
}

// Logger is the log surface; defaults to slog.Default.
type Logger = *slog.Logger

// Forwarder is the per-peer-mesh proxy engine. Wire it up to a mesh
// once; from then on it manages exposes and dials for that mesh.
type Forwarder struct {
	mesh   Mesh
	logger *slog.Logger

	mu sync.Mutex
	// inbound streams accepted on our side (we are the host of the
	// exposed service). Keyed by (remotePeerID, streamID).
	hostStreams map[streamKey]*hostStream
	// outbound streams we initiated (we are the dialer). Keyed by
	// (remotePeerID, streamID).
	dialStreams map[streamKey]*dialStream

	// next stream ID counter (per Forwarder, monotonic).
	nextID uint32

	// active local TCP listeners we created via Dial(). Keyed by
	// "tcp:remotePeerID:remotePort" or "udp:remotePeerID:remotePort".
	listeners map[string]*localListener

	// udpDialStreams is the UDP counterpart of dialStreams. Each entry
	// maps a (peer, stream_id) to the source address of the original
	// local datagram so reply DATA frames find their way back.
	udpDialStreams map[streamKey]*udpDialStream
	udpListeners   []*localListener

	// expose registry: ports we offered to the mesh.
	exposed map[int]struct{}

	closed chan struct{}
}

// udpDialStream holds the bookkeeping for a single source→host UDP
// flow on the dialer side.
type udpDialStream struct {
	ll       *localListener
	src      *net.UDPAddr
	opened   chan error
	openOnce *sync.Once
	lastSeen *atomic.Int64
}

type streamKey struct {
	peerID string
	id     uint32
}

type hostStream struct {
	id       uint32
	peerID   string
	conn     net.Conn
	closed   atomic.Bool
	writeMu  sync.Mutex // serializes writes to conn (we may receive interleaved DATA frames)
	udp      bool
}

type dialStream struct {
	id       uint32
	peerID   string
	port     int
	conn     net.Conn // local TCP connection from a third-party app
	openCh   chan error
	openOnce sync.Once
	done     chan struct{} // closed when the stream ends in any direction
	doneOnce sync.Once
	closed   atomic.Bool
}

type localListener struct {
	listener   net.Listener
	udpConn    *net.UDPConn // populated for UDP listeners; nil for TCP
	remotePeer string
	remotePort int
	protocol   string // "tcp" | "udp"
	cancel     context.CancelFunc
}

// New constructs a Forwarder bound to the given mesh.
func New(m Mesh, logger *slog.Logger) *Forwarder {
	if logger == nil {
		logger = slog.Default()
	}
	return &Forwarder{
		mesh:           m,
		logger:         logger.With("component", "proxy"),
		hostStreams:    make(map[streamKey]*hostStream),
		dialStreams:    make(map[streamKey]*dialStream),
		udpDialStreams: make(map[streamKey]*udpDialStream),
		listeners:      make(map[string]*localListener),
		exposed:        make(map[int]struct{}),
		closed:         make(chan struct{}),
	}
}

// Close tears down everything: open streams, listeners, dial pumps.
func (f *Forwarder) Close() error {
	f.mu.Lock()
	select {
	case <-f.closed:
		f.mu.Unlock()
		return nil
	default:
		close(f.closed)
	}
	for _, s := range f.hostStreams {
		s.close()
	}
	for _, s := range f.dialStreams {
		s.close()
	}
	for _, l := range f.listeners {
		l.cancel()
		if l.listener != nil {
			_ = l.listener.Close()
		}
		if l.udpConn != nil {
			_ = l.udpConn.Close()
		}
	}
	f.hostStreams = nil
	f.dialStreams = nil
	f.udpDialStreams = nil
	f.listeners = nil
	f.udpListeners = nil
	f.mu.Unlock()
	return nil
}

// ----------------------------------------------------------------------------
// Expose — the host side
// ----------------------------------------------------------------------------

// Expose marks a local TCP port as available to the mesh. The actual
// service must already be listening on localhost:<port>; this just
// registers the port so peers' OPEN(tcp:port) requests get routed
// to a fresh net.Dial("tcp", "localhost:port").
//
// Idempotent. The mesh-level announcement (control channel) is the
// caller's job (mesh.Manager.AnnounceService); this is the local
// bookkeeping.
func (f *Forwarder) Expose(port int) {
	f.mu.Lock()
	f.exposed[port] = struct{}{}
	f.mu.Unlock()
}

// Unexpose removes the local registration.
func (f *Forwarder) Unexpose(port int) {
	f.mu.Lock()
	delete(f.exposed, port)
	f.mu.Unlock()
}

func (f *Forwarder) isExposed(port int) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	_, ok := f.exposed[port]
	return ok
}

// ----------------------------------------------------------------------------
// Dial — the client side: open a local TCP listener that pumps to
// remote peer's exposed port.
// ----------------------------------------------------------------------------

// Dial opens a local TCP listener on `localAddr` (e.g. "127.0.0.1:0"
// for an OS-chosen port). Every accepted connection on that listener
// is multiplexed over the proxy channel as a fresh stream targeting
// `remotePeerID`'s exposed `remotePort`.
//
// Returns the listener so the caller can read its Addr (when localAddr
// uses :0). Closing the listener stops accepting; existing streams
// continue until they end.
func (f *Forwarder) Dial(ctx context.Context, remotePeerID string, remotePort int, localAddr string) (net.Listener, error) {
	ln, err := net.Listen("tcp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("listen %s: %w", localAddr, err)
	}
	lctx, cancel := context.WithCancel(ctx)
	ll := &localListener{
		listener:   ln,
		remotePeer: remotePeerID,
		remotePort: remotePort,
		protocol:   "tcp",
		cancel:     cancel,
	}
	f.mu.Lock()
	f.listeners[fmt.Sprintf("tcp:%s:%d", remotePeerID, remotePort)] = ll
	f.mu.Unlock()

	go f.acceptLoop(lctx, ll)
	return ln, nil
}

// DialUDP is the UDP counterpart of Dial. The local listener is a UDP
// socket; each unique source address coming in gets its own multiplexed
// stream targeting `remotePeerID`'s exposed UDP `remotePort`.
//
// Returns the local UDP socket so the caller can read its address. Idle
// streams (no datagrams in either direction for 60s) are reaped to keep
// the stream-id table bounded.
func (f *Forwarder) DialUDP(ctx context.Context, remotePeerID string, remotePort int, localAddr string) (*net.UDPConn, error) {
	uaddr, err := net.ResolveUDPAddr("udp", localAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve %s: %w", localAddr, err)
	}
	conn, err := net.ListenUDP("udp", uaddr)
	if err != nil {
		return nil, fmt.Errorf("listen udp %s: %w", localAddr, err)
	}
	lctx, cancel := context.WithCancel(ctx)
	ll := &localListener{
		udpConn:    conn,
		remotePeer: remotePeerID,
		remotePort: remotePort,
		protocol:   "udp",
		cancel:     cancel,
	}
	f.mu.Lock()
	f.listeners[fmt.Sprintf("udp:%s:%d", remotePeerID, remotePort)] = ll
	f.mu.Unlock()

	go f.udpListenLoop(lctx, ll)
	return conn, nil
}

func (f *Forwarder) acceptLoop(ctx context.Context, ll *localListener) {
	for {
		conn, err := ll.listener.Accept()
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			f.logger.Debug("listener Accept error", "err", err)
			return
		}
		go f.handleLocalConn(ctx, ll, conn)
	}
}

// udpListenLoop reads datagrams off the local UDP socket and forwards
// them. It maintains a per-source-addr → stream_id map so replies from
// the host can be routed back to the correct local source. Streams
// idle out after udpIdleTimeout of inactivity in either direction.
func (f *Forwarder) udpListenLoop(ctx context.Context, ll *localListener) {
	type udpStream struct {
		id        uint32
		src       *net.UDPAddr
		opened    chan error
		openOnce  sync.Once
		lastSeen  atomic.Int64
		closed    atomic.Bool
	}

	const udpIdleTimeout = 60 * time.Second
	bySrc := map[string]*udpStream{}
	byID := map[uint32]*udpStream{}
	var mu sync.Mutex

	// Reap idle streams.
	go func() {
		t := time.NewTicker(15 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				cutoff := time.Now().Add(-udpIdleTimeout).UnixNano()
				mu.Lock()
				for k, s := range bySrc {
					if s.lastSeen.Load() < cutoff && !s.closed.Load() {
						s.closed.Store(true)
						_ = f.send(ll.remotePeer, frameClose, s.id, nil)
						delete(bySrc, k)
						delete(byID, s.id)
					}
				}
				mu.Unlock()
			}
		}
	}()

	// Register a UDP fanout so onData can route reply datagrams back
	// here. We piggyback on the dialStreams map by storing a tiny
	// adapter conn that wraps the UDP write.
	f.mu.Lock()
	f.udpListeners = append(f.udpListeners, ll)
	f.mu.Unlock()
	defer func() {
		f.mu.Lock()
		for i, x := range f.udpListeners {
			if x == ll {
				f.udpListeners = append(f.udpListeners[:i], f.udpListeners[i+1:]...)
				break
			}
		}
		f.mu.Unlock()
	}()

	buf := make([]byte, 64*1024)
	for {
		_ = ll.udpConn.SetReadDeadline(time.Now().Add(time.Second))
		n, srcAddr, err := ll.udpConn.ReadFromUDP(buf)
		if ctx.Err() != nil {
			return
		}
		if err != nil {
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				continue
			}
			return
		}
		key := srcAddr.String()
		mu.Lock()
		st := bySrc[key]
		if st == nil {
			id := atomic.AddUint32(&f.nextID, 1)
			st = &udpStream{id: id, src: srcAddr, opened: make(chan error, 1)}
			bySrc[key] = st
			byID[id] = st
			// Register so onOpenOK / onData can find it.
			f.mu.Lock()
			f.udpDialStreams[streamKey{ll.remotePeer, id}] = &udpDialStream{
				ll:       ll,
				src:      srcAddr,
				opened:   st.opened,
				openOnce: &st.openOnce,
				lastSeen: &st.lastSeen,
			}
			f.mu.Unlock()
			mu.Unlock()
			// Send OPEN udp:port; if it fails we drop the packet.
			_ = f.send(ll.remotePeer, frameOpen, id,
				[]byte(fmt.Sprintf("udp:%d", ll.remotePort)))
			// Wait briefly for OPEN_OK; if it doesn't arrive in 4s
			// we'll keep buffering data and trust the host to
			// dial and process retries.
			select {
			case err := <-st.opened:
				if err != nil {
					mu.Lock()
					delete(bySrc, key)
					delete(byID, id)
					mu.Unlock()
					f.mu.Lock()
					delete(f.udpDialStreams, streamKey{ll.remotePeer, id})
					f.mu.Unlock()
					continue
				}
			case <-time.After(4 * time.Second):
				// proceed; OPEN_OK may still come and DATA may
				// already be buffered on the host
			case <-ctx.Done():
				return
			}
			st.lastSeen.Store(time.Now().UnixNano())
			_ = f.send(ll.remotePeer, frameData, id, buf[:n])
		} else {
			st.lastSeen.Store(time.Now().UnixNano())
			id := st.id
			mu.Unlock()
			_ = f.send(ll.remotePeer, frameData, id, buf[:n])
		}
	}
}

func (f *Forwarder) handleLocalConn(ctx context.Context, ll *localListener, local net.Conn) {
	id := atomic.AddUint32(&f.nextID, 1)
	ds := &dialStream{
		id:     id,
		peerID: ll.remotePeer,
		port:   ll.remotePort,
		conn:   local,
		openCh: make(chan error, 1),
		done:   make(chan struct{}),
	}
	f.mu.Lock()
	f.dialStreams[streamKey{ll.remotePeer, id}] = ds
	f.mu.Unlock()

	defer func() {
		ds.close()
		f.mu.Lock()
		delete(f.dialStreams, streamKey{ll.remotePeer, id})
		f.mu.Unlock()
	}()

	// Send OPEN.
	if err := f.send(ll.remotePeer, frameOpen, id, []byte("tcp:"+strconv.Itoa(ll.remotePort))); err != nil {
		f.logger.Debug("OPEN send failed", "err", err)
		return
	}

	// Wait for OPEN_OK or OPEN_ERR.
	select {
	case err := <-ds.openCh:
		if err != nil {
			f.logger.Debug("remote rejected OPEN", "err", err)
			return
		}
	case <-time.After(15 * time.Second):
		f.logger.Debug("OPEN timeout")
		return
	case <-ctx.Done():
		return
	}

	// Pump local → remote. When this goroutine finishes (because the
	// local app closed its side), tell the host so it can flush.
	go func() {
		buf := make([]byte, 16*1024)
		for {
			n, err := local.Read(buf)
			if n > 0 {
				if errSend := f.send(ll.remotePeer, frameData, id, buf[:n]); errSend != nil {
					ds.markDone()
					return
				}
			}
			if err != nil {
				_ = f.send(ll.remotePeer, frameClose, id, nil)
				ds.markDone()
				return
			}
		}
	}()

	// Block until either side closes the stream. The remote → local
	// pump lives in HandleFrame.onData (writes straight to ds.conn);
	// onClose closes the conn (breaking our local Read above) and
	// signals done.
	select {
	case <-ds.done:
	case <-ctx.Done():
	}
}

// ----------------------------------------------------------------------------
// HandleFrame — invoked by mesh.Manager for every inbound proxy frame
// ----------------------------------------------------------------------------

// HandleFrame is the entry point from mesh.Manager. peerID is the
// sender's peer ID; payload is the already-decrypted frame.
//
// It implements both sides of the protocol since a mesh peer can be
// both host (accepting OPENs) and dialer (receiving OPEN_OKs / DATA).
func (f *Forwarder) HandleFrame(peerID string, payload []byte) {
	if len(payload) < 5 {
		return
	}
	t := payload[0]
	id := binary.BigEndian.Uint32(payload[1:5])
	body := payload[5:]

	switch t {
	case frameOpen:
		f.onOpen(peerID, id, body)
	case frameOpenOK:
		f.onOpenOK(peerID, id)
	case frameOpenErr:
		f.onOpenErr(peerID, id, string(body))
	case frameData:
		f.onData(peerID, id, body)
	case frameClose:
		f.onClose(peerID, id)
	}
}

func (f *Forwarder) onOpen(peerID string, id uint32, body []byte) {
	bs := string(body)
	switch {
	case len(bs) >= 5 && bs[:4] == "tcp:":
		f.onOpenTCP(peerID, id, bs[4:])
	case len(bs) >= 5 && bs[:4] == "udp:":
		f.onOpenUDP(peerID, id, bs[4:])
	default:
		_ = f.send(peerID, frameOpenErr, id, []byte("only tcp:/udp: supported"))
	}
}

func (f *Forwarder) onOpenTCP(peerID string, id uint32, portStr string) {
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		_ = f.send(peerID, frameOpenErr, id, []byte("bad port"))
		return
	}
	if !f.isExposed(port) {
		_ = f.send(peerID, frameOpenErr, id, []byte("port not exposed"))
		return
	}
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", port), 5*time.Second)
	if err != nil {
		_ = f.send(peerID, frameOpenErr, id, []byte(err.Error()))
		return
	}
	hs := &hostStream{id: id, peerID: peerID, conn: conn}
	f.mu.Lock()
	f.hostStreams[streamKey{peerID, id}] = hs
	f.mu.Unlock()
	if err := f.send(peerID, frameOpenOK, id, nil); err != nil {
		hs.close()
		return
	}
	go func() {
		buf := make([]byte, 16*1024)
		for {
			n, err := conn.Read(buf)
			if n > 0 {
				if errSend := f.send(peerID, frameData, id, buf[:n]); errSend != nil {
					break
				}
			}
			if err != nil {
				_ = f.send(peerID, frameClose, id, nil)
				break
			}
		}
		hs.close()
		f.mu.Lock()
		delete(f.hostStreams, streamKey{peerID, id})
		f.mu.Unlock()
	}()
}

// onOpenUDP creates a connected UDP socket to localhost:<port>. Each
// stream gets its own socket, which means responses naturally carry
// the right source port. Replies stream back as DATA frames.
func (f *Forwarder) onOpenUDP(peerID string, id uint32, portStr string) {
	port, err := strconv.Atoi(portStr)
	if err != nil || port <= 0 || port > 65535 {
		_ = f.send(peerID, frameOpenErr, id, []byte("bad port"))
		return
	}
	if !f.isExposed(port) {
		_ = f.send(peerID, frameOpenErr, id, []byte("port not exposed"))
		return
	}
	conn, err := net.DialTimeout("udp", fmt.Sprintf("127.0.0.1:%d", port), 3*time.Second)
	if err != nil {
		_ = f.send(peerID, frameOpenErr, id, []byte(err.Error()))
		return
	}
	hs := &hostStream{id: id, peerID: peerID, conn: conn, udp: true}
	f.mu.Lock()
	f.hostStreams[streamKey{peerID, id}] = hs
	f.mu.Unlock()
	if err := f.send(peerID, frameOpenOK, id, nil); err != nil {
		hs.close()
		return
	}
	go func() {
		buf := make([]byte, 64*1024)
		for {
			_ = conn.SetReadDeadline(time.Now().Add(60 * time.Second))
			n, err := conn.Read(buf)
			if hs.closed.Load() {
				return
			}
			if n > 0 {
				if errSend := f.send(peerID, frameData, id, buf[:n]); errSend != nil {
					break
				}
			}
			if err != nil {
				if ne, ok := err.(net.Error); ok && ne.Timeout() {
					// Idle — close stream, peer can reopen.
					_ = f.send(peerID, frameClose, id, nil)
					break
				}
				_ = f.send(peerID, frameClose, id, nil)
				break
			}
		}
		hs.close()
		f.mu.Lock()
		delete(f.hostStreams, streamKey{peerID, id})
		f.mu.Unlock()
	}()
}

func (f *Forwarder) onOpenOK(peerID string, id uint32) {
	f.mu.Lock()
	ds := f.dialStreams[streamKey{peerID, id}]
	uds := f.udpDialStreams[streamKey{peerID, id}]
	f.mu.Unlock()
	if ds != nil {
		ds.openOnce.Do(func() { ds.openCh <- nil })
	}
	if uds != nil {
		uds.openOnce.Do(func() { uds.opened <- nil })
	}
}

func (f *Forwarder) onOpenErr(peerID string, id uint32, msg string) {
	f.mu.Lock()
	ds := f.dialStreams[streamKey{peerID, id}]
	uds := f.udpDialStreams[streamKey{peerID, id}]
	f.mu.Unlock()
	if ds != nil {
		ds.openOnce.Do(func() { ds.openCh <- errors.New(msg) })
	}
	if uds != nil {
		uds.openOnce.Do(func() { uds.opened <- errors.New(msg) })
	}
}

func (f *Forwarder) onData(peerID string, id uint32, body []byte) {
	f.mu.Lock()
	hs := f.hostStreams[streamKey{peerID, id}]
	ds := f.dialStreams[streamKey{peerID, id}]
	uds := f.udpDialStreams[streamKey{peerID, id}]
	f.mu.Unlock()
	if hs != nil {
		// Either TCP or UDP — Conn.Write works in both cases.
		hs.writeMu.Lock()
		_, _ = hs.conn.Write(body)
		hs.writeMu.Unlock()
	}
	if ds != nil {
		_, _ = ds.conn.Write(body)
	}
	if uds != nil {
		// Reply datagram — write back to the original local source.
		uds.lastSeen.Store(time.Now().UnixNano())
		_, _ = uds.ll.udpConn.WriteToUDP(body, uds.src)
	}
}

func (f *Forwarder) onClose(peerID string, id uint32) {
	f.mu.Lock()
	hs := f.hostStreams[streamKey{peerID, id}]
	ds := f.dialStreams[streamKey{peerID, id}]
	uds := f.udpDialStreams[streamKey{peerID, id}]
	delete(f.hostStreams, streamKey{peerID, id})
	delete(f.dialStreams, streamKey{peerID, id})
	delete(f.udpDialStreams, streamKey{peerID, id})
	f.mu.Unlock()
	if hs != nil {
		hs.close()
	}
	if ds != nil {
		ds.openOnce.Do(func() { ds.openCh <- io.EOF })
		ds.close()
	}
	if uds != nil {
		uds.openOnce.Do(func() { uds.opened <- io.EOF })
	}
}

// ----------------------------------------------------------------------------
// Helpers
// ----------------------------------------------------------------------------

func (f *Forwarder) send(peerID string, t byte, id uint32, body []byte) error {
	buf := make([]byte, 5+len(body))
	buf[0] = t
	binary.BigEndian.PutUint32(buf[1:5], id)
	copy(buf[5:], body)
	return f.mesh.SendProxyFrame(peerID, buf)
}

func (s *hostStream) close() {
	if s.closed.Swap(true) {
		return
	}
	_ = s.conn.Close()
}

func (s *dialStream) close() {
	if s.closed.Swap(true) {
		return
	}
	_ = s.conn.Close()
	s.markDone()
}

func (s *dialStream) markDone() {
	s.doneOnce.Do(func() { close(s.done) })
}
