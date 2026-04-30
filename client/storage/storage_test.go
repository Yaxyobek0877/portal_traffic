package storage

import (
	"path/filepath"
	"testing"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	p := filepath.Join(t.TempDir(), "test.db")
	s, err := Open(p)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = s.Close() })
	return s
}

func TestSettingsRoundTrip(t *testing.T) {
	s := openTemp(t)
	if v := s.GetOr("missing", "fallback"); v != "fallback" {
		t.Errorf("GetOr = %q, want fallback", v)
	}
	if err := s.PutSetting(KeySignalingURL, "wss://example/ws"); err != nil {
		t.Fatal(err)
	}
	v, err := s.GetSetting(KeySignalingURL)
	if err != nil {
		t.Fatal(err)
	}
	if v != "wss://example/ws" {
		t.Errorf("got %q", v)
	}
}

func TestHistoryAddDeduplicates(t *testing.T) {
	s := openTemp(t)
	for i := 0; i < 3; i++ {
		if err := s.AddHistory(HistoryEntry{
			PortalID: "111111", Code: "222222", Nickname: "alice", IsOwner: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.RecentHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 deduplicated row, got %d", len(rows))
	}
}

func TestHistoryTrimToMaxRows(t *testing.T) {
	s := openTemp(t)
	for i := 0; i < HistoryMaxRows+5; i++ {
		if err := s.AddHistory(HistoryEntry{
			PortalID: rsId(i), Code: "000000", Nickname: "x", IsOwner: false,
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, _ := s.RecentHistory(HistoryMaxRows + 100)
	if len(rows) > HistoryMaxRows {
		t.Errorf("history not trimmed: %d rows", len(rows))
	}
}

func rsId(i int) string {
	const ds = "0123456789"
	out := []byte("000000")
	out[5] = ds[i%10]
	out[4] = ds[(i/10)%10]
	out[3] = ds[(i/100)%10]
	return string(out)
}
