// Active-session persistence — the slot the App writes into when
// bringUpSession completes a CreatePortal / JoinPortal, and reads
// back from on Startup to auto-reconnect every portal that was
// running when the app last shut down (or crashed).
//
// Owner rows hold a stale portal_id between launches because the
// signaling server destroys an owner's portal the moment they
// disconnect — so 'restore' for owners is really 'recreate with
// the same nickname, get a fresh portal_id, update the row in
// place'. Joiner rows can attempt the original portal_id+code; if
// the portal is gone the row is deleted and the user just doesn't
// see that session come back.
//
// The table is intentionally distinct from portal_history. History
// is the long-running 'where have I been' log (kept for the
// dashboard's Recent strip); active_sessions is volatile state
// (kept only while the user wants the portal up). LeavePortal /
// LeaveAllPortals are the only paths that delete rows.

package storage

import "time"

type ActiveSessionRow struct {
	PortalID  string
	Code      string
	Nickname  string
	IsOwner   bool
	CreatedAt time.Time
	LastSeen  time.Time
}

// UpsertActiveSession inserts or updates a row keyed on portal_id.
// CreatedAt is set on first insert and preserved on update (the
// "I've been in this portal since…" timestamp); LastSeen bumps to
// now on every call.
func (s *Store) UpsertActiveSession(row ActiveSessionRow) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := Now().Unix()
	owner := 0
	if row.IsOwner {
		owner = 1
	}
	// SQLite UPSERT — INSERT … ON CONFLICT DO UPDATE keeps the original
	// created_at (we only re-set last_seen + the mutable fields).
	_, err := s.db.Exec(`
		INSERT INTO active_sessions(portal_id, code, nickname, is_owner, created_at, last_seen)
		VALUES(?, ?, ?, ?, ?, ?)
		ON CONFLICT(portal_id) DO UPDATE SET
			code      = excluded.code,
			nickname  = excluded.nickname,
			is_owner  = excluded.is_owner,
			last_seen = excluded.last_seen
	`, row.PortalID, row.Code, row.Nickname, owner, now, now)
	return err
}

// DeleteActiveSession removes one row. Used by LeavePortal /
// LeaveAllPortals (user explicitly said 'don't bring this back')
// and by the resume path when the portal is confirmed gone.
func (s *Store) DeleteActiveSession(portalID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM active_sessions WHERE portal_id = ?`, portalID)
	return err
}

// ListActiveSessions returns every row, newest last_seen first.
// Read by the resume path on app startup.
func (s *Store) ListActiveSessions() ([]ActiveSessionRow, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`
		SELECT portal_id, code, nickname, is_owner, created_at, last_seen
		FROM active_sessions
		ORDER BY last_seen DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ActiveSessionRow
	for rows.Next() {
		var (
			r        ActiveSessionRow
			owner    int
			created  int64
			lastSeen int64
		)
		if err := rows.Scan(
			&r.PortalID, &r.Code, &r.Nickname, &owner, &created, &lastSeen,
		); err != nil {
			return nil, err
		}
		r.IsOwner = owner == 1
		r.CreatedAt = time.Unix(created, 0)
		r.LastSeen = time.Unix(lastSeen, 0)
		out = append(out, r)
	}
	return out, rows.Err()
}

// ClearActiveSessions wipes the table. Used by ResetVault and
// LeaveAllPortals so a sign-out / wipe doesn't auto-reconnect to
// stale sessions on the next sign-in.
func (s *Store) ClearActiveSessions() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM active_sessions`)
	return err
}
