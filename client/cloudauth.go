// Cloud-account authentication bridge for the desktop app.
//
// The signaling server (signaling.1pro.uz) hosts a server-side
// account store at /api/auth/{signup,signin,signout,me}. This file
// exposes those endpoints to the Wails frontend so the user can
// link the desktop install to a cloud account.
//
// What linking buys you (and what's still local):
//   - LINKED: when the desktop creates a portal, the signaling
//     server tags it with your user ID. The web dashboard
//     (portal.1pro.uz/admin/dashboard) then lists every portal
//     owned by you across every signed-in device, so opening the
//     site from your phone shows the rooms running on your
//     laptop. Same goes for the reverse — a phone can list a
//     portal created on the desktop and join it as a separate
//     device.
//   - LOCAL ONLY: the password vault, exposed services, transfer
//     history, autorun toggle, and all per-device settings stay
//     on this machine. Cloud auth is purely a directory layer —
//     it never sees your local portal codes or chat traffic.
//
// Persistence: we save the token + username under
// .portal/cloud-auth.json (mode 0600) so the link survives a
// restart. Sign-out wipes the file. The token is opaque to us;
// only the server can decode it.

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// CloudUser is the JSON-side representation of cloudAuthState.
// Wails marshals struct field names lowercase-first; mirroring that
// shape lets the frontend treat this as a plain object.
type CloudUser struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	HasToken bool   `json:"hasToken"` // never expose the raw token to the renderer
}

// CloudErr is the canonical error code returned from server endpoints.
// We let the frontend localize them, the same way the existing local
// SignIn flow does.
type CloudErr struct {
	Code           string `json:"error"`
	LockoutSeconds int    `json:"lockoutSeconds,omitempty"`
}

func (e *CloudErr) Error() string { return e.Code }

// cloudAuthFile is the on-disk path. Sits under ~/.portal/ next to
// the existing portal.db so backup tooling that already targets the
// app dir picks it up too.
func (a *App) cloudAuthFile() string {
	dir, err := os.UserHomeDir()
	if err != nil || dir == "" {
		dir = os.TempDir()
	}
	return filepath.Join(dir, ".portal", "cloud-auth.json")
}

// loadCloudAuth restores the saved token, if any. Missing file is OK.
// Called from Startup; failures are logged but don't block boot.
func (a *App) loadCloudAuth() {
	path := a.cloudAuthFile()
	data, err := os.ReadFile(path)
	if err != nil {
		return
	}
	var st cloudAuthState
	if err := json.Unmarshal(data, &st); err != nil {
		a.logger.Warn("cloud auth: parse failed", "err", err)
		return
	}
	a.cloudMu.Lock()
	a.cloudAuth = st
	a.cloudMu.Unlock()
	a.logger.Info("cloud auth: restored", "username", st.Username)
}

func (a *App) saveCloudAuth(st cloudAuthState) {
	path := a.cloudAuthFile()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		a.logger.Warn("cloud auth: mkdir failed", "err", err)
		return
	}
	b, _ := json.MarshalIndent(st, "", "  ")
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, b, 0o600); err != nil {
		a.logger.Warn("cloud auth: write failed", "err", err)
		return
	}
	if err := os.Rename(tmp, path); err != nil {
		a.logger.Warn("cloud auth: rename failed", "err", err)
	}
}

func (a *App) clearCloudAuth() {
	_ = os.Remove(a.cloudAuthFile())
}

// CloudAuthToken returns the active session token, or "" if not
// signed in. Called by the mesh wiring path so a fresh portal
// session sends ?token=... on its WebSocket connect.
func (a *App) CloudAuthToken() string {
	a.cloudMu.RLock()
	defer a.cloudMu.RUnlock()
	return a.cloudAuth.Token
}

