# Architecture

This document explains how Portal works under the hood. For the ground
truth, the code is the spec — start at
[`shared/protocol/messages.go`](shared/protocol/messages.go).

---

## Design goals

1. **Direct connections by default.** Once two peers have shaken hands,
   their packets must not pass through a server we operate. Latency,
   privacy, and cost all benefit.
2. **The signaling server is replaceable.** It's a thin broker that
   matches peers and forwards SDP/ICE. Anyone can host one; clients
   point at whichever URL they like.
3. **The signaling server can never read application traffic.** It
   only sees JSON envelopes for handshakes. WebRTC's DTLS plus an
   app-layer secretbox derived from the portal code give us two
   independent layers of protection.
4. **A peer in portal A cannot probe peers in portal B.** Even though
   the server connects every peer over a single WebSocket, the relay
   path explicitly checks portal membership before forwarding.

---

## High-level diagram

```
┌──────────────────────────────────────────────┐
│         Signaling Server (Go, private)       │
│  - Portal registry (6-digit ID + code)       │
│  - Nickname directory (opt-in public)        │
│  - WebRTC SDP/ICE relay (handshake only)     │
└──────────────────┬───────────────────────────┘
                   │ WebSocket TLS (handshake only)
        ┌──────────┴──────────┐
        │                     │
   ┌────▼────┐  ◄── P2P ───►  ┌────▼────┐
   │ Peer A  │   WebRTC mesh  │ Peer B  │
   └────┬────┘                └────┬────┘
        │   ▲           ▲           │
        │   └─── P2P ───┘           │
        │       ┌─────┐             │
        └──────►│  C  │◄────────────┘
                └─────┘
```

Once the mesh is established, every pair of peers has a direct
WebRTC connection carrying four multiplexed data channels:

| Channel | Reliability | Carries |
| --- | --- | --- |
| `control` | reliable, ordered | heartbeat, presence, service announcements |
| `chat` | reliable, ordered | text messages, both group and DM |
| `transfer` | reliable, ordered | chunked file transfers |
| `proxy` | unreliable, unordered | tunneled TCP/UDP application traffic |

---

## Joining a portal: the full sequence

```
Bob                            Server                            Alice (owner)
 │                                │                                │
 ├── portal.join(id, code) ──────▶│                                │
 │                                │                                │
 │                                │  validate id + code            │
 │                                │  allocate virtual IP           │
 │                                │                                │
 │◀── portal.joined(peers) ───────┤                                │
 │                                ├── portal.peer_joined(bob) ────▶│
 │                                │                                │
 │                                │                                │
 │  for each existing peer P:                                      │
 │     ├── webrtc.offer(to=P) ──▶ ├── webrtc.offer(from=bob) ─▶ P  │
 │     │◀── webrtc.answer ◀────── ├──◀── webrtc.answer ────────── P│
 │     │── ICE candidates ──────▶ ├── ICE candidates ───────────▶ P│
 │     │      (until ICE done)                                     │
 │     │                                                           │
 │     └── direct WebRTC connection up                             │
 │            ↓                                                    │
 │       open data channels (control, chat, transfer, proxy)       │
```

Once the data channels are open, all subsequent traffic flows directly
over them. The signaling server has no further role in this session.

---

## Component map

```
shared/
└── protocol/                Wire format — every message type, every payload struct
                              Source of truth for both sides; bumping it must be
                              backward-compatible (only add optional fields).

server/                       Private. Lives on a VPS behind Cloudflare.
├── main.go                  Entry, TLS listener, /ws + /healthz
├── connection.go            Per-WS read/write pumps, ping/pong handlers,
│                              CF-Connecting-IP / X-Forwarded-For honoring
├── hub.go                   Connection registry, broadcast helpers,
│                              pending-join tracking, GC ticker
├── portal.go                6-digit ID generation (crypto/rand), members,
│                              virtual IP allocation in 10.42.0.0/24
├── nickname.go              Public/private nickname directory, IP-hash hint
├── handlers.go              Message dispatch + WebRTC relay (server stamps
│                              `from` so peers can't lie about origin)
├── ratelimit.go             Per-IP token buckets (golang.org/x/time/rate)
└── deploy/                  systemd unit + Cloudflare-fronted deploy guide

client/                       Public. Wails + React desktop app.
└── (Phase 2)
```

