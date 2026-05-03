# Maxfiylik siyosati / Privacy Policy

**Oxirgi yangilanish / Last updated:** 2026-05-01
**Tegishli versiya / Applies to:** Portal v0.4.0 va keyin / and later

---

## 1. Qisqa xulosa / TL;DR

Portal — peer-to-peer mesh tarmoq dasturi. Sizning chat, fayl va
boshqa peer-to-peer trafikingiz **bevosita boshqa peer'larga** boradi va
**hech qanday markaziy serverga saqlanmaydi**. Signal serveri faqat
peer'larni topish uchun ishlatiladi va ulangandan keyin trafikni ko'rmaydi.

Portal is a peer-to-peer mesh networking app. Your chat, file, and other
peer-to-peer traffic flows **directly between peers** and is **never
stored on any central server**. The signaling server is only used to
discover peers and does not see traffic after the connection is
established.

---

## 2. Biz to'playdigan ma'lumotlar / Data we collect

### 2.1 Signaling serveri / Signaling server (`signaling.1pro.uz`)

Portal yaratishda yoki qo'shilishda, ulanish davomida quyidagilar
qisqa muddat (joriy sessiya davomida) RAM'da saqlanadi va sessiya
tugagandan so'ng o'chiriladi:

While creating or joining a portal, the following are kept **in memory
only** for the duration of your session and are deleted when the session
ends:

| Ma'lumot / Data | Maqsad / Purpose | Saqlash / Retention |
| --- | --- | --- |
| IP manzil / IP address | Rate limiting, abuse prevention | Sessiya davomida + 24h log / Session + 24h logs |
| Taxallus / Nickname | Boshqa peer'larga ko'rsatish / Show to other peers | Sessiya davomida / Session only |
| Portal ID + kod / code | Peer'larni topishtirish / Peer matching | Portal yopilgunga qadar / Until portal closes |
| WebRTC handshake (SDP, ICE) | Peer'larni ulashtirish / Connect peers | RAM'dan o'tib ketadi / Passes through RAM |

**Biz quyidagilarni ko'rmaymiz / We never see:**
- Sizning chat xabarlaringizni / Your chat messages
- Uzatilayotgan fayllarni / File transfer contents
- Tunnel qilingan TCP/UDP trafikni (Minecraft, dev servers, hokazo) / Tunneled traffic
- Telefon raqamlari, email yoki haqiqiy ismlar / Phone numbers, emails, real names

### 2.2 TURN serveri / TURN server

Agar siz Symmetric NAT ortida bo'lsangiz, ulanish TURN serveri orqali
o'tadi. Portal sukut bo'yicha **siz tanlagan TURN provayderini**
ishlatadi (Cloudflare yoki o'zingizniki). TURN serveri shifrlangan
baytlarni ko'radi, lekin **shifrini ocha olmaydi** — DTLS + PCP-1 ikki
qatlam himoya qiladi.

If you are behind Symmetric NAT, the connection routes through a TURN
server. Portal uses **the TURN provider you choose** (Cloudflare or your
own). The TURN server sees encrypted bytes but **cannot decrypt them** —
DTLS + PCP-1 protect with two layers.

Cloudflare TURN ishlatilsa, ularning [Privacy Policy](https://www.cloudflare.com/privacy/)
ham qo'llaniladi.
If using Cloudflare TURN, their privacy policy also applies.

### 2.3 Lokal qurilmangizda / On your local device

Portal sizning qurilmangizda (`~/.portal/portal.db` SQLite faylida)
quyidagilarni saqlaydi:

Portal stores the following on your device (`~/.portal/portal.db`):

- Sozlamalar (signal URL, TURN credentials, til) / Settings (signaling URL, TURN creds, language)
- Portal tarixi (oxirgi 50 ta) / Portal history (last 50)
- Kontaktlar (siz qo'shganlar) / Contacts (those you added)
- PCP-1 identity keypair (OS keychain'da shifrlangan) / PCP-1 identity keypair (encrypted in OS keychain)

Bu ma'lumotlar **hech qaerga yuborilmaydi**. Ularni xohlasangiz qo'lda
o'chirib tashlashingiz mumkin: `~/.portal/` papkasini o'chirish bilan.

This data **never leaves your device**. You can delete it anytime by
removing the `~/.portal/` directory.

### 2.4 Crash reporting (ixtiyoriy / opt-in)

Agar sozlamalarda yoqib qo'ysangiz, dastur halokatga uchraganda
quyidagilar yuboriladi:

If you opt in via Settings, the following is sent on a crash:

- Portal versiyasi / Portal version
- OS va versiyasi / OS and version
- Stack trace (so'rovlar va sezgir ma'lumotlarsiz / no PII or query data)

Hech qachon: foydalanuvchi nomi, taxallus, IP, portal ID, chat tarkibi.
Never: username, nickname, IP, portal ID, chat content.

Bu standart bo'yicha **o'chirilgan**. / This is **off by default**.

---

## 3. Server loglari / Server logs

Signal serveri quyidagilarni 24 soat saqlaydi (xato debug qilish va
abuse aniqlash uchun):

The signaling server keeps the following for 24 hours (for debugging and
abuse detection):

- Ulanish vaqti, IP (hash qilingan oxirgi 12 soatdan keyin) / Connection time, IP (hashed after 12 hours)
- Xato kodlari (`PORTAL_NOT_FOUND` va h.k.) / Error codes
- Rate limit hisoblagichlari / Rate limit counters

24 soatdan keyin avtomatik o'chiriladi. /
Automatically deleted after 24 hours.

---

## 4. Sizning huquqlaringiz / Your rights

- **Foydalanmaslik:** Hisob qaydnomasi yo'q, ro'yxatdan o'tish yo'q —
  shunchaki dasturni o'chirib tashlang.
  No account, no signup — just delete the app to stop.
- **Ma'lumotlarni o'chirish:** `~/.portal/` papkasini o'chiring.
  Delete the `~/.portal/` directory.
- **Server tomonida sizga bog'lanadigan persistent identifikator yo'q.**
  No persistent server-side identifier tied to you.

GDPR, CCPA va shunga o'xshash qonunlar bo'yicha biz oddiyga eng
yaqin yondashuvni tutdik: kerak bo'lmagan ma'lumotni umuman to'plamadik.

For GDPR / CCPA and similar laws, we took the simplest approach:
we don't collect data we don't need.

---

## 5. Bog'lanish / Contact

Maxfiylik bo'yicha savollar / Privacy questions:

- GitHub: https://github.com/Yaxyobek0877/portal_traffic/issues
- Email: portal@1pro.uz

---

## 6. O'zgarishlar / Changes

Ushbu siyosat o'zgarganda, "Oxirgi yangilanish" sanasi yangilanadi va
o'zgarish CHANGELOG.md'da qayd etiladi. Mavjud foydalanuvchilarga
ahamiyatli o'zgarishlar haqida ilova ichidan xabar beriladi (yangilash
paytida).

When this policy changes, the "Last updated" date is updated and the
change is recorded in CHANGELOG.md. Existing users are notified of
material changes inside the app (during the update flow).
