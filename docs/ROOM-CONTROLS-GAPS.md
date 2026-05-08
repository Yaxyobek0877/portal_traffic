# Xona boshqaruvi — server tomonida talab qilinadigan ishlar

Bu hujjat foydalanuvchi so'ragan **xona sozlamalari** ro'yxatidagi qaysi narsalar
client'da to'g'ridan-to'g'ri qo'shilganini va qaysilari **signaling server**
(`signaling.1pro.uz`) tomonida ham o'zgarish talab qilishini ko'rsatadi.

## Hozirda client'da to'liq amalga oshirilgan

### Per-port "tasdiqlab yoqish" gate
- `exposed_services.require_approval` kolonkasi
- `proxy.Forwarder.SetApprover` interfeysi: `onOpenTCP/UDP` peer dial
  qilganda gated bo'lsa `App.approvePeerOpen` ga murojaat qiladi
- `service:approval-request` event'i UI'ga modal chiqaradi; foydalanuvchi
  Allow/Deny bossa `app.ApproveServiceRequest` / `DenyServiceRequest` qaytadi
- Decision per `(peer, port, protocol)` cache qilinadi (sticky session davomida)
- 5s TCP / 3s UDP timeout — javob bo'lmasa avtomatik rad etiladi
- ServicesPanel'da har servis qatorida shield ikoni — bossangiz toggle bo'ladi

### Localize qilingan xatolar
- `NICKNAME_TAKEN` → "Bu taxallus portalda allaqachon ishlatilmoqda" / "That nickname is already in use"
- Boshqa server xatolari avval ham localize qilingan: `no such portal`, `portal_full`,
  `portal_locked`, `code does not match`

## Server tomonida ish kerak

Bu uchta narsa **client tomondan yagona o'zgartirib bo'lmaydi** — signaling
server protokolida ko'rsatish kerak.

### 1. Strong 8-char password (current: 6-digit numeric code)

Hozir signal server `portal.create` javobida **6 raqamli kod** beradi.
Foydalanuvchi 8 belgili kuchli parol so'radi (raqam + harf + maxsus belgi).

**Server o'zgarish**:
- `portal.create` handler 8-char alphanumeric password generate qilsin
- `portal.join` handler 6-digit ham, 8-char ham qabul qilsin (o'tish davri uchun)
- Code validation regex'i yangilansin

**Client tomonida**:
- `Welcome.tsx`'dagi 6-digit input cap'i (`maxLength={6}`, `replace(/[^0-9]/g, "")`)
  8 ga ko'tarilsin va alphanumeric'ga ruxsat bersin
- `parseInvite` deeplink parser ham yangi format'ni qo'llasin

### 2. Code regenerate (egasi xona davomida parolni yangilashi)

Hozir code portal yaratilganda fix bo'ladi va o'zgarmaydi. Foydalanuvchi
egasi xohlaganda yangilash mumkin bo'lsin dedi.

**Server o'zgarish**:
- Yangi `portal.regenerate_code` xabari (faqat egasi yubora oladi)
- Server yangi code generate qilsin, eski code'ni invalidate qilsin
- Hamma faol peerlarga `portal.code_changed` event yuborsin

**Client tomonida**:
- PortalHeader'da "Yangi kod" tugmasi (faqat `isOwner` bo'lsa ko'rinadi)
- `app.RegeneratePortalCode()` Wails metodi mesh ga `portal.regenerate_code`
  yuborsin
- `portal.code_changed` event'iga subscribe — UI yangi code'ni ko'rsatsin

### 3. "Ask admin" join mode (egasi har qo'shilishni tasdiqlasin)

Hozir kod to'g'ri bo'lsa server avtomatik admit qiladi. Foydalanuvchi
xona uchun "**every join asks admin**" rejimini xohladi.

**Server o'zgarish**:
- Portal'da yangi flag: `approval_required` (egasi `portal.create_options`
  bilan o'rnatadi yoki `portal.update_options` bilan o'zgartiradi)
- Code to'g'ri kelsa, peer **pending** holatda turadi (mesh'ga qo'shilmaydi)
- Egasiga `portal.join_request` event — peer_id, nickname bilan
- Egasi `portal.allow_join` yoki `portal.deny_join` yuborsin
- Allow → peer to'liq admit; Deny → peer'ga `portal.access_denied` xabar

