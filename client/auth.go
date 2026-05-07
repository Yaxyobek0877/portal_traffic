package main

// Local vault unlock. Portal is a P2P tool that keeps history,
// nicknames, exposed-service rules and TURN credentials on the local
// disk under ~/.portal/portal.db — anyone with physical access to the
// laptop can otherwise list the user's portals and connect on their
// behalf. A single master password gates the UI on launch.
//
// We deliberately do NOT use the password to derive an encryption key
// for the database. Threat model is "someone else opens my laptop",
// not "an attacker has the disk image" — bcrypt-verified UI lock is
// the right level of friction for that. If we later need at-rest
// encryption, that's a separate KDF + sqlcipher layer.

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"portal_traffic_client/storage"
)

// Minimum bytes the user must type before SetPassword accepts. Four
// is what every banking PIN uses; we don't enforce more because this
// is a local-only secret and false friction will push users to skip.
const minPasswordLen = 4

// HasPassword reports whether a master password is configured. The
// frontend checks this on startup to pick between "create password"
// (first launch) and "enter password" (every subsequent launch).
func (a *App) HasPassword() bool {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return false
	}
	h, _ := store.GetSetting(storage.KeyAuthHash)
	return h != ""
}

// SetPassword installs a new master password, hashing with bcrypt at
// the default cost (currently 10). Returns an error if the password
// is shorter than minPasswordLen, all whitespace, or storage fails.
//
// Used both for first-time setup and for "change password" later from
// Settings — overwrites any existing hash on success.
func (a *App) SetPassword(pwd string) error {
	if len(strings.TrimSpace(pwd)) < minPasswordLen {
		return errors.New("password_too_short")
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage_unavailable")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(pwd), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	return store.PutSetting(storage.KeyAuthHash, string(h))
}

// VerifyPassword returns true iff pwd matches the stored hash. If no
// hash is configured it returns false — callers should use HasPassword
// to detect that case before prompting.
//
// bcrypt.CompareHashAndPassword is the constant-time path; we don't
// short-circuit on length to avoid leaking the hash format.
func (a *App) VerifyPassword(pwd string) bool {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return false
	}
	h, err := store.GetSetting(storage.KeyAuthHash)
	if err != nil || h == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(pwd)) == nil
}

// ResetVault is the "I forgot my password" escape hatch. Wipes the
// auth hash AND the portal history (so a forgotten password can't be
// used to learn which portals the previous owner was in). Settings
// like signaling URL and TURN config are kept — they're not personal.
//
// This is destructive on purpose; the frontend confirms before calling.
func (a *App) ResetVault() error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	_ = store.DeleteSetting(storage.KeyAuthHash)
	return store.ClearHistory()
}
