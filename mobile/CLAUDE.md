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

Stage-zero scaffold. `app/src/main/java/uz/aihealth/portal_mobile/MainActivity.kt` is the default Compose "Hello Android" template; theme files are the Studio-generated defaults. No Portal protocol, signaling, WebRTC, or crypto code has been written yet — assume work here is new implementation, not modification.

## Toolchain pins (mismatches will break builds)

- Gradle daemon JVM toolchain **21** (`gradle/gradle-daemon-jvm.properties` — auto-provisioned via foojay; don't replace with system Java 17).
- AGP **9.0.1** + Kotlin **2.0.21** + Compose BOM 2024.09.00. AGP 9 is canary-track; this requires Android Studio on the canary channel.
- `compileSdk = release(36) { minorApiLevel = 1 }` — uses the new minor-API-level DSL; older AGP versions can't parse it.
- `minSdk 26`, `targetSdk 36`, namespace/applicationId `uz.aihealth.portal_mobile`.
- Versions are centralized in `gradle/libs.versions.toml`. Add new deps there, not as inline `"group:name:ver"` strings.

`local.properties` is gitignored and supplies `sdk.dir`. Don't commit it.

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
- **Encryption is two layers, not one.** WebRTC DTLS-SRTP is automatic; on top of that, app-layer NaCl secretbox wraps every chat/file/proxy payload, with the key derived from the 6-digit portal code via PBKDF2 with **100,000** iterations. See `../client/crypt/` — match the KDF parameters or peers can't decrypt each other.
- **Default signal endpoint:** `wss://signaling.1pro.uz/ws`. Make it overridable in settings, same as the desktop client.
- **WebRTC on Android:** the standard option is `org.webrtc:google-webrtc`. The reference client uses pion in Go — APIs differ but the SDP/ICE flow on the wire is identical.
