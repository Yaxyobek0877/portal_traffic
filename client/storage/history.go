package storage

import (
	"database/sql"
	"time"
)

// HistoryEntry is one row in portal_history.
//
// Label is a user-assigned name for the portal — empty by default; the
// UI falls back to "Portal <portalId>" when it's blank. Survives across
// owner re-creations because owner rows are deduped by nickname (not by
// portal_id, which the server churns every disconnect).
type HistoryEntry struct {
	ID       int64     `json:"id"`
	PortalID string    `json:"portalId"`
	Code     string    `json:"code"`
	Nickname string    `json:"nickname"`
	IsOwner  bool      `json:"isOwner"`
	Label    string    `json:"label"`
	JoinedAt time.Time `json:"joinedAt"`
	LastSeen time.Time `json:"lastSeen"`
}

// HistoryMaxRows caps how many we keep — older entries are GC'd
// when AddHistory is called.
const HistoryMaxRows = 50

// AddHistory upserts a portal-history entry.
//
// Owner rows are special: each "create as owner" call gives the user a
// fresh portal_id from the server, because the signaling layer destroys
// the previous portal as soon as its owner disconnects. Keying dedup
// off portal_id therefore creates a new row per session, which the user
// experiences as the recent-portals list growing every time they pop
// in and out of their own portal. We instead key owner rows by
// nickname: one slot per (nickname, isOwner=1), updated in place with
// the freshest portal_id and code. The user's saved label stays
// stitched to that slot.
//
// Joiner rows (isOwner=0) keep the original key — (portal_id, isOwner).
// A user who joins the same friend's portal repeatedly should see one
// row, not one-per-rejoin.
func (s *Store) AddHistory(e HistoryEntry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := Now().Unix()
	owner := 0
	if e.IsOwner {
		owner = 1
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	// Owner rows dedupe by (nickname, isOwner=1) — see HistoryEntry
	// docstring. Joiner rows dedupe by (portal_id, isOwner=0).
	//
	// For owners we may already have multiple stale rows (a database
	// from before the dedup change can carry one row per past session).
	// The strategy is: pick the best survivor — prefer a row that
	// already carries a user-given label, otherwise the most recent —
	// update *that one* with the fresh portal_id/code, and delete every
	// other matching row in the same transaction. The user keeps the
	// label they typed across owner re-creations and doesn't see the
	// list grow on every restart.
	var keepID int64
	if e.IsOwner {
		err = tx.QueryRow(`
			SELECT id FROM portal_history
			WHERE nickname = ? AND is_owner = 1
			ORDER BY (label = '') ASC, last_seen DESC
			LIMIT 1
		`, e.Nickname).Scan(&keepID)
	} else {
		err = tx.QueryRow(`
			SELECT id FROM portal_history
			WHERE portal_id = ? AND is_owner = 0
			ORDER BY last_seen DESC
			LIMIT 1
		`, e.PortalID).Scan(&keepID)
	}
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if keepID > 0 {
		// Update the survivor.
		if e.IsOwner {
			if _, err = tx.Exec(`
				UPDATE portal_history
				SET portal_id = ?, code = ?, last_seen = ?
				WHERE id = ?
			`, e.PortalID, e.Code, now, keepID); err != nil {
				return err
			}
			// Drop any sibling rows for the same (nickname, isOwner=1)
			// — these are the ghosts from previous sessions on databases
			// migrated from the pre-dedup schema.
			if _, err = tx.Exec(`
				DELETE FROM portal_history
				WHERE nickname = ? AND is_owner = 1 AND id != ?
			`, e.Nickname, keepID); err != nil {
				return err
			}
		} else {
			if _, err = tx.Exec(`
				UPDATE portal_history
				SET last_seen = ?, code = ?, nickname = ?
				WHERE id = ?
			`, now, e.Code, e.Nickname, keepID); err != nil {
				return err
			}
			// Same cleanup for joiner rows that may have piled up under
			// older code.
			if _, err = tx.Exec(`
				DELETE FROM portal_history
				WHERE portal_id = ? AND is_owner = 0 AND id != ?
			`, e.PortalID, keepID); err != nil {
				return err
			}
		}
	} else {
		// No existing row → insert. Label stays at SQL default '' so
		// the row is unlabelled until the user renames via the UI.
		if _, err = tx.Exec(`
			INSERT INTO portal_history(portal_id, code, nickname, is_owner, joined_at, last_seen)
			VALUES(?, ?, ?, ?, ?, ?)
		`, e.PortalID, e.Code, e.Nickname, owner, now, now); err != nil {
			return err
		}
	}

	// Trim oldest beyond HistoryMaxRows.
	if _, err = tx.Exec(`
		DELETE FROM portal_history
		WHERE id NOT IN (
			SELECT id FROM portal_history ORDER BY last_seen DESC LIMIT ?
		)
	`, HistoryMaxRows); err != nil {
		return err
	}
	return tx.Commit()
}

// SetHistoryLabel assigns (or clears, when label is "") the user-given
// name on an existing history row. The label is purely cosmetic —
// nothing else in the app keys off it. Returns ErrNotFound when no row
// matches the id, so the UI can refresh in case the entry was deleted
// out from under it.
func (s *Store) SetHistoryLabel(id int64, label string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	res, err := s.db.Exec(
		`UPDATE portal_history SET label = ? WHERE id = ?`,
		label, id,
	)
	if err != nil {
		return err
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

// DeleteHistory removes one row by id. Used by the "remove from
// recent" affordance in the UI. No-op (returns nil) if the id doesn't
// exist — clicking remove twice should not surface an error.
func (s *Store) DeleteHistory(id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM portal_history WHERE id = ?`, id)
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
		SELECT id, portal_id, code, nickname, is_owner, label, joined_at, last_seen
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
		if err := rows.Scan(
			&e.ID, &e.PortalID, &e.Code, &e.Nickname,
			&owner, &e.Label, &joined, &lastSeen,
		); err != nil {
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