// cloudBaseURL derives the API base from the signaling URL — we
// intentionally point at the same origin so the same TLS cert and
// the same Cloudflare tunnel terminate both. Falls back to the
// production host when the signaling URL is unrecognised.
func (a *App) cloudBaseURL() string {
	a.mu.RLock()
	sigURL := a.url
	a.mu.RUnlock()
	if sigURL == "" {
		return "https://portal.1pro.uz"
	}
	u, err := url.Parse(sigURL)
	if err != nil || u.Host == "" {
		return "https://portal.1pro.uz"
	}
	scheme := "https"
	if u.Scheme == "ws" || u.Scheme == "http" {
		scheme = "http"
	}
	host := u.Host
	// Map signaling.<domain> → portal.<domain> when we can; the auth
	// API lives on the marketing host. Falls back to using the
	// signaling host itself, which is also fine because the new
	// signaling binary serves /api/* on the same listener.
	if strings.HasPrefix(host, "signaling.") {
		host = "portal." + strings.TrimPrefix(host, "signaling.")
	}
	return scheme + "://" + host
}

// cloudPost issues a JSON POST and parses the response. Returns the
// raw response bytes plus the structured error (if any). The
// frontend uses err.Code to render the appropriate message.
func (a *App) cloudPost(path string, payload any) ([]byte, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	url := a.cloudBaseURL() + path
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https") // for the __Host- cookie path
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if resp.StatusCode/100 != 2 {
		var e CloudErr
		_ = json.Unmarshal(raw, &e)
		if e.Code == "" {
			e.Code = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		return raw, &e
	}
	return raw, nil
}

// CloudSignUp creates a new server account. On success the token is
// stored both in memory and on disk; subsequent CloudCurrent /
// CloudAuthToken reads return it. The frontend should treat any
// returned error as a JSON-shaped {error, lockoutSeconds}.
func (a *App) CloudSignUp(username, password string) (*CloudUser, error) {
	if username == "" || password == "" {
		return nil, errors.New("invalid_credentials")
	}
	type body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	raw, err := a.cloudPost("/api/auth/signup", body{Username: username, Password: password})
	if err != nil {
		return nil, err
	}
	var resp struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	if uerr := json.Unmarshal(raw, &resp); uerr != nil {
		return nil, uerr
	}
	st := cloudAuthState{Token: resp.Token, UserID: resp.ID, Username: resp.Username}
	a.cloudMu.Lock()
	a.cloudAuth = st
	a.cloudMu.Unlock()
	a.saveCloudAuth(st)
	a.logger.Info("cloud auth: signed up", "username", st.Username)
	return &CloudUser{UserID: st.UserID, Username: st.Username, HasToken: st.Token != ""}, nil
}

// CloudSignIn authenticates against an existing server account.
// Same shape as CloudSignUp on the wire.
func (a *App) CloudSignIn(username, password string) (*CloudUser, error) {
	if username == "" || password == "" {
		return nil, errors.New("invalid_credentials")
	}
	type body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	raw, err := a.cloudPost("/api/auth/signin", body{Username: username, Password: password})
	if err != nil {
		return nil, err
	}
	var resp struct {
		ID       string `json:"id"`
		Username string `json:"username"`
		Token    string `json:"token"`
	}
	if uerr := json.Unmarshal(raw, &resp); uerr != nil {
		return nil, uerr
	}
	st := cloudAuthState{Token: resp.Token, UserID: resp.ID, Username: resp.Username}
	a.cloudMu.Lock()
	a.cloudAuth = st
	a.cloudMu.Unlock()
	a.saveCloudAuth(st)
	a.logger.Info("cloud auth: signed in", "username", st.Username)
	return &CloudUser{UserID: st.UserID, Username: st.Username, HasToken: st.Token != ""}, nil
}

// CloudSignOut clears the token locally. We don't bother hitting
// /api/auth/signout — the cookie/token is an opaque local secret,
// dropping the file is enough; the server-side session expires on
// its own.
func (a *App) CloudSignOut() {
	a.cloudMu.Lock()
	a.cloudAuth = cloudAuthState{}
	a.cloudMu.Unlock()
	a.clearCloudAuth()
	a.logger.Info("cloud auth: signed out")
}

// CloudCurrent returns the cached user for UI binding. An empty
// struct (UserID == "") means no active link. Doesn't network.
func (a *App) CloudCurrent() CloudUser {
	a.cloudMu.RLock()
	st := a.cloudAuth
	a.cloudMu.RUnlock()
	return CloudUser{UserID: st.UserID, Username: st.Username, HasToken: st.Token != ""}
}
