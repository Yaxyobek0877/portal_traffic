package proxy

import (
	"bytes"
	"context"
	"net"
	"testing"
	"time"
)

// TestUDPRoundTrip:
//   - B exposes a fake UDP echo on localhost:<port>.
//   - A dials B's <port> via UDP proxy, getting a local UDP socket.
//   - A sends a datagram to its local socket, expects the echoed
//     reply to come back to the same local source addr.
func TestUDPRoundTrip(t *testing.T) {
	p := newPair(t)
	defer p.a.Close()
	defer p.b.Close()

	// UDP echo server on B's host.
	echo, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer echo.Close()
	go func() {
		buf := make([]byte, 4096)
		for {
			_ = echo.SetReadDeadline(time.Now().Add(2 * time.Second))
			n, src, err := echo.ReadFromUDP(buf)
			if err != nil {
				return
			}
			reply := append([]byte("pong-"), buf[:n]...)
			_, _ = echo.WriteToUDP(reply, src)
		}
	}()

	port := echo.LocalAddr().(*net.UDPAddr).Port
	p.b.Expose(port)

	conn, err := p.a.DialUDP(context.Background(), "b", port, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	// Local "client": separate UDP socket sending to A's listener.
	cli, err := net.DialUDP("udp", nil, conn.LocalAddr().(*net.UDPAddr))
	if err != nil {
		t.Fatal(err)
	}
	defer cli.Close()
	if _, err := cli.Write([]byte("ping")); err != nil {
		t.Fatal(err)
	}

	_ = cli.SetReadDeadline(time.Now().Add(3 * time.Second))
	got := make([]byte, 4096)
	n, err := cli.Read(got)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got[:n], []byte("pong-ping")) {
		t.Errorf("got %q, want %q", got[:n], "pong-ping")
	}
}

// TestUDPMultipleSources: two distinct local source addrs should produce
// two distinct streams, replies should route back correctly.
func TestUDPMultipleSources(t *testing.T) {
	p := newPair(t)
	defer p.a.Close()
	defer p.b.Close()

	echo, _ := net.ListenUDP("udp", &net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 0})
	defer echo.Close()
	go func() {
		buf := make([]byte, 4096)
		for {
			_ = echo.SetReadDeadline(time.Now().Add(3 * time.Second))
			n, src, err := echo.ReadFromUDP(buf)
			if err != nil {
				return
			}
			_, _ = echo.WriteToUDP(buf[:n], src)
		}
	}()

	port := echo.LocalAddr().(*net.UDPAddr).Port
	p.b.Expose(port)

	conn, err := p.a.DialUDP(context.Background(), "b", port, "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer conn.Close()

	target := conn.LocalAddr().(*net.UDPAddr)
	for i := 0; i < 2; i++ {
		cli, err := net.DialUDP("udp", nil, target)
		if err != nil {
			t.Fatal(err)
		}
		payload := []byte{byte('A' + i)}
		if _, err := cli.Write(payload); err != nil {
			t.Fatal(err)
		}
		_ = cli.SetReadDeadline(time.Now().Add(3 * time.Second))
		buf := make([]byte, 64)
		n, err := cli.Read(buf)
		if err != nil {
			t.Fatalf("source %d read: %v", i, err)
		}
		if n != 1 || buf[0] != payload[0] {
			t.Errorf("source %d: got %v, want %v", i, buf[:n], payload)
		}
		cli.Close()
	}
}
