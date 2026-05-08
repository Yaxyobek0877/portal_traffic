package main

// Local-account gate. Portal keeps portal history, nickname, exposed-
// service rules and TURN credentials on disk under ~/.portal/portal.db
// — anyone with physical access to the laptop could otherwise list
// the user's portals or rejoin on their behalf. A username + password
// gate sits in front of the UI on every launch.
//
// Threat model is "someone else opens my laptop". For that scenario
// the right defenses are:
//
//   1. bcrypt at the default cost (12) so a stolen DB takes hours
//      to brute-force per password.
//   2. A strong-password policy (8 chars + lower + upper + digit +
//      special) so even a stolen-DB attacker has to actually search.
//   3. An in-memory sign-in rate limiter (5 attempts per username
//      then a 30-second lockout) so the UI can't be sat on and
//      hammered while the user steps away.
//
// What this does NOT defend against: an attacker who has the disk
// image and unlimited time. For that we'd need at-rest encryption
// (sqlcipher) keyed off the password — a separate layer.

import (
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"portal_traffic_client/storage"
)

const (
	minUsernameLen = 1
	maxUsernameLen = 24

	// Min password length. Bumped from 4 → 8 with the strong-password
	// rollout; existing accounts created at the old floor still sign
	// in (SignIn doesn't re-validate the strength of stored secrets).
	minPasswordLen = 8

	// bcrypt cost. DefaultCost (10) is fine for a desktop app; we
	// declare the constant explicitly so changing it is one edit
	// rather than a hunt through the file.
	bcryptCost = bcrypt.DefaultCost

	// Sign-in lockout after this many failures per username.
	maxSignInAttempts  = 5
	signInLockoutDelay = 30 * time.Second
)

// SignInResult is what the bridge ships to the frontend. lockoutSeconds
// is non-zero when the account is currently rate-limited; the UI shows
// a countdown instead of a generic "wrong credentials" banner.
type SignInResult struct {
	OK             bool `json:"ok"`
	LockoutSeconds int  `json:"lockoutSeconds"`
}

// signInLimiter keeps recent failure counts in memory. Restart of the
// process resets it — fine for the local threat model (the attacker
// would have to restart the GUI between every guess, and bcrypt makes
// that brutally slow regardless).
type signInLimiter struct {
	mu    sync.Mutex
	state map[string]*signInState
}

type signInState struct {
	failures    int
	lockedUntil time.Time
}

// Package-level singleton — App is the singleton already, but its
// methods are bound by Wails reflection and we don't want to add a
// field to it just for this. A package var is simpler and equivalent.
var signInLimit = &signInLimiter{state: map[string]*signInState{}}

// HasAccount reports whether a username + password pair is configured.
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

// CurrentUsername returns the stored username for nickname pre-fill.
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

// validatePasswordStrength enforces the strong-password policy on
// new passwords. Returns nil if the password passes; otherwise an
// error whose Error() string the frontend matches on. Errors are
// kept stable so localised UI messages can map cleanly.
//
// Rules: at least 8 characters, AND at least one of each:
//
//   - ASCII lowercase letter (a-z)
//   - ASCII uppercase letter (A-Z)
//   - ASCII digit (0-9)
//   - punctuation/symbol (anything that's not letter/digit/space)
//
// We don't ban specific weak passwords (no "password123" blocklist) —
// that ends up as theatre against a bcrypted DB. The four-category
// rule plus length is the meaningful win.
func validatePasswordStrength(pwd string) error {
	if len(pwd) < minPasswordLen {
		return fmt.Errorf("password_too_short")
	}
	var hasLower, hasUpper, hasDigit, hasSpecial bool
	for _, r := range pwd {
		switch {
		case unicode.IsLower(r):
			hasLower = true
		case unicode.IsUpper(r):
			hasUpper = true
		case unicode.IsDigit(r):
			hasDigit = true
		case !unicode.IsSpace(r) && !unicode.IsLetter(r) && !unicode.IsDigit(r):
			hasSpecial = true
		}
	}
	missing := []string{}
	if !hasLower {
		missing = append(missing, "lower")
	}
	if !hasUpper {
		missing = append(missing, "upper")
	}
	if !hasDigit {
		missing = append(missing, "digit")
	}
	if !hasSpecial {
		missing = append(missing, "special")
	}
	if len(missing) > 0 {
		return fmt.Errorf("password_weak:%s", strings.Join(missing, ","))
	}
	return nil
}

