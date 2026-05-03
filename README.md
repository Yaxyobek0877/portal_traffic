<div align="center">

# Portal

**To'g'ridan-to'g'ri ulanish. Orada hech qanday server yo'q.**
**Direct connections. Zero servers between you.**

[![CI](https://github.com/Yaxyobek0877/portal_traffic/actions/workflows/ci.yml/badge.svg)](https://github.com/Yaxyobek0877/portal_traffic/actions/workflows/ci.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)
[![Go 1.24+](https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go)](https://go.dev/)
[![Latest release](https://img.shields.io/github/v/release/Yaxyobek0877/portal_traffic?include_prereleases&sort=semver)](https://github.com/Yaxyobek0877/portal_traffic/releases)
[![Downloads](https://img.shields.io/github/downloads/Yaxyobek0877/portal_traffic/total)](https://github.com/Yaxyobek0877/portal_traffic/releases)

[**Sayt → portal.1pro.uz**](https://portal.1pro.uz) ·
[Yuklab olish / Download](https://github.com/Yaxyobek0877/portal_traffic/releases) ·
[Roadmap](ROADMAP.md) ·
[PCP-1 Spec](client/crypt/pcp/SPEC.md)

Portal — odamlar o'rtasida xususiy peer-to-peer (tengma-teng) mesh
tarmoq quradigan dastur. Portal oching, kodni ulashing — har bir
qurilma boshqa har biri bilan to'g'ridan-to'g'ri ulanadi: chat, fayl
uzatish, o'yin serveri, dev URL ulashish va boshqa narsalar uchun.

Portal builds a private peer-to-peer mesh network between people.
Open a portal, share the code — every device connects directly to
every other: chat, file transfer, game servers, dev URL sharing,
and more.

> Signal serveri faqat dastlabki handshake uchun. Undan keyin hamma
> trafik to'g'ridan-to'g'ri oqadi va **WebRTC DTLS + PCP-1** ikki
> qatlam bilan shifrlanadi.
>
> The signaling server is only used for the initial handshake. After
> that, all traffic flows directly and is encrypted with two layers:
> **WebRTC DTLS + PCP-1**.

</div>

---

## 📸 Skrinshotlar / Screenshots

<!-- TODO: docs/screenshots/ ga rasmlar qo'shilgach yangilang.
     Replace with real screenshots after adding to docs/screenshots/.
     Tavsiya etilgan rasmlar:
     - welcome.png — welcome ekrani (logo, taxallus inputi)
     - portal-mesh.png — jonli mesh diagrammasi va peer ro'yxati
     - chat-services.png — chat paneli + servislar (Minecraft expose)
-->

| Welcome | Mesh + Chat | Servislar / Services |
| :---: | :---: | :---: |
| _coming soon_ | _coming soon_ | _coming soon_ |

---

## 🚀 Bu loyiha bilan nima qila olasiz / What you can do

| Imkoniyat / Feature | Tafsilot / Detail |
| --- | --- |
| **2–16 ta qurilmani bog'lash / Connect 2–16 devices** | Portal oching, ID + kodni ulashing. Internetning istalgan joyidan kim xohlasa qo'shila oladi — router sozlamasi yo'q, VPN yo'q, port forwarding yo'q. |
| **Chat va fayl uzatish / Chat and file transfer** | Guruh chat, drag-and-drop fayllar, progress bar. PCP-1 bilan uchidan-uchiga shifrlangan. |
| **TCP/UDP tunnel** | Minecraft serverni `localhost:25565`'da ishga tushirasiz, do'stlaringiz `10.42.0.3:25565`'ga ulanishadi — Portal baytlarni mesh orqali uzatadi. SSH, dev server'lar, har qanday TCP/UDP. |
| **Jonli mesh vizualizatsiyasi / Live mesh viz** | Kim kim bilan ulangani, RTT (ping), NAT-traversal indikatorlari. |
| **Maxfiylik / Privacy** | Handshake'dan keyin signal serveri trafikni hech qachon ko'rmaydi. WebRTC DTLS + PCP-1. |

---

## 📥 O'rnatish / Installation

### Tayyor installerlar / Pre-built installers

[**Releases sahifasidan**](https://github.com/Yaxyobek0877/portal_traffic/releases)
o'z OS'ingiz uchun yuklab oling:

| Platform | Fayl |
| --- | --- |
| macOS (Apple Silicon) | `Portal-vX.Y.Z-darwin-arm64.zip` |
| macOS (Intel) | `Portal-vX.Y.Z-darwin-amd64.zip` |
| Windows 10/11 (64-bit) | `Portal-vX.Y.Z-windows-amd64.zip` |
| Linux (Ubuntu/Debian) | `Portal-vX.Y.Z-linux-amd64.tar.gz` |

> ⚠️ **v0.4.0:** binarlar imzolanmagan. macOS'da right-click → Open;
> Windows'da SmartScreen → "Run anyway". Sertifikatlar [keyingi
> reliz'da](docs/CODE_SIGNING.md).

> ⚠️ **v0.4.0:** binaries are unsigned. macOS: right-click → Open;
> Windows: SmartScreen → "Run anyway". Code signing in [next
> release](docs/CODE_SIGNING.md).

### Manbadan build / Build from source

[INSTALL.md](INSTALL.md) — har bir platforma uchun batafsil qadamlar.

[INSTALL.md](INSTALL.md) — detailed steps for each platform.

### Mobile (Android) — beta

Android client to'liq mavjud va ishlaydi, lekin v0.4.0 reliz tarkibiga
hali kirmaydi. Manbadan build qilish: [mobile/README.md](mobile/README.md).

Android client is fully functional but not bundled in the v0.4.0
release. Build from source: [mobile/README.md](mobile/README.md).

---

## 🎬 Sessiya qanday ko'rinadi / How a session looks

```
1. Dasturni ochasiz, taxallus yozasiz.
2. "Portal yaratish" ni bosasiz.
3. Portal sizga ko'rsatadi:
       ID:    428591
       KOD:   X3K9-A2F1
   (yoki QR kodni skanerlatib oling)
4. Do'stingiz dasturni ochadi, "Portalga qo'shilish" → ID + kod.
5. Mesh hosil bo'ldi: chat, fayl uzatish, port ulashish.
```

NAT/CGNAT bo'yicha 30s ichida `Failed` chiqsa: [NAT-VA-TURN.md](NAT-VA-TURN.md)
bo'yicha TURN sozlang.

If you see `Failed` after 30s due to symmetric NAT/CGNAT: configure
TURN per [NAT-VA-TURN.md](NAT-VA-TURN.md).

---

## 🛰 Trafik qayerdan oqadi / Where the traffic flows

Handshake bitgandan keyin Portal client'i ICE algoritmi tanlagan
yo'lni ishlatadi. Ikki rejim bor — Portal har peer ustida **kichik
belgi** ko'rsatadi:

After the handshake completes, the Portal client uses whichever path
ICE selects. There are two regimes — Portal shows a small **badge**
on each peer card:

| Belgi / Badge | Holat / Mode | Trafik qayerdan o'tadi / Path |
| :---: | --- | --- |
| ⚡ **P2P** | host / srflx pair | Ikki qurilma o'rtasida to'g'ridan-to'g'ri. Hech qanday server yo'q. *Direct between the two devices. No server in the middle.* |
| ☁️ **TURN** | relay pair | TURN serveri orqali relay qilinadi (DTLS+PCP-1 shifrlanganligicha qoladi). *Relayed via a TURN server (still encrypted end-to-end).* |

Aksariyat uy interneti `P2P`'da ulanadi. Symmetric NAT, ko'p mobil
operator (4G/5G), CGNAT yoki qattiq korporativ tarmoq ortida bo'lsangiz
— `TURN` kerak bo'ladi. TURN sozlanmagan bo'lsa, ulanish 30s'da
`Failed` bo'ladi. Settings → Network bo'limidan Cloudflare TURN yoki
o'zingizning TURN URL'ingizni kiritsangiz `TURN` rejim ishga tushadi.
Batafsil: [NAT-VA-TURN.md](NAT-VA-TURN.md).

Most home networks land on `P2P`. Symmetric NAT, mobile carriers
(4G/5G), CGNAT, or strict corporate networks fall back to `TURN`.
If TURN isn't configured, the connection fails after ~30s. Add
Cloudflare TURN or a custom TURN URL in Settings → Network to enable
the `TURN` mode. Details: [NAT-VA-TURN.md](NAT-VA-TURN.md).

---

## 🔒 Xavfsizlik / Security

Portal ikki qatlam shifrlash ishlatadi:

Portal uses two layers of encryption:

1. **WebRTC DTLS-SRTP** — har data kanal uchun standart, SDP handshake
   davomida muzokara qilinadi.
2. **[PCP-1 (Portal Cipher Protocol v1)](client/crypt/pcp/SPEC.md)** —
   Ed25519 identity + X25519 ephemeral session + XChaCha20-Poly1305
   AEAD. Per-pair forward secrecy, identity-bound portal codes.

Xavfsizlik audit natijalari: [SECURITY-AUDIT.md](SECURITY-AUDIT.md).
Zaiflik topdingizmi? [Xususiy security advisory](https://github.com/Yaxyobek0877/portal_traffic/security/advisories/new)
oching, ommaviy issue emas.

Audit results: [SECURITY-AUDIT.md](SECURITY-AUDIT.md). Found a
vulnerability? Open a [private security advisory](https://github.com/Yaxyobek0877/portal_traffic/security/advisories/new),
not a public issue.

---

## 📚 Hujjatlar / Documentation

| Hujjat / Doc | Nima haqida / What |
| --- | --- |
| [ROADMAP.md](ROADMAP.md) | 1–5 bosqichlar va joriy holat / Phases 1–5 and status |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Mesh, NAT, shifrlash, tahdid modeli / Topology, NAT, encryption, threat model |
| [PROTOCOL.md](PROTOCOL.md) | Signal va P2P simli protokol / Signal and P2P wire protocol |
| [INSTALL.md](INSTALL.md) | Manbadan build / Build from source |
| [NAT-VA-TURN.md](NAT-VA-TURN.md) | NAT, ICE, TURN — sodda tushuntirish |
| [PCP-1 SPEC](client/crypt/pcp/SPEC.md) | Portal Cipher Protocol v1 spec |
| [CHANGELOG.md](CHANGELOG.md) | Versiyalar / Versions |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Hissa qo'shish / Contributing |
| [SECURITY.md](SECURITY.md) | Xavfsizlik siyosati / Security policy |
| [SECURITY-AUDIT.md](SECURITY-AUDIT.md) | Audit natijalari / Audit results |
| [PRIVACY.md](PRIVACY.md) | Maxfiylik siyosati / Privacy |
| [TERMS.md](TERMS.md) | Foydalanish shartlari / Terms |

---

## 🏗 Repository tuzilishi / Repository layout

```
portal_traffic/
├── shared/        Simli protokol — har client uchun manba / Wire protocol — source for all clients
├── client/        Wails + React desktop dastur / Desktop app
│   └── crypt/pcp/ Portal Cipher Protocol v1 (PCP-1)
├── mobile/        Android (Kotlin) — beta, manbadan build / beta, build from source
├── web/           Landing sayti (portal.1pro.uz) / Landing site
└── server/        Signal serveri (xususiy — VPS, repodan tashqarida) / Private — runs on our VPS
```

Signal serveri kodi ataylab repoga kiritilmagan. Portal client istalgan
signal URL'ga ulanadi — server deyarli almashtirib qo'yiladigan
komponent. Standart endpoint: **`wss://signaling.1pro.uz/ws`**.
O'zingizniki ishga tushirish — [PROTOCOL.md](PROTOCOL.md) +
[`shared/protocol/messages.go`](shared/protocol/messages.go) bo'yicha
bir necha yuz qator Go.

The signaling server source is intentionally not in this repo. The
Portal client connects to whatever URL you configure, so the server
is essentially swappable. Default endpoint: **`wss://signaling.1pro.uz/ws`**.
Run your own — wire protocol fully documented.

---

## ⚖️ Litsenziya / License

Portal **MIT litsenziyasi** ostida — [LICENSE](LICENSE). Kodni istalgan
maqsadda (shu jumladan kommersiya) ishlatishingiz, o'zgartirib
tarqatishingiz mumkin; faqat asl litsenziya matni saqlanishi kerak.

Portal is **MIT-licensed** — see [LICENSE](LICENSE). Use it for any
purpose (including commercially), modify, redistribute. Just keep the
license text.

"Portal" nomi va brendi (logo, app ikonkasi, `portal.1pro.uz` domeni)
MIT ostida emas. Agar fork qilsangiz, boshqa nom tanlang.

The "Portal" name and brand (logo, icon, `portal.1pro.uz` domain) are
separate. If you fork, please use a different name.

[PCP-1](client/crypt/pcp/SPEC.md) ham MIT ostida — boshqa loyihalarda
ham ishlatish mumkin.
