package storage

import "time"

// HistoryEntry is one row in portal_history.
type HistoryEntry struct {
	ID       int64     `json:"id"`
	PortalID string    `json:"portalId"`
	Code     string    `json:"code"`
	Nickname string    `json:"nickname"`
	IsOwner  bool      `json:"isOwner"`
	JoinedAt time.Time `json:"joinedAt"`
	LastSeen time.Time `json:"lastSeen"`
}

// HistoryMaxRows caps how many we keep — older entries are GC'd
// when AddHistory is called.
const HistoryMaxRows = 50

// AddHistory upserts a (portal_id, nickname, is_owner) tuple. If the
// tuple already exists we just bump last_seen; otherwise we insert a
// new row and trim to HistoryMaxRows.
func (s *Store) AddHistory(e HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := Now().Unix()
	owner := 0
	if e.IsOwner {
		owner = 1
	}
	// Try update first.
	res, err := s.db.Exec(`
		UPDATE portal_history
		SET last_seen = ?, code = ?, nickname = ?
		WHERE portal_id = ? AND is_owner = ?
	`, now, e.Code, e.Nickname, e.PortalID, owner)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows > 0 {
		return nil
	}
	// Insert new row.
	_, err = s.db.Exec(`
		INSERT INTO portal_history(portal_id, code, nickname, is_owner, joined_at, last_seen)
		VALUES(?, ?, ?, ?, ?, ?)
	`, e.PortalID, e.Code, e.Nickname, owner, now, now)
	if err != nil {
		return err
	}
	// Trim oldest beyond HistoryMaxRows.
	_, err = s.db.Exec(`
		DELETE FROM portal_history
		WHERE id NOT IN (
			SELECT id FROM portal_history ORDER BY last_seen DESC LIMIT ?
		)
	`, HistoryMaxRows)
	return err
}

// RecentHistory returns the most-recent N entries, newest first.
func (s *Store) RecentHistory(n int) ([]HistoryEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if n <= 0 {
		n = 10
	}
	rows, err := s.db.Query(`
		SELECT id, portal_id, code, nickname, is_owner, joined_at, last_seen
		FROM portal_history
		ORDER BY last_seen DESC
		LIMIT ?
	`, n)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []HistoryEntry
	for rows.Next() {
		var (
			e        HistoryEntry
			owner    int
			joined   int64
			lastSeen int64
		)
		if err := rows.Scan(&e.ID, &e.PortalID, &e.Code, &e.Nickname, &owner, &joined, &lastSeen); err != nil {
			return nil, err
		}
		e.IsOwner = owner == 1
		e.JoinedAt = time.Unix(joined, 0)
		e.LastSeen = time.Unix(lastSeen, 0)
		out = append(out, e)
	}
	return out, rows.Err()
}

// ClearHistory removes every history row.
func (s *Store) ClearHistory() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM portal_history`)
	return err
}
