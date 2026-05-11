# ARCHITECTURE — Portal Android client

Bu fayl Android client'ining ichki tuzilishi haqida. Asosiy loyiha
arxitekturasi (mesh topology, NAT, encryption layers) `../ARCHITECTURE.md` da.

This file covers the Android client's internals. The overall project
architecture (mesh topology, NAT, encryption layers) lives in
`../ARCHITECTURE.md`.

---

## Loyihaning kontekstdagi o'rni

Portal — peer-to-peer mesh networking app. Peer'lar 6 xonali ID + kod
almashadi, signal server orqali WebRTC handshake o'tkazib, keyin
to'g'ridan-to'g'ri mesh ustida gaplashadi.

`../client/` — reference Wails desktop client (Go backend + React/TS frontend).
`mobile/` — Android client. **Wire-compatible** desktop bilan: bir xil
protokol xabarlari, bir xil channel ID'lar, bir xil PBKDF2 + secretbox
parametrlari.

```
┌──────────────┐         ┌─────────────┐         ┌──────────────┐
│  desktop     │◄───────►│  signal     │◄───────►│  Android     │
│  (Wails+pion)│ wss     │  server     │ wss     │  (Compose+   │
│              │         │  .1pro.uz   │         │  stream-     │
│              │         │             │         │   webrtc)    │
└──────┬───────┘         └─────────────┘         └──────┬───────┘
       │                  signaling only:                │
       │                  offer / answer / ice           │
       │                                                 │
       └────────── direct WebRTC mesh ───────────────────┘
                  (DTLS-SRTP + app-layer NaCl secretbox)
```

Mobile peer Wails desktop peer'lar bilan bir portalda ishtirok eta oladi
— tekshirilgan: bir xil protokol turlari, bir xil channel ID'lar, bir
xil PBKDF2 + secretbox parametrlari.

---

## Paket layout

`app/src/main/java/uz/aihealth/portal_mobile/` ichi:

| Paket | Mirror'i | Maqsad |
| --- | --- | --- |
| `protocol/` | `../shared/protocol/messages.go` | `@Serializable` data class'lar + xabar turi diskriminatori (snake_case JSON tag'lar Go bilan o'zaro mos) |
| `crypt/` | `../client/crypt/` | PBKDF2-SHA256 KDF + libsodium secretbox (lazysodium-android orqali) |
| `signaling/` | `../client/signaling/` | OkHttp WebSocket → typed `SignalingEvent` Flow |
| `peer/` | `../client/peer/` | stream-webrtc `PeerConnection` + 4 ta negotiated data channel |
| `mesh/` | `../client/mesh/` | Orchestrator: handshake, heartbeat, chat, transfer dispatch, joiner reconnect |
| `transfer/` | `../client/transfer/` | Bo'lakli fayl uzatish wire format'i (FRAME_START/CHUNK/END/ABORT). Receiver diskka oqim qiladi (desktop hali ham xotirada buffer qiladi v1) |
| `turn/` | `../client/cloudflareturn.go` + `../client/mesh/manager.go` (TurnURL) | Cloudflare Calls TURN credentials minting + qo'lda TURN config + ICE-server resolver |
| `i18n/` | `../client/frontend/src/i18n/` | UZ/EN dictionary, Compose `t()` helper, `LocalLang` CompositionLocal |
| `data/` | (mobile-only) | `PortalSettings` — Preferences DataStore wrapper (taxallus, signal URL, til, TURN, oxirgi portallar) |
| `service/` | (mobile-only — Android lifecycle, Go counterpart yo'q) | `MeshService` foreground service — portal sessiyasi davomida process priority'ni saqlaydi |
| `ui/` | (Compose o'zicha — Go counterpart yo'q) | `PortalViewModel` + Welcome / Join / Portal / Settings / Logo composables; `QrUtils` |

---

## Channel ID'lar (negotiated)

Mobile va desktop bir xil channel ID'larni ishlatadi — ulanish ishlashi
shartiga ega:

| Channel | ID | Ordered | Reliability |
| --- | --- | --- | --- |
| `control` | 1 | yes | reliable |
| `chat` | 2 | yes | reliable |
| `transfer` | 3 | yes | reliable |
| `proxy` | 4 | no | maxRetransmits=0 (datagram) |

`peer/PortalPeerConnection.kt` — channel ta'rifi. Agar ID'lar yoki
reliability profili Go reference bilan farqlasa, peer'lar bir-birini
"data channel topilmadi" xatosiga olib boradi.

---

## Crypto layer

Ikki qatlamli, bir-biri ustiga:

1. **Transport-level: DTLS-SRTP** — WebRTC ichida avtomatik. Stream-webrtc
   tomonidan boshqariladi.
2. **Application-level: NaCl secretbox** — har bir chat/file/proxy
   payload `crypt.seal(key, plaintext)` orqali o'raladi. Kalit 6 xonali
   portal kod'idan PBKDF2-SHA256 (200,000 iteration, salt
   `"portal-app-v1:secretbox"`) bilan olinadi.

Konstantalar `crypt/Crypt.kt` da. **MUTLAQO** `client/crypt/crypt.go`
bilan mos kelishi kerak — aks holda peer'lar bir-birini decrypt qila
olmaydi. Hozirgi parametrlar:

- KDF: PBKDF2-SHA256
- Iteration: 200,000
- Salt: `"portal-app-v1:secretbox"` (UTF-8)
- Output: 32-byte key (NaCl secretbox kalit hajmi)
- Nonce: 24-byte random per frame (secretbox talabi)

