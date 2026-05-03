# Changelog

Bu hujjat Portal'ning har bir public chiqarilishidagi o'zgarishlarni qayd
etadi. Format [Keep a Changelog](https://keepachangelog.com/en/1.1.0/) ga
asoslangan; loyiha [Semantic Versioning](https://semver.org/spec/v2.0.0.html)
ga rioya qiladi.

## [Unreleased]

Joriy ishlanma. v0.4.0 dan keyingi o'zgarishlar bu yerga yoziladi.

## [0.4.0] — 2026-XX-XX (rejada)

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

[Unreleased]: https://github.com/Yaxyobek0877/portal_traffic/compare/v0.4.0...HEAD
[0.4.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.4.0
[0.3.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.3.0
[0.2.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.2.0
[0.1.0]: https://github.com/Yaxyobek0877/portal_traffic/releases/tag/v0.1.0
