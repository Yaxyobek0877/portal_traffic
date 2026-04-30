# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Where this fits in the larger repo

The git root is `..`. This `mobile/` directory is the Android sub-project of a workspace called **Portal** — a peer-to-peer mesh networking app where peers exchange a 6-digit ID + code, do a WebRTC handshake through a signal server, and then talk directly over a mesh. Sibling directories you will need to read often:

- `../shared/protocol/messages.go` — the wire protocol. **Source of truth** for every client (desktop, mobile, third-party). `../PROTOCOL.md` is the prose version.
- `../client/` — the reference desktop client (Wails + Go backend + React/TS frontend). When you implement protocol behavior here, mirror it: `client/signaling/` (WebSocket → signal server), `client/peer/` (WebRTC handshake on `pion/webrtc/v4`), `client/mesh/` (per-peer mesh + heartbeat RTT), `client/crypt/` (NaCl secretbox + PBKDF2 from portal code), `client/transfer/`, `client/proxy/`.
- `../ARCHITECTURE.md` — mesh topology, NAT/TURN, encryption layers, threat model.
- `../ROADMAP.md` — situates this module: the mobile app is a post-Phase-5 companion ("dastlab faqat ko'rish rejimida" — view-only initially).

Most prose docs are in Uzbek. Match the existing language when editing them; don't translate to English.

Backwards compatibility on `shared/protocol/messages.go` is a hard contract: only **add** optional fields. Renaming or removing fields breaks every deployed client.

## Current state

Phase-1 mesh client implemented. `./gradlew assembleDebug` produces a working APK. The on-the-wire behavior is byte-compatible with the desktop client (verified: same protocol message types, same channel IDs, same PBKDF2 + secretbox parameters), so an Android peer can join a portal alongside Wails desktop peers.

Working features: create / join portal, full-mesh WebRTC handshake, encrypted broadcast chat, RTT heartbeat, peer roster, **chunked file transfer** (sender streams via `ContentResolver.openInputStream`; receiver streams to a `.part` file on app-private external storage and renames on END), **QR display + scan** (desktop-compatible plaintext invite format), **DataStore-backed settings** (nickname + signal URL + last 10 portals for one-tap rejoin), **auto-reconnect for joiners** (8 attempts, exponential backoff up to 60 s, mirrors `../client/mesh/manager.go attemptReconnect`), **foreground service** (`service/MeshService.kt`) that anchors process priority while in a portal so backgrounding doesn't kill the WebSocket / WebRTC connections, **settings screen** for signal-URL override.

Tested: protocol JSON encode/decode round-trips with Go-compatible snake_case (11 tests), transfer frame wire format (7 tests), QR-invite parser (7 tests). Run with `./gradlew :app:testDebugUnitTest`. **Crypt is NOT host-JVM-testable** — lazysodium-android only ships .so for Android ABIs; verify on device or with an instrumented test.

Not yet implemented (would mirror named files under `../client/`): TCP/UDP proxy (`proxy/`), service expose UI, NAT-type detection, join-by-nick.

Source layout under `app/src/main/java/uz/aihealth/portal_mobile/`:

| Package | Mirrors | Purpose |
| --- | --- | --- |
| `protocol/` | `../shared/protocol/messages.go` | `@Serializable` data classes + type discriminator |
| `crypt/` | `../client/crypt/` | PBKDF2-SHA256 KDF + libsodium secretbox via lazysodium-android |
| `signaling/` | `../client/signaling/` | OkHttp WebSocket → typed `SignalingEvent` Flow |
| `peer/` | `../client/peer/` | Stream-WebRTC `PeerConnection` + 4 negotiated data channels |
| `mesh/` | `../client/mesh/` | Orchestrator: handshake, heartbeat, chat, transfer dispatch, joiner reconnect |
| `transfer/` | `../client/transfer/` | Chunked file transfer wire format (FRAME_START/CHUNK/END/ABORT). Receiver streams to disk (the desktop still buffers in memory in v1). Frame encode/decode helpers at top level are `internal` so the unit tests can exercise them. |
| `data/` | (mobile-only) | `PortalSettings` — Preferences DataStore wrapper for nickname / signal URL / recent portals |
| `service/` | (mobile-only — Android lifecycle concern, no Go counterpart) | `MeshService` foreground service that anchors process priority while a portal session is active |
| `ui/` | (no Go counterpart — Compose is its own thing) | `PortalViewModel` + Welcome / Join / Portal / Settings screens; `QrUtils` for invite gen + scan-result parsing |

