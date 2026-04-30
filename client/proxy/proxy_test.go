package proxy

import (
	"bytes"
	"context"
	"io"
	"net"
	"sync"
	"testing"
	"time"
)

// In-process loopback mesh: two Forwarders share a fake transport so
// frames sent by one are delivered to the other. Lets us exercise the
// full open/data/close protocol without running a WebRTC stack.

type pair struct {
	a, b *Forwarder
}

// fakeMesh approximates a real ordered+reliable WebRTC data channel:
// frames written by goroutine A arrive at peer B in order. We use a
// single-goroutine queue per direction to enforce that — concurrent
// SendProxyFrame calls would otherwise race and the proxy's protocol
// (which assumes DATA before CLOSE) breaks.
type fakeMesh struct {
	peerID string
	other  *Forwarder
	queue  chan []byte
	done   chan struct{}
}

func newFakeMesh(peerID string) *fakeMesh {
	return &fakeMesh{
		peerID: peerID,
		queue:  make(chan []byte, 256),
		done:   make(chan struct{}),
	}
}

func (f *fakeMesh) start() {
	go func() {
		for {
			select {
			case <-f.done:
				return
			case buf := <-f.queue:
				f.other.HandleFrame(f.peerID, buf)
			}
		}
	}()
}

func (f *fakeMesh) stop() {
	select {
	case <-f.done:
	default:
		close(f.done)
	}
}

func (f *fakeMesh) SendProxyFrame(peerID string, payload []byte) error {
	buf := make([]byte, len(payload))
	copy(buf, payload)
	select {
	case f.queue <- buf:
	case <-f.done:
	}
	return nil
}

func newPair(t *testing.T) *pair {
	t.Helper()
	p := &pair{}
	meshA := newFakeMesh("a")
	meshB := newFakeMesh("b")
	p.a = New(meshA, nil)
	p.b = New(meshB, nil)
	meshA.other = p.b
	meshB.other = p.a
	meshA.start()
	meshB.start()
	t.Cleanup(func() { meshA.stop(); meshB.stop() })
	return p
}

// TestProxyTCPRoundtrip:
//   - B exposes a fake echo server on localhost:<port>
//   - A dials B's <port> via its proxy
//   - A connects to its local listener, writes "ping", expects "pong-ping"
func TestProxyTCPRoundtrip(t *testing.T) {
	p := newPair(t)
	defer p.a.Close()
	defer p.b.Close()

	// Echo server with a "pong-" prefix so we can verify direction.
	echo, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()

	go func() {
		for {
			c, err := echo.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				n, err := c.Read(buf)
				if err != nil {
					return
				}
				_, _ = c.Write(append([]byte("pong-"), buf[:n]...))
			}(c)
		}
	}()

	// B exposes the echo port.
	echoPort := echo.Addr().(*net.TCPAddr).Port
	p.b.Expose(echoPort)

	// A dials B's echo port through the proxy.
	ln, err := p.a.Dial(context.Background(), "b", echoPort, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	// Connect locally to A's listener; writes traverse the fake mesh
	// to B, into the real echo server, back to A.
	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	if _, err := conn.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}

	_ = conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	got := make([]byte, 1024)
	n, err := conn.Read(got)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got[:n], []byte("pong-ping")) {
		t.Errorf("got %q, want %q", got[:n], "pong-ping")
	}
}

// TestProxyOpenRefused: dialer asks for a port that the host hasn't
// exposed; we expect the local connection to drop quickly without
// hanging.
func TestProxyOpenRefused(t *testing.T) {
	p := newPair(t)
	defer p.a.Close()
	defer p.b.Close()

	ln, err := p.a.Dial(context.Background(), "b", 9999, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer ln.Close()

	conn, err := net.Dial("tcp", ln.Addr().String())
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()
	_ = conn.SetReadDeadline(time.Now().Add(2 * time.Second))
	_, _ = conn.Write([]byte("hello"))
	buf := make([]byte, 64)
	n, _ := conn.Read(buf)
	if n != 0 {
		t.Errorf("expected dropped connection, got %d bytes: %q", n, buf[:n])
	}
}

// TestProxyConcurrentStreams: two simultaneous local connections
// should produce two independent streams.
func TestProxyConcurrentStreams(t *testing.T) {
	p := newPair(t)
	defer p.a.Close()
	defer p.b.Close()

	echo, _ := net.Listen("tcp", "127.0.0.1:0")
	defer echo.Close()
	go func() {
		for {
			c, err := echo.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				_, _ = io.Copy(c, c)
			}(c)
		}
	}()
	port := echo.Addr().(*net.TCPAddr).Port
	p.b.Expose(port)

	ln, _ := p.a.Dial(context.Background(), "b", port, "127.0.0.1:0")
	defer ln.Close()

	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		i := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			c, err := net.Dial("tcp", ln.Addr().String())
			if err != nil {
				t.Errorf("dial %d: %v", i, err)
				return
			}
			defer c.Close()
			payload := []byte{byte('A' + i)}
			_, _ = c.Write(payload)
			_ = c.SetReadDeadline(time.Now().Add(2 * time.Second))
			buf := make([]byte, 1)
			n, err := c.Read(buf)
			if err != nil || n != 1 || buf[0] != payload[0] {
				t.Errorf("stream %d: got %v err=%v want %v", i, buf[:n], err, payload)
			}
		}()
	}
	wg.Wait()
}
