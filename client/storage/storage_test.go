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

// Server gives a fresh portal_id every time the same user creates a
// portal as owner (the old portal is gone the moment they
// disconnected). The history list shouldn't grow one row per session;
// dedupe-by-nickname-when-owner consolidates them into a single slot
// pointing at the freshest id/code.
func TestHistoryOwnerDedupesByNickname(t *testing.T) {
	s := openTemp(t)
	for i, id := range []string{"111111", "222222", "333333"} {
		if err := s.AddHistory(HistoryEntry{
			PortalID: id, Code: "0000" + string(rune('0'+i)) + "0",
			Nickname: "alice", IsOwner: true,
		}); err != nil {
			t.Fatal(err)
		}
	}
	rows, err := s.RecentHistory(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 owner row, got %d", len(rows))
	}
	if rows[0].PortalID != "333333" {
		t.Errorf("expected freshest portal_id 333333, got %s", rows[0].PortalID)
	}
}

// Joiner rows rejoining the SAME friend's portal repeatedly should
// also collapse to one row, but they're keyed by portal_id (joiners
// don't churn IDs). Different portal_ids → different rows.
func TestHistoryJoinerDedupesByPortalID(t *testing.T) {
	s := openTemp(t)
	_ = s.AddHistory(HistoryEntry{PortalID: "111111", Code: "a", Nickname: "alice", IsOwner: false})
	_ = s.AddHistory(HistoryEntry{PortalID: "111111", Code: "a", Nickname: "alice", IsOwner: false})
	_ = s.AddHistory(HistoryEntry{PortalID: "222222", Code: "b", Nickname: "alice", IsOwner: false})
	rows, _ := s.RecentHistory(10)
	if len(rows) != 2 {
		t.Errorf("expected 2 distinct joiner rows, got %d", len(rows))
	}
}

// Renaming should survive an owner re-create: the user labels their
// portal "Game Night", leaves, comes back as owner with a fresh id,
// and the label is still there because the owner row is the same row.
func TestHistoryLabelSurvivesOwnerRecreate(t *testing.T) {
	s := openTemp(t)
	_ = s.AddHistory(HistoryEntry{PortalID: "111111", Code: "a", Nickname: "alice", IsOwner: true})
	rows, _ := s.RecentHistory(10)
	if len(rows) != 1 {
		t.Fatalf("setup: want 1 row, got %d", len(rows))
	}
	if err := s.SetHistoryLabel(rows[0].ID, "Game Night"); err != nil {
		t.Fatal(err)
	}

	// Owner re-creates; new portal_id from server.
	_ = s.AddHistory(HistoryEntry{PortalID: "222222", Code: "b", Nickname: "alice", IsOwner: true})
	rows, _ = s.RecentHistory(10)
	if len(rows) != 1 {
		t.Fatalf("post-recreate: want 1 row, got %d", len(rows))
	}
	if rows[0].PortalID != "222222" {
		t.Errorf("portal_id should refresh, got %s", rows[0].PortalID)
	}
	if rows[0].Label != "Game Night" {
		t.Errorf("label should survive, got %q", rows[0].Label)
	}
}

// Pre-dedup databases can carry multiple owner rows for the same
// nickname (one per past CreatePortal session). The next AddHistory
// call should collapse them into a single row, preserving any label
// that survived from the old behaviour.
func TestHistoryConsolidatesPreExistingOwnerDuplicates(t *testing.T) {
	s := openTemp(t)
	// Manually seed the table the way the old (buggy) code would: four
	// distinct owner rows, same nickname, no label except on the second.
	for _, id := range []string{"111111", "222222", "333333", "444444"} {
		now := Now().Unix()
		if _, err := s.db.Exec(`
			INSERT INTO portal_history(portal_id, code, nickname, is_owner, label, joined_at, last_seen)
			VALUES(?, ?, ?, 1, ?, ?, ?)
		`, id, "x", "alice", "", now, now); err != nil {
			t.Fatal(err)
		}
	}
	// Tag the second one with a label so we can assert it's the survivor.
	if _, err := s.db.Exec(
		`UPDATE portal_history SET label = ? WHERE portal_id = ?`,
		"Game Night", "222222",
	); err != nil {
		t.Fatal(err)
	}

	// Owner re-creates — fresh server-issued id.
	if err := s.AddHistory(HistoryEntry{
		PortalID: "555555", Code: "y", Nickname: "alice", IsOwner: true,
	}); err != nil {
		t.Fatal(err)
	}

	rows, _ := s.RecentHistory(20)
	if len(rows) != 1 {
		t.Fatalf("want 1 consolidated row, got %d", len(rows))
	}
	if rows[0].PortalID != "555555" {
		t.Errorf("want fresh portal_id 555555, got %s", rows[0].PortalID)
	}
	if rows[0].Label != "Game Night" {
		t.Errorf("want labelled survivor's label preserved, got %q", rows[0].Label)
	}
}

// SetHistoryLabel on a missing id returns ErrNotFound (so the UI can
// refresh) instead of silently succeeding.
func TestHistoryLabelMissingID(t *testing.T) {
	s := openTemp(t)
	if err := s.SetHistoryLabel(99999, "x"); err != ErrNotFound {
		t.Errorf("want ErrNotFound, got %v", err)
	}
}

// DeleteHistory removes one row but leaves siblings intact, and is a
// no-op when the id is gone (so click-twice doesn't error).
func TestHistoryDeleteOne(t *testing.T) {
	s := openTemp(t)
	_ = s.AddHistory(HistoryEntry{PortalID: "111111", Nickname: "a", IsOwner: false})
	_ = s.AddHistory(HistoryEntry{PortalID: "222222", Nickname: "a", IsOwner: false})
	rows, _ := s.RecentHistory(10)
	if err := s.DeleteHistory(rows[0].ID); err != nil {
		t.Fatal(err)
	}
	rows, _ = s.RecentHistory(10)
	if len(rows) != 1 {
		t.Errorf("want 1 row after delete, got %d", len(rows))
	}
	if err := s.DeleteHistory(99999); err != nil {
		t.Errorf("delete missing id should be no-op, got %v", err)
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
