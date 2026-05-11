# Portal Mobile (Android)

Bu papka Portal'ning Android client'i. Desktop versiya bilan to'liq simli
protokol mosligiga ega — Android peer Wails desktop peer'lari bilan bir
xil portalda ishtirok eta oladi.

This directory holds Portal's Android client. Wire-compatible with the
desktop client; an Android peer can join a portal alongside Wails
desktop peers.

---

## Tezkor havolalar / Quick links

- [INSTALL.md](INSTALL.md) — APK ni telefonga yuklash (sideload)
- [BUILDING.md](BUILDING.md) — manbadan qurish, toolchain, xatolar
- [ARCHITECTURE.md](ARCHITECTURE.md) — paket-paket Go reference bilan moslik
- [CLAUDE.md](CLAUDE.md) — Claude Code uchun loyiha qoidalari

---

## Hozirgi imkoniyatlar / Current features

- **Portal yaratish va qo'shilish** (6 xonali ID + kod) / Create and join portal
- **To'liq mesh WebRTC handshake** — desktop bilan birga ishlaydi
- **Shifrlangan broadcast chat** (PBKDF2-SHA256 + NaCl secretbox)
- **RTT heartbeat** va peer ro'yxati
- **Bo'lakli fayl uzatish** — `.part` faylga oqim, END kelganda renaming
- **QR kod ko'rsatish va skanerlash** (desktop bilan mos plaintext format)
- **TURN qo'llab-quvvatlash** — Cloudflare Calls TURN (token‑id + api‑token)
  va qo'lda TURN (URL/user/pass). Simmetrik NAT (CGNAT, mobile internet)
  ortida ishlash uchun shart
- **Til tugmasi** — UZ/EN, har lahza almashtiriladi (DataStore'da saqlanadi)
- **Yaqindagi portallar** — oxirgi 10 ta, bir tegishda qayta kirish
- **Joiner uchun avtomatik qayta ulanish** — 8 urinish, exponential backoff
  (60s gacha), `client/mesh/manager.go attemptReconnect` ga oyna oydek mos
- **Foreground service** (`MeshService`) — backgroundga ketganda
  WebSocket/WebRTC ulanishlarini saqlaydi
- **Adaptive launcher icon** — wormhole motivli, monochrome variant ham
  bor (Android 13+ themed icons)

## Hali yo'q / Not yet implemented

| Mobile | Desktop'dagi nom | Sabab |
| --- | --- | --- |
| TCP/UDP proxy ekspoz | `client/proxy/` | UI yo'q; backend yozilishi kerak |
| Servislar paneli | `ServicesPanel.tsx` | proxy yo'q ekan, UI ham yo'q |
| NAT turi aniqlash | `client/nat/` | mobile'da STUN classify keyinroq |
| Join-by-nick | server tomon yangi RPC | server protokoli kengaytirilishi kerak |
| Loglar viewer + krash hisobotlari | desktop About panel | mobile'da Logcat'dan o'qiladi |
| PCP-1 (encryption v2) | `client/crypt/pcp/` | desktop v0.4.0'da; mobile v0.5.0 da |

---

## Sinab ko'rish / Quick start

### 1. APK ni qurish (manbadan)

```bash
export JAVA_HOME="/Applications/Android Studio.app/Contents/jbr/Contents/Home"
cd mobile
./gradlew assembleDebug
```

APK joyi: `app/build/outputs/apk/debug/app-debug.apk` (~50 MB).

Batafsil: [BUILDING.md](BUILDING.md).

### 2. Telefoniga yuklash

```bash
adb install app/build/outputs/apk/debug/app-debug.apk
```

`adb` topilmasa, [INSTALL.md](INSTALL.md) — sideload va USB debugging.

### 3. Birinchi marta ishga tushganda

1. Welcome ekranidan til tanlang (UZ/EN).
2. Taxallus kiriting.
3. **Yangi portal yaratish** → 6 xonali ID + kod paydo bo'ladi.
4. Do'stingiz **Mavjud portalga qo'shilish** ni bosib QR ni skanerlasin
   yoki ID + kodni yozsin.

Mobile internet (CGNAT) bo'lsa va ulanishlar barbod bo'lsa: Settings →
Cloudflare TURN qismida Token ID + API Token kiriting (1 TB/oy bepul,
[Cloudflare Calls](https://developers.cloudflare.com/calls/turn/) dan).

---

## Toolchain pinlari (qisqa)

Mismatchlar build'ni buzadi:

| Pin | Qiymat | Joyi |
| --- | --- | --- |
| Gradle daemon JDK | 21 | `gradle/gradle-daemon-jvm.properties` |
| AGP | 9.0.1 (canary) | `gradle/libs.versions.toml` |
| Kotlin | 2.0.21 | `gradle/libs.versions.toml` |
| Compose BOM | 2024.09.00 | `gradle/libs.versions.toml` |
| compileSdk | 36 (minorApiLevel 1) | `app/build.gradle.kts` |
| minSdk | 26 (Android 8.0) | `app/build.gradle.kts` |
| WebRTC | `io.getstream:stream-webrtc-android` 1.3.8 | catalog |

To'liq jadval: [BUILDING.md](BUILDING.md#toolchain).

---

## Hujjatlar / Documentation map

```
mobile/
├── README.md          ← bu fayl
├── INSTALL.md         ← APK ni qurilmaga yuklash
├── BUILDING.md        ← manbadan qurish, troubleshooting
├── ARCHITECTURE.md    ← paket layout, Go reference moslik jadvali
└── CLAUDE.md          ← Claude Code uchun loyiha qoidalari
```

Asosiy loyiha hujjatlari root'da: [`../PROTOCOL.md`](../PROTOCOL.md),
[`../ARCHITECTURE.md`](../ARCHITECTURE.md), [`../NAT-VA-TURN.md`](../NAT-VA-TURN.md),
[`../PRIVACY.md`](../PRIVACY.md), [`../SECURITY.md`](../SECURITY.md).

---

## Hissa qo'shish / Contributing

[`../CONTRIBUTING.md`](../CONTRIBUTING.md) ni o'qing va shu papkadagi
[CLAUDE.md](CLAUDE.md) ni ham. Asosiy qoidalar:

- **Simli protokol o'zgarishi taqiqlanadi** — faqat ixtiyoriy maydon
  qo'shish. Renaming/o'chirish desktop bilan moslikni buzadi.
- **Go reference'ga oyna oydek moslik** — `protocol/`, `crypt/`,
  `signaling/`, `peer/`, `mesh/`, `transfer/` paketlari `../client/`
  ostidagi tegishli paketlar bilan bir xil so'zlashishi kerak.
- **Hujjatlar Uzbekcha** (kod komentariyalari Inglizcha bo'la oladi).
