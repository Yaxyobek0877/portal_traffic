package storage

// Setting access — a tiny key/value store backed by the settings table.
// We don't bother with typed columns because the set of keys is small
// and slow-changing; everything serialises as a string and the caller
// parses on the read side.

func (s *Store) GetSetting(key string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var v string
	err := s.db.QueryRow(`SELECT value FROM settings WHERE key=?`, key).Scan(&v)
	if err != nil {
		return "", wrapNoRows(err)
	}
	return v, nil
}

// GetOr returns the stored value or the default if no row exists.
func (s *Store) GetOr(key, def string) string {
	v, err := s.GetSetting(key)
	if err != nil {
		return def
	}
	return v
}

// PutSetting upserts a key/value.
func (s *Store) PutSetting(key, value string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`
		INSERT INTO settings(key, value) VALUES(?, ?)
		ON CONFLICT(key) DO UPDATE SET value=excluded.value
	`, key, value)
	return err
}

// DeleteSetting removes a key. No-op if missing.
func (s *Store) DeleteSetting(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	_, err := s.db.Exec(`DELETE FROM settings WHERE key=?`, key)
	return err
}

// AllSettings returns every key/value pair. Used for the settings page
// "export" feature.
func (s *Store) AllSettings() (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rows, err := s.db.Query(`SELECT key, value FROM settings`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]string)
	for rows.Next() {
		var k, v string
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		out[k] = v
	}
	return out, rows.Err()
}

// Canonical setting keys.
const (
	KeySignalingURL   = "signaling_url"
	KeyNickname       = "nickname"
	KeyPublicNickname = "public_nickname"
	KeyTheme          = "theme"
	KeyAnimationLevel = "animation_level"

	KeyTurnURL        = "turn_url"
	KeyTurnUsername   = "turn_username"
	KeyTurnCredential = "turn_credential"

	// Log upload — opt-out diagnostic shipping. See client/logsink/.
	// Empty = default-on, "0" disables, anything else (e.g. "1") forces on.
	KeyLogUpload    = "log_upload_enabled"
	KeyLogUploadURL = "log_upload_url"

	// Local account — username/password gate the app shows on launch.
	// auth_username is plaintext (it's not a secret; it doubles as the
	// default Portal nickname after sign-in). auth_hash is bcrypt-hashed.
	// Both empty → no account configured → SignUp screen is shown.
	// See client/auth.go.
	KeyAuthUsername = "auth_username"
	KeyAuthHash     = "auth_hash"

	// Device name — short label the user attaches to THIS install
	// ('uy' / 'ish' / 'mac' / 'win 64'), so when the same account
	// signs in from a phone + a laptop + a work desktop, room peers
	// can tell them apart. Empty defaults to a platform-derived name
	// at first read (App.CurrentDeviceName). Combined with the
	// nickname when announcing to the mesh: 'texuz · uy'.
	KeyDeviceName = "device_name"

	// Auto-run — when "1" the app installs an OS-level startup hook
	// (LaunchAgent / Run registry / .desktop autostart) so it comes
	// back after reboot. App.SetAutoRun(true) writes the hook;
	// SetAutoRun(false) removes it. Empty / "0" = no auto-run.
	KeyAutoRun = "auto_run"
)
