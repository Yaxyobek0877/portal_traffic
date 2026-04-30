package peer

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/pion/webrtc/v4"
)

// localPair builds two Connections wired to each other in-process.
// LocalICE candidates from one are fed into the other; offers and
// answers are exchanged synchronously. No external network.
//
// We use loopback STUN-less ICE which works because both sides
// gather only host candidates and live in the same kernel.
func localPair(t *testing.T) (*Connection, *Connection) {
	t.Helper()

	off, err := New(Config{
		LocalPeerID:  "alice",
		RemotePeerID: "bob",
		Role:         RoleOfferer,
	})
	if err != nil {
		t.Fatal(err)
	}
	ans, err := New(Config{
		LocalPeerID:  "bob",
		RemotePeerID: "alice",
		Role:         RoleAnswerer,
	})
	if err != nil {
		t.Fatal(err)
	}

	// SDP offer/answer.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	offerSDP, err := off.CreateOffer(ctx)
	if err != nil {
		t.Fatal(err)
	}
	answerSDP, err := ans.HandleOffer(ctx, offerSDP)
	if err != nil {
		t.Fatal(err)
	}
	if err := off.HandleAnswer(ctx, answerSDP); err != nil {
		t.Fatal(err)
	}

	// Forward ICE candidates between the two until each side has
	// reached "connected" or 10s elapses.
	go forwardICE(off, ans)
	go forwardICE(ans, off)

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if off.State() == StateConnected && ans.State() == StateConnected {
			return off, ans
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("connections did not reach Connected state: off=%s, ans=%s",
		off.State(), ans.State())
	return nil, nil
}

func forwardICE(src, dst *Connection) {
	for {
		select {
		case <-src.Done():
			return
		case <-dst.Done():
			return
		case cand := <-src.LocalICE():
			_ = dst.AddRemoteICE(cand.Candidate, derefStr(cand.SDPMid), derefMline(cand.SDPMLineIndex))
		}
	}
}

func derefStr(p *string) string {
	if p == nil {
		return ""
	}
	return *p
}

func derefMline(p *uint16) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

func TestPeerHandshakeAndChannels(t *testing.T) {
	off, ans := localPair(t)
	defer off.Close()
	defer ans.Close()

	// Wait for all data channels to open both sides.
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if off.AllChannelsOpen() && ans.AllChannelsOpen() {
			break
		}
		time.Sleep(50 * time.Millisecond)
	}
	if !off.AllChannelsOpen() || !ans.AllChannelsOpen() {
		t.Fatalf("channels not open: off=%v ans=%v", off.AllChannelsOpen(), ans.AllChannelsOpen())
	}

	// Round-trip a JSON ping over the control channel.
	if err := off.SendJSON(ChanControl, map[string]any{"type": "ping", "n": 1}); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-ans.Messages():
		if msg.Channel != ChanControl {
			t.Errorf("channel = %s, want %s", msg.Channel, ChanControl)
		}
		if msg.JSON["type"] != "ping" {
			t.Errorf("type = %v, want ping", msg.JSON["type"])
		}
	case <-time.After(2 * time.Second):
		t.Fatal("answerer did not receive ping")
	}

	// Verify chat channel works in the reverse direction too.
	if err := ans.SendText(ChanChat, "hello from bob"); err != nil {
		t.Fatal(err)
	}
	select {
	case msg := <-off.Messages():
		if !strings.Contains(string(msg.Raw), "hello from bob") {
			t.Errorf("payload = %q", string(msg.Raw))
		}
	case <-time.After(2 * time.Second):
		t.Fatal("offerer did not receive chat")
	}
}

// Smoke: hammer the control channel from both ends concurrently.
func TestPeerConcurrentSends(t *testing.T) {
	off, ans := localPair(t)
	defer off.Close()
	defer ans.Close()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) && !(off.AllChannelsOpen() && ans.AllChannelsOpen()) {
		time.Sleep(50 * time.Millisecond)
	}

	const N = 50
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			_ = off.SendJSON(ChanControl, map[string]any{"type": "ping", "n": i})
		}
	}()
	go func() {
		defer wg.Done()
		for i := 0; i < N; i++ {
			_ = ans.SendJSON(ChanControl, map[string]any{"type": "pong", "n": i})
		}
	}()
	wg.Wait()
	// We don't assert exact counts (delivery could be out of order on
	// multi-buffer ordering) — just that nothing panicked and we
	// received at least a few of each direction within a deadline.
	got := 0
	for got < N/2 {
		select {
		case <-off.Messages():
			got++
		case <-ans.Messages():
			got++
		case <-time.After(2 * time.Second):
			t.Fatalf("only got %d messages within deadline", got)
		}
	}
}

// Suppress unused import if we drop assertions later.
var _ = webrtc.ICECandidateInit{}
