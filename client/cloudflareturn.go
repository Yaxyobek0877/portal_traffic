// Cloudflare Calls TURN integration.
//
// Cloudflare offers a globally-distributed TURN service ($0.05/GB
// outbound, free for the first 1 TB/month). Unlike most TURN
// providers, credentials are short-lived: the user's browser/app
// authenticates with Cloudflare's API using a long-lived TOKEN_ID +
// API_TOKEN pair, and Cloudflare returns ICE-server credentials
// valid for a configurable window (default 1 hour).
//
// We persist the long-lived pair in SQLite. On every CreatePortal /
// JoinPortal we fetch fresh creds and inject them into the WebRTC
// configuration before constructing the mesh.Manager.
//
// Docs: https://developers.cloudflare.com/calls/turn/
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/pion/webrtc/v4"
	"portal_traffic_client/mesh"
)

// mesh_DefaultICEServers exposes mesh's default servers for our
// resolveICEServers helper without circular imports.
var mesh_DefaultICEServers = mesh.DefaultICEServers

// CloudflareTurnConfig is the long-lived API credentials.
type CloudflareTurnConfig struct {
	TokenID  string `json:"tokenId"`
	APIToken string `json:"apiToken"`
}

// cloudflareTurnCache holds the most recently fetched short-lived
// credentials so we don't burn a Cloudflare API call on every
// CreatePortal/JoinPortal. Cached for 30 minutes — well under the
// 1-hour default TTL.
type cloudflareTurnCache struct {
	mu      sync.Mutex
	servers []webrtc.ICEServer
	until   time.Time
}

// cfTurnGenResponse mirrors the Cloudflare API response shape.
type cfTurnGenResponse struct {
	IceServers struct {
		URLs       interface{} `json:"urls"`
		Username   string      `json:"username"`
		Credential string      `json:"credential"`
	} `json:"iceServers"`
}

// fetchCloudflareTurn calls Cloudflare's credentials/generate
// endpoint with the configured token pair and returns ICE servers
// the WebRTC engine can use. ttlSeconds=3600 (1h) is the default.
//
// Errors are returned for: missing config, API failure, malformed
// response. The caller should fall back to default ICE servers.
func fetchCloudflareTurn(ctx context.Context, cfg CloudflareTurnConfig, ttlSeconds int) ([]webrtc.ICEServer, error) {
	if cfg.TokenID == "" || cfg.APIToken == "" {
		return nil, errors.New("cloudflare turn: token_id va api_token to'liq emas")
	}
	if ttlSeconds <= 0 {
		ttlSeconds = 3600
	}

	body, _ := json.Marshal(map[string]int{"ttl": ttlSeconds})
	url := fmt.Sprintf(
		"https://rtc.live.cloudflare.com/v1/turn/keys/%s/credentials/generate",
		cfg.TokenID,
	)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+cfg.APIToken)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cloudflare turn request: %w", err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("cloudflare turn HTTP %d: %s", resp.StatusCode, snippet(raw))
	}
	var parsed cfTurnGenResponse
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("cloudflare turn parse: %w", err)
	}

	urls := normaliseURLs(parsed.IceServers.URLs)
	if len(urls) == 0 {
		return nil, errors.New("cloudflare turn: no urls in response")
	}
	return []webrtc.ICEServer{
		{
			URLs:       urls,
			Username:   parsed.IceServers.Username,
			Credential: parsed.IceServers.Credential,
		},
	}, nil
}

// normaliseURLs accepts the Cloudflare response shape: either a
// single string or a string slice. Returns []string.
func normaliseURLs(v interface{}) []string {
	switch t := v.(type) {
	case string:
		return []string{t}
	case []interface{}:
		out := make([]string, 0, len(t))
		for _, x := range t {
			if s, ok := x.(string); ok {
				out = append(out, s)
			}
		}
		return out
	}
	return nil
}

func snippet(b []byte) string {
	if len(b) > 200 {
		b = b[:200]
	}
	return string(b)
}

// ----------------------------------------------------------------------------
// App methods
// ----------------------------------------------------------------------------

