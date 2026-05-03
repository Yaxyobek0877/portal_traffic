# Server-issued TURN credentials

Default TURN — bu signaling serveri har clientga **qisqa muddatli**
TURN credentials yuborib bersin, foydalanuvchi qo'lda Cloudflare
hisob ochib token kiritmasin.

Mijoz tomon (client/) **tayyor** — `protocol.PortalCreated`/`PortalJoined`
ichidagi `ice_servers` maydoni mavjud bo'lsa, mesh.Manager uni o'qib
har peer connection'ga qo'shadi (`mesh.Manager.applyServerICE`,
`mesh.Manager.iceServersForPeer`).

Qoldiq: signaling serverida (xususiy repo) shu maydonni to'ldirish.

## Cloudflare TURN credentials API

Long-lived API token bilan har bir sessiya uchun yangi credentials
hosil qilinadi:

```
POST https://rtc.live.cloudflare.com/v1/turn/keys/{TOKEN_ID}/credentials/generate
Authorization: Bearer {API_TOKEN}
Content-Type: application/json

{ "ttl": 3600 }
```

Javob:

```json
{
  "iceServers": {
    "urls": [
      "stun:stun.cloudflare.com:3478",
      "turn:turn.cloudflare.com:3478?transport=udp",
      "turn:turn.cloudflare.com:3478?transport=tcp",
      "turns:turn.cloudflare.com:5349?transport=tcp"
    ],
    "username": "<short-lived>",
    "credential": "<short-lived>"
  }
}
```

## Go integratsiyasi (signaling server)

Quyidagini xususiy `server/` repo'ga qo'shing. `CF_TURN_TOKEN_ID` va
`CF_TURN_API_TOKEN` env-var sifatida boshlanish vaqtida o'qiladi —
hech qachon kodga yozmang.

```go
// server/cfturn.go
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sync"
	"time"

	"portal_traffic/shared/protocol"
)

type cfTurnClient struct {
	tokenID  string
	apiToken string
	http     *http.Client

	mu       sync.Mutex
	cached   []protocol.ICEServer
	expiry   time.Time
}

func newCFTurn() *cfTurnClient {
	return &cfTurnClient{
		tokenID:  os.Getenv("CF_TURN_TOKEN_ID"),
		apiToken: os.Getenv("CF_TURN_API_TOKEN"),
		http:     &http.Client{Timeout: 5 * time.Second},
	}
}

// ICEServers returns a fresh credential bundle. Cached for half the
// TTL so we don't hammer Cloudflare; clients reconnecting within the
// window get the same bundle, which is fine — TURN doesn't care.
func (c *cfTurnClient) ICEServers(ctx context.Context) []protocol.ICEServer {
	if c == nil || c.tokenID == "" || c.apiToken == "" {
		return nil
	}
	c.mu.Lock()
	if time.Now().Before(c.expiry) && len(c.cached) > 0 {
		out := c.cached
		c.mu.Unlock()
		return out
	}
	c.mu.Unlock()

	body, _ := json.Marshal(map[string]any{"ttl": 3600})
	url := fmt.Sprintf(
		"https://rtc.live.cloudflare.com/v1/turn/keys/%s/credentials/generate",
		c.tokenID,
	)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil
	}
	req.Header.Set("Authorization", "Bearer "+c.apiToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil
	}

	var parsed struct {
		IceServers struct {
			URLs       []string `json:"urls"`
			Username   string   `json:"username"`
			Credential string   `json:"credential"`
		} `json:"iceServers"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		return nil
	}
	if len(parsed.IceServers.URLs) == 0 {
		return nil
	}

	out := []protocol.ICEServer{{
		URLs:       parsed.IceServers.URLs,
		Username:   parsed.IceServers.Username,
		Credential: parsed.IceServers.Credential,
	}}

	// Cache for half the TTL.
	c.mu.Lock()
	c.cached = out
	c.expiry = time.Now().Add(30 * time.Minute)
	c.mu.Unlock()

	return out
}
```

## Hub'ga ulash

`server/hub.go` (yoki shu portal.create / portal.join handler'i bor
joyda):

```go
// In Hub:
type Hub struct {
	// ... existing fields
	cfTurn *cfTurnClient
}

// In NewHub() / construction:
h.cfTurn = newCFTurn()

// In the portal.create response builder:
resp := protocol.PortalCreated{
	Type:       protocol.TypePortalCreated,
	PortalID:   p.ID,
	Code:       p.Code,
	PeerID:     conn.peerID,
	VirtualIP:  vip,
	Capacity:   p.Capacity,
	ICEServers: h.cfTurn.ICEServers(ctx),  // ← add this line
}

// In the portal.join response builder (PortalJoined):
resp := protocol.PortalJoined{
	Type:       protocol.TypePortalJoined,
	PortalID:   p.ID,
	PeerID:     conn.peerID,
	VirtualIP:  vip,
	Peers:      roster,
	ICEServers: h.cfTurn.ICEServers(ctx),  // ← add this line
}
```

## Deploy

```bash
# /etc/systemd/system/portal-signaling.service yoki kompose:
Environment="CF_TURN_TOKEN_ID=bc793e8bc36a8fdb71496d85256e18ac"
Environment="CF_TURN_API_TOKEN=…"  # Cloudflare dashboardidan oling

systemctl restart portal-signaling
```

(Xavfsizlik: token'larni bu fayldan boshqa joyga yuborma. Agar leak
bo'lsa, Cloudflare dashboard → Calls → TURN'da rotate qil.)

## Test

```bash
# Server log'i yangi credentials so'rovi qilganini ko'rsatishi kerak:
journalctl -u portal-signaling -f | grep -i turn

# Client log'i (~/.portal/logs/portal-…log) endi:
#   "server ICE applied" count=1
# satrini chiqarishi kerak — ya'ni serverdan creds keldi va qo'llandi.
```

Endi har qanday yangi foydalanuvchi (Cloudflare hisobi yo'q, Settings'ni
bilmaydigan, telefon hotspot ortida) Portal'ni ochib ulanadi va ☁️ TURN
badge'i avtomatik paydo bo'ladi.
