package protocol

import (
	"encoding/json"
	"testing"
)

func TestTypeOf(t *testing.T) {
	cases := []struct {
		raw     string
		want    string
		wantErr bool
	}{
		{`{"type":"portal.create","nickname":"alice"}`, "portal.create", false},
		{`{"type":"webrtc.offer","to":"abc","sdp":"v=0..."}`, "webrtc.offer", false},
		{`{"nickname":"alice"}`, "", true},
		{`not json`, "", true},
	}
	for _, c := range cases {
		got, err := TypeOf([]byte(c.raw))
		if c.wantErr && err == nil {
			t.Errorf("TypeOf(%q) expected error, got %q", c.raw, got)
		}
		if !c.wantErr && got != c.want {
			t.Errorf("TypeOf(%q) = %q, want %q (err=%v)", c.raw, got, c.want, err)
		}
	}
}

func TestRoundTripPortalCreate(t *testing.T) {
	msg := PortalCreate{Type: TypePortalCreate, Nickname: "alice", PublicNick: true, Capacity: 8}
	b, err := json.Marshal(msg)
	if err != nil {
		t.Fatal(err)
	}
	var got PortalCreate
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatal(err)
	}
	if got != msg {
		t.Errorf("roundtrip mismatch: %+v vs %+v", got, msg)
	}
}

func TestErrorHelper(t *testing.T) {
	e := NewError(ErrPortalNotFound, "no such portal", "req-1")
	if e.Type != TypeError || e.Code != ErrPortalNotFound || e.RequestID != "req-1" {
		t.Errorf("NewError produced unexpected struct: %+v", e)
	}
}
