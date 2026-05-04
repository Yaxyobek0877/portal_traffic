// Package storage owns the local SQLite database. It persists three
// kinds of data across runs:
//
//   - Settings   — KV table for things like signaling URL and nickname.
//   - History    — last N portals the user created or joined.
//   - Contacts   — nicknames the user has talked to, with note + last-seen.
//
// All three live in a single file at ~/.portal/portal.db; we open it
// with WAL mode so concurrent reads/writes from the Wails event loop
// don't block. The schema is created on Open if missing.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

// Store wraps the *sql.DB plus per-section helpers.
type Store struct {
	mu       sync.Mutex
	db       *sql.DB
	path     string
	closed   bool
}

// DefaultPath returns ~/.portal/portal.db.
func DefaultPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".portal", "portal.db"), nil
}

// Open opens or creates the SQLite database at path. Creates parent
// directories as needed. Schema is migrated to the latest version.
func Open(path string) (*Store, error) {
	if path == "" {
		var err error
		path, err = DefaultPath()
		if err != nil {
			return nil, err
		}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return nil, fmt.Errorf("storage: mkdir: %w", err)
	}

	dsn := fmt.Sprintf("file:%s?_journal=WAL&_busy_timeout=2000&_foreign_keys=on", path)
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, fmt.Errorf("storage: open: %w", err)
	}
	db.SetMaxOpenConns(1) // SQLite is happiest with serialised writes

	s := &Store{db: db, path: path}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

// Close shuts down the database. Idempotent.
func (s *Store) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return nil
	}
	s.closed = true
	return s.db.Close()
}

// Path returns the on-disk file path.
func (s *Store) Path() string { return s.path }

// migrate applies any pending schema changes. The version table lets
// us add migrations without breaking existing installs.
func (s *Store) migrate() error {
	const schema = `
	CREATE TABLE IF NOT EXISTS schema_version (
		version INTEGER NOT NULL PRIMARY KEY
	);

	CREATE TABLE IF NOT EXISTS settings (
		key   TEXT NOT NULL PRIMARY KEY,
		value TEXT NOT NULL
	);

	CREATE TABLE IF NOT EXISTS portal_history (
		id          INTEGER PRIMARY KEY AUTOINCREMENT,
		portal_id   TEXT NOT NULL,
		code        TEXT NOT NULL,
		nickname    TEXT NOT NULL,
		is_owner    INTEGER NOT NULL,
		joined_at   INTEGER NOT NULL,
		last_seen   INTEGER NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_portal_history_seen ON portal_history(last_seen DESC);

	CREATE TABLE IF NOT EXISTS contacts (
		nickname    TEXT NOT NULL PRIMARY KEY,
		peer_id     TEXT NOT NULL,
		note        TEXT NOT NULL DEFAULT '',
		last_seen   INTEGER NOT NULL
	);

	-- Services the user wants exposed automatically every time they
	-- enter a portal. Re-applied by app.bringUpMesh after the mesh
	-- comes online — saves users from having to re-Och every camera /
	-- game server / dev URL on every session.
	CREATE TABLE IF NOT EXISTS exposed_services (
		port      INTEGER NOT NULL,
		protocol  TEXT NOT NULL,
		name      TEXT NOT NULL,
		target    TEXT NOT NULL DEFAULT '',
		enabled   INTEGER NOT NULL DEFAULT 1,
		updated_at INTEGER NOT NULL,
		PRIMARY KEY (port, protocol)
	);
	`
	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("storage: migrate: %w", err)
	}
	// Stamp the current version (idempotent).
	_, _ = s.db.Exec(`INSERT OR REPLACE INTO schema_version(version) VALUES(1)`)
	return nil
}

// Now is broken out so tests can substitute a frozen clock.
var Now = func() time.Time { return time.Now() }

// scanError makes sql.ErrNoRows easier to detect for callers.
var ErrNotFound = errors.New("storage: not found")

func wrapNoRows(err error) error {
	if errors.Is(err, sql.ErrNoRows) {
		return ErrNotFound
	}
	return err
}
