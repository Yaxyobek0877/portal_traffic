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

### 4. Multi-device login (bir akkaunt — bir nechta qurilma)

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