---

## Concurrency model (server)

There is one read goroutine and one write goroutine per WebSocket. The
read goroutine deserialises frames and calls into hub methods directly;
the hub uses `sync.RWMutex` per registry plus per-portal locks. There
is no central event loop — the handler and the goroutine that read the
frame are the same goroutine, which keeps tracing simple.

Outbound frames are funnelled through a per-connection buffered channel
(`send chan []byte`, default 64 slots). If a client is too slow and the
channel fills, the connection is dropped — we'd rather kick a stuck
client than back-pressure the whole hub.

---

## NAT & TURN fallback

WebRTC's ICE collects candidates from three sources:

1. **Host candidates** — the local IP/port. Works on the same LAN.
2. **Server-reflexive candidates** — public IP/port discovered via STUN.
   Most home/office NATs work here.
3. **Relayed candidates** — through a TURN server. Required for
   symmetric NAT and most CGNAT.

Portal will ship with multiple STUN servers configured by default
(Google, Cloudflare). For TURN we recommend self-hosting `coturn` on a
small VPS with a couple of GB of bandwidth budget. The desktop app
shows a small "relayed" indicator next to peers whose connection went
through TURN, and the status bar surfaces a one-line warning if the
local NAT type would otherwise hide it.

NAT type detection happens at app startup using STUN; the result is
cached in settings until the network changes.

---

## Encryption

Two independent layers protect peer-to-peer traffic:

1. **WebRTC DTLS-SRTP.** Default for all data-channel traffic. Keys are
   negotiated during the SDP handshake. Even an attacker who compromises
   the signaling server cannot decrypt this, because they don't get the
   ephemeral DTLS keys — those are derived from the SDP fingerprint
   exchange.
2. **App-layer secretbox.** All chat, file, and proxy payloads are
   additionally sealed with NaCl secretbox using a key derived from the
   portal code via PBKDF2 (100k iterations). This protects against a
   *malicious peer inside the portal* abusing data-channel access — they
   still need the portal code to read anything.

The portal code is short on purpose: it's a six-digit number that
travels over a side channel (chat, voice, paper) outside Portal itself.
That short code becomes the second-factor secret.

---

## Threat model

| Adversary | What they get | What they don't |
| --- | --- | --- |
| Network observer between peer and server | Sees TLS-wrapped WebSocket frames | Cannot read SDP after TLS |
| Compromised signaling server | Sees handshakes, peer IDs, IPs, nicknames | Cannot read DTLS-protected app traffic |
| Random scanner on internet | Sees open `/ws` endpoint | No portal access without ID + code |
| Brute-forcer guessing codes | Rate-limited at 30 join/min/IP, 1M codespace | Realistically ~10 years to half-exhaust |
| Malicious peer inside a portal | Can chat & receive files (they joined!) | Cannot impersonate other peers (server-stamped `from`); cannot reach peers in other portals |
| User who lost their device | Worst case: future portals owned by them | DTLS keys are ephemeral; old session traffic isn't recoverable |

What's *not* defended:

- A motivated attacker who is in the portal and has the code can do
  everything any honest peer can do. Don't share codes with strangers.
- Cloudflare (if you use it as a proxy) sees the TLS-wrapped traffic
  between client and origin. Use **Full (strict)** mode + Origin
  Certificates so the Cloudflare↔origin hop is also encrypted.
- Side-channel timing attacks against PBKDF2 are not in scope; the
  attacker who wants to brute-force portal codes is rate-limited
  far below useful throughput regardless.
