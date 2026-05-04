# Changelog

Bu hujjat Portal'ning har bir public chiqarilishidagi o'zgarishlarni qayd
etadi. Format [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) ga
asoslangan; loyiha [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
ga rioya qiladi.

## [Unreleased]

v0.4.1 dan keyingi ishlanma bu yerga yoziladi.

## [0.4.1] — 2026-05-04

v0.4.0'dan keyingi katta UX va xavfsizlik ishlari. Asosiy yo'nalishlar:
peer ulanish ishonchliligi, UDP / o'yin server qo'llab-quvvatlashi, LAN
qurilmalarni mesh orqali ulashish (kamera / NVR / printer), va remote
debug uchun avtomatik log yuklash.

### Added — Yangi imkoniyatlar
- **PeerTable + ⚡P2P / ☁️TURN badge** — qimirlovchi mesh diagrammasi
  o'rniga 6 ustunli jadval. Har peer uchun ICE selected pair tipi
  (`host` / `srflx` / `relay`) va manzillari ko'rinadi; LAN bo'lsa
  "bir tarmoq ichida", srflx pair bo'lsa "Internet — turli tarmoq",
  TURN bo'lsa "TURN relay" deb belgilanadi.
- **3 sekundlik bandwidth probe** — har peer kartochkasidagi "Tezligi"
  tugmasi 32 KB binary chunk'larini control channel'da yuborib qabul
  qiluvchi tomondan o'lchangan Mbps ni qaytaradi.
- **UDP servislar** — listeners.go endi `lsof -iUDP` / `ss -lnup` ham
  yuradi, GUI'dagi expose va dial path UDP'ni qabul qiladi. CS2,
  Valorant, Minecraft Bedrock kabi UDP o'yin serverlari mesh orqali
  ulashish mumkin. Single port'ga TCP+UDP "Ikkalasi" rejimi ham bor
  (Steam dedicated server'lar uchun).
- **LAN qurilmalarni forward qilish** — Expose'da yangi "+ LAN qurilma"
  maydoni: `192.168.1.100:554` kabi target kiritsangiz, mehmon
  ulanganda traffic shu qurilmaga (kamera / NVR / printer) yo'naltiriladi.
- **TCP-probe LAN scanner** — bir bosishda local /24'ni 11 ta well-known
  portga skanerlaydi (~10s), topilgan kameralar/printerlarni one-click
  expose. "Hammasini och" — barcha topilganlarni avtomatik ekspoz
  qiladi (xavfli portlarni avtomatik o'tkazib yuboradi).
- **Servis health check** — har 30 sekundda exposed target'ni TCP
  probe qiladi; UI'da yashil/sariq/qizil dot ko'rinadi (kamera
  uzilsa darhol bilib turasiz).
- **Risk warning** — DB / router admin / RDP kabi xavfli portlarni
  expose qilishda confirm dialog (false positives bilan, "danger"
  va "warn" darajalari).
- **Activity log** — Settings → Faollik: kim qachon qaysi servisga
  ulandi, success/error bilan. 200 ta yozuvli ring buffer.
- **Service presets** — ✨ Tezkor tugmasi: Minecraft (Java/Bedrock),
  CS2, Rust, Factorio, Terraria, RTSP kamera, Web UI, SSH, Vite —
  bir bosishda nom + port + protokol to'ldiriladi.
- **Servis persistence** — exposed servislar SQLite'da saqlanadi va
  har portal yaratish/qo'shilishda avtomatik qayta ekspoz qilinadi.
  ⏸ pause toggle: vaqtincha to'xtatish (LAN target IP'ni yo'qotmasdan).
- **Server-issued TURN credentials** — signaling server clientga
  qisqa muddatli TURN creds yuborib bersa, foydalanuvchi qo'lda
  Cloudflare account ochmaydi (deploy qilingach ishlaydi).
- **Avtomatik log upload** — har 30 sekundda mahalliy log fayli
  yangi qatorlari `https://<signaling>/logs/upload`'ga yuklanadi.
  Default-on, opt-out toggle Settings'da. Anonim 32-hex client ID.
  Server-side handler `docs/SERVER_LOG_UPLOAD.md` da.
- **Live LAN scan progress** — 12s skanerlash davomida bar va
  "192.168.1.123… · 754/2794 · 1 topildi" qatori ko'rsatiladi.

### Changed — O'zgartirilgan
- ServicesPanel butun qayta dizayn: TCP/UDP/Ikkalasi toggle, LAN target
  collapsible advanced bo'limi, dialed pill "lokal" tag bilan.
- Local proxy listener endi remote port bilan teng port'ga bog'lanishni
  afzal ko'radi (`127.0.0.1:5000` ↔ `peer:5000` mos keladi); band
  bo'lsa OS-pick'ga tushadi va UI sariq "lokal*" badge ko'rsatadi.
- Signaling event'larning hammasi INFO darajada loglanadi
  (`recv portal.joined peer_count=1`, `send portal.create nick=…`).
- macOS system processes (ControlCenter, mDNSResponder, sharingd va
  hk.) "Lokalda topilgan portlar"da ko'rsatilmaydi — tasodifan tizim
  servisini ulashmaslik uchun.

### Fixed — Tuzatildi
- **Critical: deadlock in onPeerJoinedWithRoster** — `m.mu.Lock()`
  ushlab turib `iceServersForPeer()` chaqirilardi (ichida RLock).
  Go'ning sync.RWMutex recursive lock'ni qo'llab-quvvatlamaydi →
  goroutine mangu bloklangan, peer kartochkasi UI'da paydo bo'lmagan.
  `iceServersForPeerLocked` helper'i bilan tuzatildi.
- **Critical: portal:ready event payload shape** — `*ev.Portal` (raw
  `mesh.PortalInfo`) yuborilardi; Go field nomlari bosh harf bilan
  marshal qilingan, frontend `portalId`/`code` undefined ko'rgan,
  Windows'da ID/KOD blank bo'lib qolardi. `portalToView()` orqali
  o'tkazildi.
- "Tarmoqdagi qurilmalar" ro'yxatida bir port bir necha marta
  ko'rinardi; (port, protocol) bo'yicha dedup.

### Documentation
- `docs/SERVER_TURN.md` — signaling serverga Cloudflare TURN integration
  Go snippet.
- `docs/SERVER_LOG_UPLOAD.md` — log upload endpoint Go handler.
- README'ga "Game servers and dev URLs" + "Where the traffic flows"
  bo'limlari qo'shildi.

## [0.4.0] — 2026-05-03

Birinchi rasmiy ommaviy chiqarilish.

### Added — Yangi imkoniyatlar
- **MIT litsenziya** — kod ochiq, fork va commercial foydalanish ruxsat etilgan.
- **PCP-1 (Portal Cipher Protocol v1)** — yangi shifrlash qatlami: Ed25519
  identity + X25519 ephemeral session + XChaCha20-Poly1305 AEAD. Forward
  secrecy va identity-bound portal kodlari.
- **Internationalization (i18n)** — English + O'zbek UI; til Settings'dan
  tanlanadi.
- **CI/CD pipeline** — GitHub Actions: har push'da test, har tag'da
  cross-platform binarlar Releases'ga yuklanadi.
- **CHANGELOG va versiyalash** — endi har relizning aniq qaydi bor.

### Changed — O'zgartirilgan
- README endi screenshot va demo GIF bilan keladi.
- Web landing (portal.1pro.uz) qayta dizayn qilindi: feature kartalari,
  yuklab olish tugmalari har platforma uchun.

### Security — Xavfsizlik
- PBKDF2-SHA256 + NaCl secretbox o'rniga PCP-1 (yuqorida).
- Brute-force kuzatuvi: per-portal failed-join hisoblagichi (sozlashda).

### Documentation
- ENGLISH README qo'shildi (README.md asosiy, README-uz.md o'zbekcha).
- PCP-1 spetsifikatsiyasi: `client/crypt/pcp/SPEC.md`.
- PRIVACY.md va TERMS.md qo'shildi.
- Issue/PR template'lari `.github/` ga qo'shildi.

### Known limitations
- Mobile (Android) — eksperimental, bu relizga kirmaydi. Kelajakda alohida
  reliz.
- Code signing yo'q: macOS va Windows binarlari rasmiy imzolanmagan;
  foydalanuvchi qo'lda ruxsat berishi kerak. Keyingi reliz'da imzolanadi.

## [0.3.0] — 2026-04-30

Repodagi oxirgi commit holatigacha — desktop UI va proxy tugagan ichki
versiya. Public reliz emas.

### Added
- Wails + React desktop UI (welcome ekrani, mesh diagrammasi, chat,
  servislar paneli, Settings).
- TCP va UDP proxy (`portal expose`, `portal dial`).
- App-layer NaCl secretbox shifrlash (PBKDF2 portal kodidan).
- SQLite persistence (`~/.portal/portal.db`).
- NAT detection + Cloudflare TURN integratsiyasi.
- Drag-and-drop fayl uzatish.

### Fixed
- Heartbeat outstanding map data race.
- 3+ peer mesh glare (joiner-always-offerer qoidasi).
- Channel close race teardown'da.

Batafsil: [SECURITY-AUDIT.md](SECURITY-AUDIT.md).

## [0.2.0] — 2026-04-30 (ichki)

Client mesh dvigateli — Wails'siz CLI test harness.

## [0.1.0] — 2026-04-30 (ichki)

Signal serveri (xususiy repo'da) ishga tushdi: `signaling.1pro.uz/ws`.

[Unreleased]: https://github.com/Yaxyobek0877/portal_traffic/compare/v0.4.1...HEAD
[0.4.1]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.4.1
[0.4.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.4.0
[0.3.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.3.0
[0.2.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.2.0
[0.1.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.1.0