> **Reja:** v0.5.0 da PCP-1 (Portal Cipher Protocol v1) ga o'tamiz —
> Noise XX handshake'iga asoslangan, forward secrecy bilan. Hozirgi
> "shared key from code" pattern oddiy va ishlaydi, lekin yangi peer
> qo'shilganda eski xabarlar replay qilinishi nazariy mumkin.

---

## Til (i18n) layeri

Desktop client zustand-backed `useT()` hook ishlatadi (`../client/frontend/src/i18n/`).
Mobile uchun bu Compose'da CompositionLocal sifatida amalga oshirildi:

```kotlin
// MainActivity.kt
CompositionLocalProvider(LocalLang provides lang) {
    NavHost(...)
}

// any composable
val nicknameLabel = t("welcome.nickname.label")
```

`LocalLang.current` qaysi til faollashtirilganini bildiradi
(default: UZ). `Strings.kt` ichida ikki `Map<String, String>`:
`uz` va `en`. Yo'qolgan kalit EN ga fallback, keyin raw key —
UI hech qachon crash qilmaydi.

Fokuslangan yondashuv: 200 ta string uchun react-i18next yoki
Android'ning resources XML pipeline'i ortiqcha. Yagona Kotlin fayl
type-safe, hot-swappable, shaffof.

---

## TURN layeri

Mobile networks (CGNAT, mobile internet) odatda Symmetric NAT ortida
bo'ladi — to'g'ridan-to'g'ri WebRTC ulanish ishlamaydi. TURN server
ma'lumotni o'tkazib beradi.

`turn/IceServerResolver.kt` ICE servers ro'yxatini quyi-yuqori tartibda
quradi:

1. Default STUN servers (har doim — srflx candidates beradi)
2. **Cloudflare TURN** (agar sozlangan VA API javob qaytarsa)
3. **Manual TURN** (agar sozlangan)

Cloudflare path: `turn/CloudflareTurn.kt` `https://rtc.live.cloudflare.com/v1/turn/keys/{TOKEN_ID}/credentials/generate`
ga POST qiladi, qisqa muddatli `username` + `credential` qaytaradi (3600s).
30 daq ichki kesh — har portal yaratish/qo'shilishda API ga zarba bermaslik
uchun.

Resolved ICE servers `MeshManager` constructor'iga `iceServers` parametr
sifatida injected qilinadi, keyin `PortalPeerConnection` ni qurganda
ishlatiladi.

---

## Lifecycle: foreground service

Android backgrounded app'lar WebSocket / WebRTC ulanishlarini
sekundlarda o'ldiradi. `service/MeshService.kt` (foregroundServiceType=`dataSync`)
portal sessiyasi davomida process priority'ni saqlaydi.

`MeshManager` ichidagi mesh logikasi shu service'ga bog'liq emas — service
faqat process priority anchor. Manifest'da quyidagilar shart:

- `FOREGROUND_SERVICE`
- `FOREGROUND_SERVICE_DATA_SYNC`
- `POST_NOTIFICATIONS` (Android 13+)

`PortalViewModel.createPortal()` / `joinPortal()` MeshService'ni boshlaydi,
`leave()` to'xtatadi.

---

## Reconnect logikasi (joiner uchun)

`mesh/MeshManager.kt` ichida `runReconnectLoop()`:

- 8 marta urinish
- Exponential backoff: 1s → 2s → 4s → 8s → 16s → 32s → 60s → 60s
- Har urinishda yangi `SignalingClient` qo'rib, eski portal ID + kod
  bilan qayta `joinPortal` chaqiriladi
- Server `Joined` qaytarsa — muvaffaqiyat; mavjud peer'lar bilan
  qayta handshake `peer_joined` push'lari orqali avtomatik

Owner uchun reconnect **yo'q** — server portal'ni egasi disconnect
bo'lganda buzadi.

`../client/mesh/manager.go attemptReconnect` bilan oyna oydek mos.

---

## Test'lar

| Test | Joyi | Run buyrug'i |
| --- | --- | --- |
| Protocol JSON encode/decode | `app/src/test/.../protocol/` | `./gradlew test` |
| Transfer wire format | `app/src/test/.../transfer/` | `./gradlew test` |
| QR-invite parser | `app/src/test/.../ui/` | `./gradlew test` |
| Crypt (secretbox round-trip) | `app/src/androidTest/.../crypt/` | `./gradlew connectedAndroidTest` |

Crypt **host JVM da test qilinmaydi** — lazysodium-android faqat Android
ABI'lariga `.so` deploy qiladi. Fizik qurilma yoki emulyator kerak.

---

## Mobile va desktop farqlari (ataylab)

Quyidagilar mobile'da boshqacha — bu xato emas, atayyob:

| Hodisa | Desktop | Mobile | Nima uchun |
| --- | --- | --- | --- |
| Background life | har doim aktiv | foreground service kerak | Android process killer |
| Save dir | `~/Documents/Portal/` | App-private external storage | runtime permission'sizdir |
| Crash reports | local SQLite | Logcat (hozir) | mobile crash UI mavjud emas |
| Auto-update | embedded updater | qo'lda yoki Play Store | Play Store distributsiyasini kutamiz |
| Logs viewer | in-app | Logcat / Android Studio | mobile UI hali yo'q |
| Activity log | in-app | yo'q | proxy hali implement qilinmagan |

Bu farqlar kelajakda kamayadi (PCP-1, Logs viewer, NAT diagnostics —
keyingi versiyalarda).
