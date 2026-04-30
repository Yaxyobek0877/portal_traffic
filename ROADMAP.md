# Roadmap

Portal is being built in five phases. Each one is a working stop —
we don't move on until the previous phase verifiably works.

## Phase 1 — Signaling backbone ✅

The thin server that brokers WebRTC handshakes. After this phase, two
clients can find each other on the internet given just a 6-digit ID +
code; everything they do afterwards is direct.

- [x] WebSocket transport with per-IP rate limiting
- [x] Portal lifecycle: create / join / kick / lock / leave
- [x] Nickname directory with public/private/friends-only visibility
- [x] WebRTC SDP/ICE relay, scoped strictly within a portal
- [x] Server-stamped `from` field on every relayed frame (anti-spoof)
- [x] TLS support (Cloudflare Origin Certificate or any PEM/key pair)
- [x] systemd hardening profile + deployment guide
- [x] Integration test suite + scripted end-to-end demo

The signaling server source is private (runs on a VPS we operate). The
wire protocol it implements is fully documented in [PROTOCOL.md](PROTOCOL.md)
so anyone can host their own.

## Phase 2 — Client core (in progress)

Go-based mesh engine, no UI yet. The goal at the end of this phase is
that two CLI test clients on different networks can:

- [ ] Open a Wails project skeleton (`client/`)
- [ ] Connect to the signaling URL over secure WebSocket
- [ ] Run the WebRTC handshake using `pion/webrtc/v4`
- [ ] Establish a full mesh between every peer
- [ ] Multiplex four data channels: `control`, `chat`, `transfer`, `proxy`
- [ ] Heartbeat ping/pong with round-trip-time tracking
- [ ] Auto-reconnect with exponential backoff
- [ ] Run a CLI smoke test: two clients exchange `ping` over the
      `control` channel and report measured RTT

## Phase 3 — Desktop UI

The pieces a normal user sees.

- [ ] Welcome screen (animated wormhole logo, nickname input,
      Create / Join buttons)
- [ ] Portal view (header with ID + code + QR, peer sidebar, animated
      mesh diagram, chat panel, status bar)
- [ ] Live mesh visualization (SVG, Framer Motion, glow on data flow)
- [ ] Wails bindings between Go backend and React frontend
- [ ] Dark mode default, light mode toggle

## Phase 4 — Polish

- [ ] QR code generation + scan modal
- [ ] STUN-based NAT type detection on startup, banner for symmetric/CGNAT
- [ ] Drag-and-drop file transfer with progress bars
- [ ] Settings page (network, privacy, appearance, diagnostics)
- [ ] Auto-reconnect surfaced in UI
- [ ] SQLite persistence for settings, portal history, saved contacts
- [ ] Bandwidth graph in settings

## Phase 5 — Power features

- [ ] Local TCP/UDP proxy mapping virtual IPs to the WebRTC `proxy`
      channel (`portal expose tcp 25565`)
- [ ] Services panel showing exposed ports across the mesh
- [ ] Voice channel with push-to-talk
- [ ] Built-in mini-games (tic-tac-toe over data channel) as
      proof-of-concept demos

---

After Phase 5, the project is feature-complete relative to the original
spec. Beyond that, likely directions:

- Mobile companion (iOS / Android, view-only at first)
- Federated discovery (publish portal IDs over an opt-in DHT so people
  can find a friend's portal without sharing the 6-digit code)
- Plugin API for third-party apps to expose services through the mesh