// GetCloudflareTurn returns the persisted Cloudflare TURN config
// (token id + API token). Empty values mean unconfigured.
func (a *App) GetCloudflareTurn() CloudflareTurnConfig {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return CloudflareTurnConfig{}
	}
	return CloudflareTurnConfig{
		TokenID:  store.GetOr("cf_turn_token_id", ""),
		APIToken: store.GetOr("cf_turn_api_token", ""),
	}
}

// SetCloudflareTurn persists the Cloudflare TURN credentials.
// Either field empty disables Cloudflare TURN; both must be set
// for it to be used.
func (a *App) SetCloudflareTurn(cfg CloudflareTurnConfig) error {
	a.mu.RLock()
	store := a.store
	a.mu.RUnlock()
	if store == nil {
		return errors.New("storage mavjud emas")
	}
	_ = store.PutSetting("cf_turn_token_id", cfg.TokenID)
	_ = store.PutSetting("cf_turn_api_token", cfg.APIToken)
	// Invalidate any cached credentials so the next call refetches.
	a.cfTurnCache.mu.Lock()
	a.cfTurnCache.until = time.Time{}
	a.cfTurnCache.servers = nil
	a.cfTurnCache.mu.Unlock()
	return nil
}

// TestCloudflareTurn calls the API and reports whether the creds work,
// what URLs / username came back, and how long the call took. Lets
// the user verify their setup without creating a portal.
func (a *App) TestCloudflareTurn() TurnTestResult {
	cfg := a.GetCloudflareTurn()
	res := TurnTestResult{}
	if cfg.TokenID == "" || cfg.APIToken == "" {
		res.Message = "Cloudflare TURN sozlanmagan (Settings → Cloudflare TURN)"
		return res
	}

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	servers, err := fetchCloudflareTurn(ctx, cfg, 600)
	res.GatherMs = time.Since(start).Milliseconds()
	if err != nil {
		res.Message = err.Error()
		a.logger.Warn("cloudflare turn test failed", "err", err)
		return res
	}
	for _, s := range servers {
		res.URLs = append(res.URLs, s.URLs...)
	}
	res.OK = true
	res.HadRelay = true
	res.Types = []string{"relay"}
	res.Message = "Cloudflare TURN ishlamoqda — short-lived credentials qaytarildi ✓"
	a.logger.Info("cloudflare turn test ok", "urls", res.URLs)
	return res
}

// resolveICEServers builds the final ICE-server list for a session.
// Precedence:
//  1. Cloudflare TURN (if configured + credentials fetch succeeds)
//  2. Manual TURN URL (if configured)
//  3. Default STUN-only fallback
//
// Cached Cloudflare creds are reused for 30 minutes to minimise API
// calls.
func (a *App) resolveICEServers() []webrtc.ICEServer {
	cf := a.GetCloudflareTurn()
	if cf.TokenID != "" && cf.APIToken != "" {
		// Try cache first.
		a.cfTurnCache.mu.Lock()
		valid := time.Now().Before(a.cfTurnCache.until) && len(a.cfTurnCache.servers) > 0
		cached := a.cfTurnCache.servers
		a.cfTurnCache.mu.Unlock()
		if valid {
			a.logger.Debug("cloudflare turn cache hit")
			return append(append([]webrtc.ICEServer{}, mesh_DefaultICEServers...), cached...)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
		defer cancel()
		servers, err := fetchCloudflareTurn(ctx, cf, 3600)
		if err == nil && len(servers) > 0 {
			a.cfTurnCache.mu.Lock()
			a.cfTurnCache.servers = servers
			a.cfTurnCache.until = time.Now().Add(30 * time.Minute)
			a.cfTurnCache.mu.Unlock()
			a.logger.Info("cloudflare turn fetched", "url_count",
				countServerURLs(servers))
			return append(append([]webrtc.ICEServer{}, mesh_DefaultICEServers...), servers...)
		}
		a.logger.Warn("cloudflare turn fetch failed; falling back", "err", err)
	}
	// No Cloudflare TURN — let the mesh layer apply manual TurnURL
	// from settings (handled inside mesh.New) on top of the STUN
	// defaults. Returning nil signals "use mesh defaults + manual".
	return nil
}

func countServerURLs(servers []webrtc.ICEServer) int {
	n := 0
	for _, s := range servers {
		n += len(s.URLs)
	}
	return n
}
