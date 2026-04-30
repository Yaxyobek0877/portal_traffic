# Build from source

The desktop app isn't packaged for users yet (Phase 2 in
[ROADMAP.md](ROADMAP.md)). If you want to build the project today, you'll
need a Go toolchain and the platform prerequisites for [Wails](https://wails.io)
on your OS.

> Once Phase 2 ships, prebuilt installers for macOS, Windows, and Linux
> will land on the Releases page and this file will become a "for
> contributors" guide.

---

## 1. Prerequisites

### All platforms

- **Go 1.22+** — `https://go.dev/dl/`
- **Git**

### macOS

```sh
xcode-select --install
brew install node          # for the React frontend (Phase 2)
```

### Linux (Debian / Ubuntu)

```sh
sudo apt update
sudo apt install -y \
    build-essential pkg-config \
    libgtk-3-dev libwebkit2gtk-4.1-dev \
    nodejs npm
```

On Fedora:

```sh
sudo dnf install -y gcc-c++ pkg-config gtk3-devel webkit2gtk4.1-devel nodejs npm
```

### Windows

- Install **Go** from the official MSI installer.
- Install [WebView2 Runtime](https://developer.microsoft.com/en-us/microsoft-edge/webview2/)
  (preinstalled on Windows 11).
- Install [Node.js LTS](https://nodejs.org/).
- Install [Git for Windows](https://git-scm.com/download/win).

---

## 2. Clone

```sh
git clone https://github.com/<your-username>/portal-traffic.git
cd portal-traffic
```

---

## 3. Build the bits available today

### The shared protocol package

```sh
cd shared
go test ./...
```

That's it — `shared/` has no binary. It's imported by the client.

### The desktop app (Phase 2 — coming up)

When `client/` lands, the build will look like this. Run from the repo root:

```sh
# Install Wails CLI once
go install github.com/wailsapp/wails/v2/cmd/wails@latest

# Development mode (live reload)
cd client
wails dev

# Production build for your current OS
wails build -clean
# → produces client/build/bin/Portal[.app|.exe|]
```

Cross-platform builds:

```sh
wails build -platform darwin/universal     # macOS .app
wails build -platform windows/amd64        # Windows .exe
wails build -platform linux/amd64          # Linux binary
```

The resulting bundles are self-contained and don't require Go or Node
on the user's machine.

---

## 4. Configure the signaling URL

Portal clients connect to a signaling server you configure. You can:

1. **Use the project's hosted signaling endpoint** — default in the
   shipped binary. No setup required.
2. **Self-host.** The wire protocol is documented in
   [PROTOCOL.md](PROTOCOL.md); a compatible server is a few hundred
   lines of Go. Once running, point the client at it via Settings →
   Network → Signaling URL, or set the env var:
   ```sh
   export PORTAL_SIGNALING_URL=wss://your-host/ws
   ```

---

## 5. Verify the build

After building the client (Phase 2+):

```sh
./build/bin/Portal      # or open Portal.app on macOS
```

You should see the welcome screen with a wormhole logo, a nickname
input, and the **Create Portal / Join Portal** buttons. If you have the
project's signaling URL configured, click **Create Portal** — within a
second or two you should get a 6-digit ID and code.

Hand the ID + code to a second device running Portal, click **Join
Portal**, and watch the mesh diagram light up.

---

## Troubleshooting

| Symptom | Likely cause | Fix |
| --- | --- | --- |
| `wails: command not found` | Wails CLI not installed | `go install github.com/wailsapp/wails/v2/cmd/wails@latest`; ensure `$GOPATH/bin` is on `PATH` |
| Stuck on "Connecting…" | Signaling URL wrong or unreachable | Check `wss://` not `ws://`, verify cert isn't expired |
| Peer shows 🔴 (failed) for >30s | NAT can't be traversed; no TURN configured | See [ARCHITECTURE.md § NAT & TURN fallback](ARCHITECTURE.md#nat--turn-fallback) |
| Linux: `Package webkit2gtk-4.1 not found` | Older distro ships `webkit2gtk-4.0` | Try `libwebkit2gtk-4.0-dev` instead |
