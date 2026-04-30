<div align="center">

# Portal

**Direct connections. Zero servers between you.**

Portal is a desktop app that builds a private peer-to-peer mesh network
between people. Open a portal, share a 6-digit code, and every device
inside connects directly to every other device — chat, transfer files,
host a game server, share a dev URL. The signaling server only carries
the initial handshake; everything after that is peer-to-peer.

</div>

---

## What you can do with it

| | |
| --- | --- |
| **Connect 2 – 16 devices** | Open a portal, hand out the 6-digit ID + code. Anyone can join from anywhere on the internet — no router config, no VPN, no port forwarding. |
| **Chat & file transfer** | Group chat, direct messages, drag-and-drop file transfer with progress bars. End-to-end encrypted with a key derived from your portal code. |
| **Tunnel any TCP/UDP** | Each peer gets a virtual IP in `10.42.0.0/24`. Run a Minecraft server on `localhost:25565` and your friends connect at `10.42.0.3:25565` — Portal proxies the bytes through the mesh. |
| **See your mesh live** | Animated visualization of who's connected to whom, real-time round-trip times, NAT-traversal indicators (direct vs. relayed). |
| **Stay private** | After the WebRTC handshake, the signaling server never sees your traffic. All app data flows P2P. |

> **Status:** the signaling backbone is live. The desktop client is in
> active development — see [ROADMAP.md](ROADMAP.md).

---

## Install

> Builds for the desktop app are not yet released. Once Phase 2 ships, the
> recommended path will be:
>
> 1. Download the installer for your OS from the **Releases** page.
> 2. Open it, type a nickname, click **Create Portal** or **Join Portal**.
>
> No CLI, no setup, no account.

While we get there, you can build from source — see [INSTALL.md](INSTALL.md).

### Required on your machine

| Platform | What you need | Why |
| --- | --- | --- |
| Any  | Internet connection that supports WebRTC (almost all do) | Mesh transport |
| macOS 12+ | Xcode Command Line Tools | code signing, native webview |
| Windows 10+ | WebView2 runtime (preinstalled on 11) | embedded UI |
| Linux | `webkit2gtk-4.1`, `libgtk-3` | embedded UI |

If you're behind a strict NAT (CGNAT or symmetric), connections will
auto-fall-back to a TURN relay — see [ARCHITECTURE.md](ARCHITECTURE.md#nat--turn-fallback).

---

## How a session looks

```
1. You open the app, type a nickname.
2. Click Create Portal.
3. Portal shows you:           ID:   428591
                                CODE: 739204
   (or scan a QR code from the modal)
4. Friend opens the app, types nickname, clicks Join Portal.
5. They type your ID + code. They're in.
6. Live mesh diagram: every peer connected to every other peer, directly.
```

---

## Documentation

| Doc | What's in it |
| --- | --- |
| [ROADMAP.md](ROADMAP.md) | Phases 1–5 with current status |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Mesh topology, NAT traversal, encryption, threat model |
| [PROTOCOL.md](PROTOCOL.md) | Wire protocol reference for both signaling and P2P channels |
| [INSTALL.md](INSTALL.md) | Build-from-source instructions per platform |
| [CONTRIBUTING.md](CONTRIBUTING.md) | How to file issues, send patches, run tests |

---

## Repository layout

```
portal_traffic/
├── shared/        Wire protocol shared by every client (this is here)
├── client/        Wails + React desktop app (Phase 2 — coming up)
└── server/        Signaling server (private — runs on a single VPS we operate)
```

The signaling server source is intentionally not part of this repository.
Portal clients connect to whichever signaling URL they're configured for,
so the server is effectively interchangeable. If you want to host your
own, the wire protocol is fully documented in [PROTOCOL.md](PROTOCOL.md)
and [`shared/protocol/messages.go`](shared/protocol/messages.go) — a
compatible signaling server is a few hundred lines of Go.

---

## License

License TBD. Until one is chosen, this code is provided for reference;
contact the maintainer before redistributing.
