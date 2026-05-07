package main

// Local-account gate. Portal keeps portal history, nickname, exposed-
// service rules and TURN credentials on disk under ~/.portal/portal.db
// — anyone with physical access to the laptop could otherwise list
// the user's portals or rejoin on their behalf. A username + password
// gate sits in front of the UI on every launch.
//
// The username is plaintext (not a secret — it doubles as the user's
// default Portal nickname). The password is bcrypt-hashed. Threat
// model is "someone else opens my laptop", not "an attacker has the
// disk image"; bcrypt-verified UI lock is the right friction. If we
// later need at-rest encryption that's a separate sqlcipher layer.

import (
	"errors"
	"strings"

	"golang.org/x/crypto/bcrypt"

	"portal_traffic_client/storage"
)

const (
	minUsernameLen = 1  // generous; the nickname-style field on Welcome accepts the same.
	maxUsernameLen = 24 // matches Welcome's input maxLength so the username can be reused.
	minPasswordLen = 4  // PIN-equivalent floor; more friction pushes users to skip auth.
)

// HasAccount reports whether a username + password pair is configured.
// Frontend uses this on startup to pick between Sign-Up (first launch)
// and Sign-In (every subsequent launch).
func (a *App) HasAccount() bool {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return false
	}
	u, _ := store.GetSetting(storage.KeyAuthUsername)
	h, _ := store.GetSetting(storage.KeyAuthHash)
	return u != "" && h != ""
}

// CurrentUsername returns the stored username, or "" if no account.
// The frontend reads this after sign-in to pre-fill the nickname
// field on Welcome — saves the user from typing it again.
func (a *App) CurrentUsername() string {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return ""
	}
	u, _ := store.GetSetting(storage.KeyAuthUsername)
	return u
}

// SignUp creates the local account. Validates lengths, hashes the
// password with bcrypt at the default cost, and stores both rows.
// Errors (returned as plain strings the frontend matches on):
//
//	username_empty       — username trimmed to nothing
//	username_too_long    — > maxUsernameLen runes
//	password_too_short   — < minPasswordLen
//	storage_unavailable  — *Store wasn't ready yet
//
// Used both for first-time setup and (later, from Settings) for
// "change account" — overwrites any existing rows on success.
func (a *App) SignUp(username, password string) error {
	uname := strings.TrimSpace(username)
	if uname == "" {
		return errors.New("username_empty")
	}
	if len([]rune(uname)) > maxUsernameLen {
		return errors.New("username_too_long")
	}
	if len(strings.TrimSpace(password)) < minPasswordLen {
		return errors.New("password_too_short")
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage_unavailable")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := store.PutSetting(storage.KeyAuthUsername, uname); err != nil {
		return err
	}
	if err := store.PutSetting(storage.KeyAuthHash, string(h)); err != nil {
		return err
	}
	// Persist the username as the nickname the Welcome screen will
	// pre-fill. Without this, after sign-up the user has to retype
	// their handle to create a portal — duplicative for no reason.
	_ = store.PutSetting(storage.KeyNickname, uname)
	return nil
}

// SignIn returns true iff (username, password) match the stored
// credentials. Username comparison is plain ==; the password goes
// through bcrypt.CompareHashAndPassword (constant time).
//
// Returns false on any storage error or missing rows so callers
// don't have to distinguish "wrong creds" from "broken DB" at the
// UI level — both surface as a single "wrong username/password"
// banner.
func (a *App) SignIn(username, password string) bool {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return false
	}
	storedUser, err := store.GetSetting(storage.KeyAuthUsername)
	if err != nil || storedUser == "" {
		return false
	}
	if strings.TrimSpace(username) != storedUser {
		return false
	}
	h, err := store.GetSetting(storage.KeyAuthHash)
	if err != nil || h == "" {
		return false
	}
	return bcrypt.CompareHashAndPassword([]byte(h), []byte(password)) == nil
}

// ResetVault is the "forgot password" escape hatch. Wipes username,
// password hash, AND portal history (so a forgotten password can't
// be used to learn which portals the previous account was in).
// Network-level settings (signaling URL, TURN config) are kept —
// they're not personal.
func (a *App) ResetVault() error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	_ = store.DeleteSetting(storage.KeyAuthUsername)
	_ = store.DeleteSetting(storage.KeyAuthHash)
	return store.ClearHistory()
}
