# Xavfsizlik siyosati / Security Policy

## Zaiflik haqida xabar berish / Reporting a vulnerability

Agar Portal'da xavfsizlik muammosini topgan bo'lsangiz, **ommaviy
oshkor qilishdan oldin** shaxsan xabar bering.

If you find a security issue in Portal, please **do not disclose
publicly**. Report privately via:

1. **GitHub Security Advisory** (afzal / preferred):
   [Report a vulnerability](https://github.com/Yaxyobek0877/portal_traffic/security/advisories/new)
2. **Email:** `portal-security@1pro.uz` — mavzu qatorida `[security]`
   bilan boshlang

Iltimos, quyidagilarni qo'shing / Please include:

1. Muammo va uning ta'siri tavsifi / Description and impact
2. Qayta ishlash bosqichlari yoki PoC / Reproduction steps or PoC
3. Boshqa joyda allaqachon ulashganmisiz / Have you shared elsewhere

**Vaqt rejimi / Timeline:**
- 72 soat ichida tasdiqlash / 72-hour acknowledgement
- 14 kun ichida yamoq / 14-day patch target after triage
- Reliz notes'da tan olinasiz (agar so'ramasangiz) / Acknowledgement
  in release notes (unless declined)

---

## Qamrov / Scope

**Qamrovda / In scope:**

- Desktop client (`client/`) va Wails-bound Go ↔ JS interfeysi
- **PCP-1 (Portal Cipher Protocol v1)** — qarang [SPEC](client/crypt/pcp/SPEC.md)
- Shared simli protokol (`shared/protocol/`)
- Signal server dizayni (deployed instance — operatorning
  mas'uliyati, lekin dizayn darajasidagi muammolar qamrovda)
- Mobile (Android, beta) — `mobile/`

**Qamrovdan tashqarida / Out of scope:**

- Foydalanuvchining yomonniyatli uchinchi tomon signal serverini
  o'rnatishi talab qiladigan zaifliklar.
- Per-IP rate limit'ni hisobga olmaydigan 6 xonali portal kodiga
  nazariy brute-force hujumlar (qarang [ARCHITECTURE.md](ARCHITECTURE.md)
  va [SECURITY-AUDIT.md](SECURITY-AUDIT.md)). PCP-1 kodlari `XXXX-YYYY`
  format bilan bog'langan; brute-force resurslari bilan ham server-side
  rate limit asosiy himoya.
- Bog'liqliklardagi xatolar (Go stdlib, Wails, pion/webrtc, gorilla/websocket).
  Ularni upstream'ga xabar bering.

---

## Biz nima va'da qilamiz / What we promise

- **Signal serveri ilova trafigini o'qiy olmaydi.** Buni qila olganingizni
  ko'rsatsangiz — kritik. *The signaling server cannot read application
  traffic. If you can show otherwise — critical.*
- **A portalidagi peer Portal orqali B portalidagi peer'ga yeta olmaydi.**
  *Peer in portal A can't reach peer in portal B via Portal — critical
  if you find a way.*
- **`webrtc.*` xabarlardagi `from` maydoni server tomonidan shtamplangan
  va soxtalashtirib bo'lmaydi.** *`from` field on relayed `webrtc.*`
  messages is server-stamped and unforgeable — high if you find a way.*
- **PCP-1 yopiq xabarlar (sealed boxes) anonim sender'ni saqlaydi.**
  *PCP-1 sealed boxes preserve sender anonymity.*

---

## Self-host hardening / Self-host hardening

Agar o'z signal serveringizni boshqarsangiz / If you run your own
signaling server:

- TLS ishlating. Cloudflare Origin Certificate'lari bepul; deploy
  qo'llanmaga qarang.
- Cloudflare ortida (yoki WebSocket qo'llab-quvvatlovchi DDoS-mitigating
  CDN ortida) ishga tushiring.
- Og'ir foydalanish kutilsa, standart bo'lmagan rate limit'larni
  o'rnating; standart qiymatlar konservativ.
- systemd hardening profili bilan root bo'lmagan foydalanuvchi sifatida
  ishga tushiring.
- Origin Certificate'lar muddati tugashidan oldin yangilang.

---

## PCP-1 — qo'shimcha xavfsizlik kontekst / additional security context

PCP-1 standart, ko'p marotaba audit qilingan kriptografik primitivlar
ustida qurilgan (Ed25519, X25519, HKDF-SHA256, XChaCha20-Poly1305).
Spetsifikatsiya: [`client/crypt/pcp/SPEC.md`](client/crypt/pcp/SPEC.md).

Audit uchun ochiq. Topilgan har qanday kamchilikni yuqoridagi
advisory kanali orqali bildirsangiz, biz ko'rib chiqib, kerak bo'lsa
v2 ga o'tkazamiz.

PCP-1 is built on standard, audited cryptographic primitives. Open for
public audit. Report findings via the advisory channel above.