## Toolchain pins (mismatches will break builds)

- Gradle daemon JVM toolchain **21** (`gradle/gradle-daemon-jvm.properties` — auto-provisioned via foojay; don't replace with system Java 17).
- AGP **9.0.1** + Kotlin **2.0.21** + Compose BOM 2024.09.00. AGP 9 is canary-track; this requires Android Studio on the canary channel.
- `compileSdk = release(36) { minorApiLevel = 1 }` — uses the new minor-API-level DSL; older AGP versions can't parse it.
- `minSdk 26`, `targetSdk 36`, namespace/applicationId `uz.aihealth.portal_mobile`.
- Versions are centralized in `gradle/libs.versions.toml`. Add new deps there, not as inline `"group:name:ver"` strings.

`local.properties` is gitignored and supplies `sdk.dir`. Don't commit it.

This system has no JDK on `PATH` — Gradle's bootstrap launcher needs one before the daemon-JVM provisioner kicks in. Use Android Studio's bundled JBR:

```bash
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
./gradlew assembleDebug
```

(`gradle/gradle-daemon-jvm.properties` only governs the daemon, not the launcher itself.)

`lazysodium-android` and `jna` must be declared with the `@aar` packaging classifier as direct strings in `app/build.gradle.kts` (not via the version catalog) — without `@aar` Gradle resolves the JAR variant of JNA which collides with its AAR variant on Android. See the inline comment in `app/build.gradle.kts`.

## Commands

Always use the wrapper:

```bash
./gradlew assembleDebug              # build debug APK
./gradlew installDebug               # install on connected device/emulator
./gradlew lint                       # Android lint
./gradlew test                       # all unit tests (host JVM)
./gradlew connectedAndroidTest       # all instrumented tests (device required)

# single unit test:
./gradlew :app:testDebugUnitTest --tests "uz.aihealth.portal_mobile.ExampleUnitTest.addition_isCorrect"

# single instrumented test class:
./gradlew :app:connectedDebugAndroidTest \
  -Pandroid.testInstrumentationRunnerArguments.class=uz.aihealth.portal_mobile.ExampleInstrumentedTest
```

There is no Go or Node tooling in this directory; those live in `../client/` and `../web/`.

## When implementing Portal features on Android

- **Read `../shared/protocol/messages.go` first.** Field names, JSON tags, and message types are what the live signal server already emits — mirror them exactly.
- **Channel IDs/labels are negotiated, not auto-assigned.** The reference client uses fixed IDs for `control`, `chat`, `transfer`, `proxy`. Peers across clients won't talk if these drift.
- **Encryption is two layers, not one.** WebRTC DTLS-SRTP is automatic; on top of that, app-layer NaCl secretbox wraps every chat/file/proxy payload, with the key derived from the 6-digit portal code via PBKDF2-SHA256 with **200,000** iterations and salt `"portal-app-v1:secretbox"` (32-byte key, 24-byte random nonce per frame). See `../client/crypt/crypt.go` — these constants must match exactly or peers can't decrypt each other.
- **Default signal endpoint:** `wss://signaling.1pro.uz/ws`. Make it overridable in settings, same as the desktop client.
- **WebRTC on Android:** uses `io.getstream:stream-webrtc-android` (an actively maintained fork of Google's library, classes still in the `org.webrtc.*` namespace). The reference client uses pion in Go — APIs differ but the SDP/ICE flow on the wire is identical.
- **Foreground service is required for non-trivial sessions.** Android kills backgrounded apps' WebSocket/WebRTC connections within seconds. The `MeshService` (foregroundServiceType=`dataSync`) keeps the process alive while in a portal. Ensure `FOREGROUND_SERVICE` + `FOREGROUND_SERVICE_DATA_SYNC` + `POST_NOTIFICATIONS` permissions are in the manifest. The mesh logic itself stays in `MeshManager` — the service is just a process-priority anchor.