**Client tomonida**:
- PortalHeader'da xona sozlamalari modal'i — checkbox "Har qo'shilishni
  tasdiqlash"
- `app.SetPortalApprovalMode(bool)` Wails metodi
- `portal.join_request` event'iga subscribe — egasiga modal (xuddi
  `ApprovalQueue` kabi)
- Pending peerlar UI'da "Ulanish kutilmoqda" indicator

### 4. Owner-offline portal survival (eng katta server gap)

**Foydalanuvchi shikoyati**: "room ni egasi offline bo'lsa qolgan qurilmalar
room ga kira olmayabdi". Hozir signal server **owner disconnect** bo'lishi
bilan portal'ni o'chiradi:

- Owner Wi-Fi'dan uziladi yoki dasturni yopadi → server portal'ni `closed`
  qiladi
- Joiner peer'lar `EventPortalClosed` oladi va xona yopiladi
- Owner qaytib kelganda **yangi portal_id** beriladi (eski o'lgan)
- Friends'da saqlangan eski code endi ishlamaydi

Client-side `ResumeActiveSessions` ishlayapti (logda
`send portal.create → recv portal.created portal_id=393789` ko'rinadi),
lekin yangi portal_id bilan. Friends'lar yangi ID + code'ni qaytadan
olishlari kerak.

**Server o'zgarish**:

Eng minimal yechim — **grace period**:
- Owner disconnect bo'lganda portal'ni darhol o'chirmasdan, 5-10 minut
  saqlab turish
- Bu vaqt ichida owner qaytib kelsa, **bir xil portal_id** bilan qayta
  ulansin
- Joiner peer'lar `signaling.reconnecting` event olishadi (yopilmaydi,
  faqat "owner ulanmoqda" indikatori)
- Grace period o'tsa, portal yopiladi (hozirgidek)

Yanada to'liq yechim — **ownership transfer**:
- Owner uzoq vaqtga uzilsa, ikkinchi peer (sort: birinchi qo'shilgan)
  egasiga aylanadi
- Original owner qaytsa, oddiy joiner sifatida ulanadi
- "Co-host" / "moderator" tushunchasi ham mumkin

Server kodida (loyihangizdagi `cmd/signaling/main.go` yoki tegishli
fayl) `Portal.OnOwnerDisconnect` handler'ni:
```go
// Hozir:
func (p *Portal) OnOwnerDisconnect() { p.Close() }

// Kerak:
func (p *Portal) OnOwnerDisconnect() {
    p.state = "owner_offline"
    p.gracePeriod = time.AfterFunc(5*time.Minute, p.Close)
    p.broadcast(EventOwnerOffline)
}

func (p *Portal) OnOwnerReconnect(peerID string) error {
    if p.gracePeriod != nil {
        p.gracePeriod.Stop()
        p.gracePeriod = nil
    }
    p.state = "active"
    p.broadcast(EventOwnerOnline)
    // … re-attach owner peerID
}
```

Client tomondan qo'shimcha ish kerak emas — `EventPortalClosed`
o'rniga `EventOwnerOffline` / `EventOwnerOnline` event'lari kelsa,
UI'da "egasi qayta ulanmoqda" yorlig'i ko'rsatish foydali bo'ladi.

### 5. Multi-device login (bir akkaunt — bir nechta qurilma)

Hozir har Portal install **mahalliy** vault (username + parol) ushlaydi.
Foydalanuvchi telefon + laptop + ishxonadan **bir xil akkaunt** bilan
kirsa, har biriga alohida ro'yxatdan o'tish kerak. Foydalanuvchi haqiqiy
"akkaunt cloud'da" tajribasini xohladi.

**Server o'zgarish**:
- HTTP `/api/auth/signup` — username + parol cloud'ga (server-server.go
  da prototip mavjud, lekin signaling.1pro.uz binarisida deploy bo'lmagan
  — qarang `Server auth deploy gap` memorisi)
- HTTP `/api/auth/signin` — JWT yoki session cookie qaytaradi
- Client har dial paytida session token'ni signaling websocket'ga
  qo'shadi → server username'ni tasdiqlaydi
- `nickname_taken` xatosi shu yerda chiqadi (bir akkaunt + bir
  device-tag bo'lsa qaytaradi)

**Hozirgi vaqt yechimi (client-only)**: Device name (`Settings → Profil
→ Qurilma nomi`) — har install o'z labelini berib turadi (mac / win 64
/ uy / ish). Mesh nicknamesi `texuz · uy` ko'rinishida announce qilinadi
→ peer'lar har installni alohida ko'radi va `nickname_taken` muammoga
duch kelmaydi (chunki har device-tag boshqa-boshqa nickname).

Server cloud-auth deploy bo'lmaguncha, har install hali ham mahalliy
vault ishlatadi — sinxron qilish qo'lda (parolni boshqa kompyuterda
ham yarating, **lekin device name boshqacha bo'lsin**).

## Status

| Feature | Client | Server | Status |
|---------|:------:|:------:|--------|
| Per-port approval gate | ✅ | — | **DONE** |
| NICKNAME_TAKEN error | ✅ | — | **DONE** (server xato yuborsa) |
| Device name (multi-device disambiguation) | ✅ | — | **DONE** (v0.5.1) |
| Auto-run on system startup | ✅ | — | **DONE** (v0.5.1) |
| App icon (mac + win + linux) | ✅ | — | **DONE** (v0.5.1) |
| 8-char password | ❌ | ❌ | server kerak |
| Code regenerate | ❌ | ❌ | server kerak |
| Join approval mode | ❌ | ❌ | server kerak |
| Cloud-sync multi-device login | ❌ | ❌ | server kerak (HTTP /api/auth) |

Kerak bo'lsa server `cmd/signaling/main.go` (loyihangizdagi server kodi)ga
yuqoridagi protokol qo'shimchalarini yozish kerak. Client tomon shu
xabarlarga subscribe qilib avtomatik ishlay boshlaydi.

## Web admin paneli (online qurilma boshqaruvi) — roadmap

Foydalanuvchi: "tola qonli sayt — roomarlni hamda qurilmalarni
portlarini online bolsa boshqarish mumkun bolsin". Ya'ni
`portal.1pro.uz/admin` ichida brauzerdan turib o'z faol portallari,
peer'lari va expose qilingan portlarini ko'rib turish, hatto pause /
remove qilish imkoniyati. Bu **butunlay server-side ish**, chunki:

1. Brauzer client desktop process bilan bevosita gaplasha olmaydi
   (NAT orqasida, protokol mos kelmaydi).
2. Hozirgi signaling server faqat WebSocket session lifecycle bilan
   ishlaydi — qaysi qurilmada qaysi servis yoqilgani haqida hech
   nimani saqlamaydi (privacy bo'yicha to'g'ri qaror, lekin admin
   panel uchun aggregator kerak).

### Talab qilinadigan server qo'shimchalari

#### Auth + per-user portal index
- `/api/auth/login` (allaqachon `server/auth_handlers.go` da bor, lekin
  prod'ga deploy qilinmagan — `memory_server_deployment_gap.md` ga qarang).
  Hozircha cookie sessiyasi etarli; oxir-oqibat OAuth qo'shish kerak.
- HTTP session cookie'lari TLS-only, SameSite=Strict.

#### `/api/portals` (GET) — foydalanuvchi faol portallari
Ro'yxat formati:
```json
[
  {
    "portal_id": "abc123",
    "label": "Uy NVR",
    "code_redacted": "***",
    "is_owner": true,
    "created_at": "2026-05-08T10:23:00Z",
    "peer_count": 2,
    "peers": [
      { "peer_id": "p1", "nickname": "texuz.mac", "rtt_ms": 23, "online": true },
      { "peer_id": "p2", "nickname": "guest.win", "rtt_ms": 89, "online": false }
    ]
  }
]
```
**Manba**: signaling server allaqachon portallarni va peer ulanishlarini
bilim sifatida saqlaydi (sessiya ichida). Ushbu state'ni HTTP'da
read-only ko'rinishda chiqarish — yangi handler.

#### `/api/portals/:id/services` (GET) — expose qilingan servislar
Hozir client *o'zi* expose qilgan servislar haqida announce yubormaydi
(privacy). Admin panel uchun ikki variant bor:

**A.** Client opt-in: foydalanuvchi `Settings → Network → "Adminda
ko'rinsin"` toggle yoqsa, client portal yaratganda servislar ro'yxatini
mesh control channel'idan tashqari signaling server'ga ham yuboradi.
Server uni in-memory store qiladi (relizga yozmaydi). Toggle o'chsa
server yozuvni o'chiradi.

**B.** Server WebRTC stats orqali surrogate sniffing — **rad etilgan**:
texnik jihatdan murakkab, foydalanuvchi privacy'siga ham zid.

A-yo'l afzal: foydalanuvchi nazoratida, opt-in.

#### `/api/portals/:id/services/:port` (PATCH) — pause / unpause / disable
Body: `{ "paused": true }` yoki `{ "deleted": true }`. Server o'z
state'ini yangilaydi va clientga `service.pause_request` push event'i
yuboradi. Client mesh forwarder'ni stop qiladi va `exposed_services`
SQLite jadvalini yangilaydi.

**Muhim**: bu ham opt-in toggle bilan bog'lanishi kerak — agar
foydalanuvchi "Adminda boshqarish ruxsati" toggle'ini o'chirsa, server
faqat **read** qila oladi, push event yubora olmaydi.

### Client tomon — minimal ish

Client opt-in toggle'i kerak (ikki shtat: ko'rsatish vs. boshqarish):
- `Settings → Profile → "Web admin paneli"`:
  - `[ ] Adminda portallarim ko'rinsin (read-only)`
  - `[ ] Admin paneldan boshqarish ruxsat (pause/unpause)`

Va ikki yangi WebSocket xabari signaling protokolida:
- Client → Server: `service.announce` (ekspoz qilinganda)
- Server → Client: `service.pause_request` (admin panel buyrug'i)

### Hozirgi statik admin stub

`web/admin/login.html` va `web/admin/dashboard.html` allaqachon yotibdi —
lekin bular **statik mockup**. Real backend'ga ulanish uchun yuqoridagi
endpointlar deploy qilinishi kerak.

### Status

| Komponent | Status |
|-----------|--------|
| Static admin login + dashboard mockup | ✅ DONE (`web/admin/`) |
| `/api/auth/*` cookie sessions | ⚠️ kod bor, prod'ga deploy yo'q |
| `/api/portals` (GET) | ❌ server kerak |
| `/api/portals/:id/services` (GET) | ❌ server + client opt-in kerak |
| `/api/portals/:id/services/:port` (PATCH) | ❌ server + client opt-in kerak |
| Client `service.announce` / `service.pause_request` xabarlari | ❌ kerak |

## Auto-update (v0.5.4 da yopildi)

Foydalanuvchi: "auto update ham kerag menimcha". v0.5.4 da to'liq
qo'shildi:

- **Backend**: `client/updater/install.go` — `PrepareInstall` GitHub
  Releases'dan asset yuklab oladi, `~/Library/Application Support/Portal`
  o'rniga `os.TempDir()/portal-update-<ns>` ga staging qiladi, OS-specific
  swap script (mac/linux: `/bin/bash`, windows: `.bat`) yozadi.
- **Detach**: `setDetached()` per-OS — Unix'da `Setsid: true` (process
  group'idan ajratish), Windows'da `cmd /c start /min` orqali.
- **Frontend**: `App.tsx` — Update banner'ga ikkita tugma: **"Yangilash"**
  (in-app installer kuchaytiradi) va **"↗"** (release sahifasini ochadi).
- **Apply**: `pending.Apply()` script'ni spawn qiladi, keyin 500ms
  kutib `runtime.Quit(ctx)` chaqiradi. Script process group'imizni
  o'chgach `rm -rf old; mv new old; relaunch` qiladi.

Sinov: `Settings → About → "Yangilashlarni tekshirish"` tugmasi
`CheckForUpdate(refresh=true)` chaqiradi → 24h cache aylantirildi → yangi
versiya ko'rinsa banner chiqadi → "Yangilash" tugmasi installer'ni
ishga tushiradi.
