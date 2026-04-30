package signaling

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"portal_traffic/shared/protocol"
)

// fakeServer mounts a tiny WebSocket handler that echoes a fixed
// PortalCreated response when it sees a portal.create request and
// echoes WebRTCOffers back to the sender. Just enough to exercise
// the client's encode/decode paths without needing the real server.
func fakeServer(t *testing.T) *httptest.Server {
	upgrader := websocket.Upgrader{}
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		ws, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade: %v", err)
			return
		}
		defer ws.Close()
		for {
			_, raw, err := ws.ReadMessage()
			if err != nil {
				return
			}
			t, _ := protocol.TypeOf(raw)
			switch t {
			case protocol.TypePortalCreate:
				resp := protocol.PortalCreated{
					Type: protocol.TypePortalCreated, PortalID: "111111", Code: "222222",
					PeerID: "test-peer", VirtualIP: "10.42.0.1", Capacity: 16,
				}
				b, _ := json.Marshal(resp)
				_ = ws.WriteMessage(websocket.TextMessage, b)
			case protocol.TypeWebRTCOffer:
				_ = ws.WriteMessage(websocket.TextMessage, raw)
			}
		}
	})
	return httptest.NewServer(mux)
}

func TestClientCreateRoundtrip(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()

	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	c, err := Dial(ctx, u.String(), nil)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer c.Close()

	if err := c.CreatePortal("alice", false, 0); err != nil {
		t.Fatalf("CreatePortal: %v", err)
	}

	select {
	case ev := <-c.Events():
		if ev.Created == nil {
			t.Fatalf("expected Created event, got %+v", ev)
		}
		if ev.Created.PortalID != "111111" {
			t.Errorf("portal_id = %s, want 111111", ev.Created.PortalID)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event received")
	}
}

func TestClientOfferEcho(t *testing.T) {
	srv := fakeServer(t)
	defer srv.Close()
	u, _ := url.Parse(srv.URL)
	u.Scheme = "ws"
	u.Path = "/ws"
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, err := Dial(ctx, u.String(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()

	if err := c.SendOffer("peer-x", "v=0... offer"); err != nil {
		t.Fatal(err)
	}
	select {
	case ev := <-c.Events():
		if ev.Offer == nil || ev.Offer.SDP != "v=0... offer" {
			t.Fatalf("expected echoed Offer, got %+v", ev)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("no event")
	}
}