// SignUp creates the local account. Validates the username (1-24
// chars, non-empty) and the password (strength policy via
// validatePasswordStrength), bcrypt-hashes, and stores both.
//
// Errors (the frontend matches on these prefixes):
//
//	username_empty       — username trimmed to nothing
//	username_too_long    — > maxUsernameLen runes
//	password_too_short   — < minPasswordLen
//	password_weak:lower,upper,digit,special  — missing categories
//	storage_unavailable  — *Store wasn't ready yet
func (a *App) SignUp(username, password string) error {
	uname := strings.TrimSpace(username)
	if uname == "" {
		return errors.New("username_empty")
	}
	if len([]rune(uname)) > maxUsernameLen {
		return errors.New("username_too_long")
	}
	if err := validatePasswordStrength(password); err != nil {
		return err
	}
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage_unavailable")
	}
	h, err := bcrypt.GenerateFromPassword([]byte(password), bcryptCost)
	if err != nil {
		return err
	}
	if err := store.PutSetting(storage.KeyAuthUsername, uname); err != nil {
		return err
	}
	if err := store.PutSetting(storage.KeyAuthHash, string(h)); err != nil {
		return err
	}
	// Username doubles as the default Portal nickname so Welcome
	// doesn't have to ask again.
	_ = store.PutSetting(storage.KeyNickname, uname)
	// New password → reset any in-memory lockout on this username.
	signInLimit.mu.Lock()
	delete(signInLimit.state, uname)
	signInLimit.mu.Unlock()
	return nil
}

// SignIn verifies (username, password) against the stored credentials
// and applies an in-memory rate limit. The return type is a struct
// rather than a bool so the UI can show a lockout countdown without
// guessing.
//
// Behaviour:
//   - Lockout active   → returns {OK: false, LockoutSeconds: N}.
//   - Wrong creds      → counts a failure; on the 5th the lockout
//                        starts and the response carries the seconds.
//   - Correct creds    → resets the counter for this username and
//                        returns {OK: true, LockoutSeconds: 0}.
//
// Storage errors and missing rows surface as "wrong creds" rather
// than as a distinct response, so the UI can't be used as an
// account-existence oracle.
func (a *App) SignIn(username, password string) SignInResult {
	uname := strings.TrimSpace(username)

	signInLimit.mu.Lock()
	st, ok := signInLimit.state[uname]
	if ok && time.Now().Before(st.lockedUntil) {
		remaining := int(time.Until(st.lockedUntil).Round(time.Second).Seconds())
		if remaining < 1 {
			remaining = 1
		}
		signInLimit.mu.Unlock()
		return SignInResult{OK: false, LockoutSeconds: remaining}
	}
	signInLimit.mu.Unlock()

	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return SignInResult{OK: false}
	}

	storedUser, err := store.GetSetting(storage.KeyAuthUsername)
	if err != nil || storedUser == "" {
		return signInRecordFailure(uname)
	}
	if uname != storedUser {
		return signInRecordFailure(uname)
	}
	h, err := store.GetSetting(storage.KeyAuthHash)
	if err != nil || h == "" {
		return signInRecordFailure(uname)
	}
	if bcrypt.CompareHashAndPassword([]byte(h), []byte(password)) != nil {
		return signInRecordFailure(uname)
	}

	// Success — wipe the failure counter for this username.
	signInLimit.mu.Lock()
	delete(signInLimit.state, uname)
	signInLimit.mu.Unlock()
	return SignInResult{OK: true}
}

// signInRecordFailure increments the failure counter for uname and,
// on the threshold, sets a lockout window. Returns the SignInResult
// the caller should ship back to the bridge.
func signInRecordFailure(uname string) SignInResult {
	signInLimit.mu.Lock()
	defer signInLimit.mu.Unlock()
	st, ok := signInLimit.state[uname]
	if !ok {
		st = &signInState{}
		signInLimit.state[uname] = st
	}
	st.failures++
	if st.failures >= maxSignInAttempts {
		st.lockedUntil = time.Now().Add(signInLockoutDelay)
		st.failures = 0 // start the count over after the lockout
		remaining := int(signInLockoutDelay.Round(time.Second).Seconds())
		return SignInResult{OK: false, LockoutSeconds: remaining}
	}
	return SignInResult{OK: false}
}

// ResetVault is the "forgot password" escape hatch. Wipes username,
// password hash, AND portal history. Network-level settings (signaling
// URL, TURN config) are kept. Also clears the in-memory rate limiter
// for the abandoned username so the next account doesn't inherit it.
func (a *App) ResetVault() error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return nil
	}
	priorUser, _ := store.GetSetting(storage.KeyAuthUsername)
	_ = store.DeleteSetting(storage.KeyAuthUsername)
	_ = store.DeleteSetting(storage.KeyAuthHash)
	if priorUser != "" {
		signInLimit.mu.Lock()
		delete(signInLimit.state, priorUser)
		signInLimit.mu.Unlock()
	}
	// Auto-reconnect rows belong to the prior identity — wipe them so
	// the next account's first launch doesn't silently re-dial portals
	// the previous user had open.
	_ = store.ClearActiveSessions()
	return store.ClearHistory()
}
