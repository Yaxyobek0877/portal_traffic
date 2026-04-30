# Simli protokol

Bu Portal almashinadigan xabarlar uchun rasmiy ma'lumotnoma. Haqiqiy
manba — [`shared/protocol/messages.go`](shared/protocol/messages.go);
bu hujjat — uning odam o'qiy oladigan hamrohi.

---

## Transport

* **Signal qatlami:** JSON matn freymlarini olib boruvchi WebSocket. Har
  bir freym — bitta mustaqil xabar. Endpoint: `wss://<signal-host>/ws`.
* **P2P qatlami:** WebRTC data kanallari, JSON matn freymlari (control,
  chat, xizmat e'lonlari uchun) yoki binary freymlar (fayl bo'laklari va
  tunnel qilingan trafik uchun).

Har bir JSON xabarda yuqori darajadagi `"type"` maydoni bo'ladi.
Dekod qilish uchun avval faqat o'sha maydonni tahlil qiling, keyin tipga
xos struct ga qayta tahlil qiling.

---

## Identifikatorlar

| Maydon | Format | Izoh |
| --- | --- | --- |
| `portal_id` | 6 ASCII raqam | foydalanuvchilarga ko'rsatiladi, `crypto/rand` dan |
| `code`      | 6 ASCII raqam | sirli, ID yonida ko'rsatiladi |
| `peer_id`   | UUID v4 | WebSocket ulanishida server tomonidan ajratiladi |
| `virtual_ip`| `10.42.0.X` | har portalga ajratish, X ∈ [1, 254] |
| `request_id`| ixtiyoriy satr | client tanlaydi, korrelatsiya uchun qaytariladi |

---

## Signal xabarlari — Client → Server

### `portal.create`

Yangi portal ajratish va yuboruvchini egasi sifatida qabul qilish.

```json
{
  "type": "portal.create",
  "nickname": "alice",
  "public_nick": true,
  "capacity": 16
}
```

| Maydon | Tur | Izoh |
| --- | --- | --- |
| `nickname` | string | 2–24 belgi, `[A-Za-z0-9_.-]` |
| `public_nick` | bool | true bo'lsa, boshqalar `portal.join_by_nick` qila oladi |
| `capacity` | int (ixtiyoriy) | 2–254; bo'sh — server standart qiymati (16) |

### `portal.join`

ID va kod orqali mavjud portalga qo'shilish.

```json
{ "type": "portal.join", "portal_id": "428591", "code": "739204", "nickname": "bob" }
```

### `portal.join_by_nick`

Server berilgan public taxallusli foydalanuvchiga join so'rovini
yuborishini so'rash. Maqsad `portal.join_request` ni oladi va
`portal.join_response` bilan javob beradi.

```json
{
  "type": "portal.join_by_nick",
  "target_nickname": "alice",
  "my_nickname": "bob",
  "request_id": "client-tomonidan-yaratilgan-id"
}
```

### `portal.join_response` (faqat egasi)

Kelgan join so'roviga javob.

```json
{ "type": "portal.join_response", "request_id": "...", "accept": true }
```

### `portal.leave`

Joriy portaldan chiqib ketish. Ulanish ochiq qoladi.

```json
{ "type": "portal.leave" }
```

### `portal.kick` (faqat egasi)

```json
{ "type": "portal.kick", "peer_id": "..." }
```

### `portal.lock` (faqat egasi)

Qulflanganda, hatto to'g'ri kod bilan ham yangi peer qo'shila olmaydi.

```json
{ "type": "portal.lock", "locked": true }
```

### `nick.set_visibility`

Taxallus katalogi yozuvini yangilash. Qiymatlar: `"hidden"`,
`"friends_only"`, `"public"`.

```json
{ "type": "nick.set_visibility", "nickname": "alice", "visibility": "public" }
```

---

## Signal xabarlari — Server → Client

### `portal.created`

Muvaffaqiyatli `portal.create` ni tasdiqlaydi.

```json
{
  "type": "portal.created",
  "portal_id": "428591",
  "code": "739204",
  "peer_id": "...",
  "virtual_ip": "10.42.0.1",
  "capacity": 16
}
```

### `portal.joined`

Muvaffaqiyatli qo'shilishni tasdiqlaydi (ID/kod orqali ham, qabul
qilingan nick so'rovi orqali ham) va siz handshake boshlashingiz uchun
mavjud peer ro'yxatini beradi.

```json
{
  "type": "portal.joined",
  "portal_id": "428591",
  "peer_id": "sizning-peer-id",
  "virtual_ip": "10.42.0.3",
  "peers": [
    { "peer_id": "...", "nickname": "alice", "virtual_ip": "10.42.0.1", "is_owner": true },
    { "peer_id": "...", "nickname": "bob",   "virtual_ip": "10.42.0.2", "is_owner": false }
  ]
}
```

### `portal.peer_joined`

Yangi a'zo qo'shilganda mavjud a'zolarga push qilinadi.

```json
{ "type": "portal.peer_joined", "peer_id": "...", "nickname": "charlie", "virtual_ip": "10.42.0.3" }
```

### `portal.peer_left`

Peer chiqib ketgani yoki chetlatilganini qolgan a'zolarga push qiladi.

```json
{ "type": "portal.peer_left", "peer_id": "...", "reason": "left" }
```

`reason`: `"left"`, `"kicked"`, `"disconnected"`.

### `portal.join_request`

Public taxallusi qidirib topilgan portal egasiga yetkaziladi.

```json
{
  "type": "portal.join_request",
  "request_id": "server-tomonidan-yaratilgan",
  "from_peer_id": "...",
  "from_nickname": "bob",
  "from_ip_hash_short": "a3f1c2"
}
```

### `portal.kicked`

Chetlatilgan peer ga yetkaziladi.

```json
{ "type": "portal.kicked", "portal_id": "..." }
```

### `portal.locked`

Qulflash holati o'zgarganda barcha a'zolarga broadcast qilinadi.

```json
{ "type": "portal.locked", "locked": true }
```

### `portal.closed`

Portal yopilganda broadcast qilinadi (eng ko'p — egasi uzilganda).

```json
{ "type": "portal.closed", "portal_id": "...", "reason": "owner_left" }
```

### `error`

Server tomonidan rad etilgan har qanday holat.

```json
{ "type": "error", "code": "PORTAL_CODE_WRONG", "message": "code does not match", "request_id": "..." }
```

| Kod | Ma'no |
| --- | --- |
| `INVALID_MESSAGE` | noto'g'ri JSON yoki noma'lum tur |
| `PORTAL_NOT_FOUND` | bunday ID li portal yo'q |
| `PORTAL_CODE_WRONG` | noto'g'ri kirish kodi |
| `PORTAL_FULL` | sig'im to'lgan |
| `PORTAL_LOCKED` | egasi portalni qulflagan |
| `NICKNAME_INVALID` | noto'g'ri taxallus |
| `NICKNAME_TAKEN` | bu taxallus boshqa ulanishda ishlatilmoqda |
| `NICKNAME_NOT_FOUND` | bunday foydalanuvchi onlayn emas |
| `NICKNAME_NOT_PUBLIC` | foydalanuvchi taklif qabul qilmaydi |
| `NOT_IN_PORTAL` | bu amal portalda bo'lishni talab qiladi |
| `ALREADY_IN_PORTAL` | yaratish/qo'shilishdan oldin chiqib keting |
| `PEER_NOT_FOUND` | maqsad peer sizning portalingizda yo'q |
| `NOT_OWNER` | bu amal faqat egasi uchun |
| `RATE_LIMITED` | per-IP rate limit ga yetildi |
| `INTERNAL` | server xatosi; xavfsiz qayta urinish mumkin |

---

## WebRTC handshake relay (ikki tomonlama)

Server bularni yuboruvchidan `to` ga **shu jumladan** `from` maydonini
yuboruvchining tasdiqlangan peer ID bilan qayta yozgan holda uzatadi
(yomonniyatli peer boshqasini taqlid qila olmaydi).

### `webrtc.offer`

```json
{ "type": "webrtc.offer", "from": "server-shtamplaydi", "to": "...", "sdp": "..." }
```

### `webrtc.answer`

```json
{ "type": "webrtc.answer", "from": "server-shtamplaydi", "to": "...", "sdp": "..." }
```

### `webrtc.ice`

```json
{
  "type": "webrtc.ice",
  "from": "server-shtamplaydi",
  "to": "...",
  "candidate": "candidate:...",
  "sdp_mid": "0",
  "sdp_mline_index": 0
}
```

`candidate` qiymati bo'sh satr bo'lsa — ICE specifikatsiyasiga muvofiq
end-of-candidates signali.

---

## P2P data kanali: `control`

WebRTC ulanishi tugagandan so'ng peer lar shu JSON freymlarni `control`
data kanali orqali almashishadi. Server bunda ishtirok etmaydi.

### `ping` / `pong`

```json
{ "type": "ping", "ts": 1730476800000 }
{ "type": "pong", "ts": 1730476800000, "echo_ts": 1730476800000 }
```

`ts` — yuboruvchining lokal soatidan unix-millisekund timestamp.
`pong.echo_ts` — asl ping ning `ts` ini takrorlaydi, shunda RTT ni
soatlarni sinxronlashtirmasdan hisoblash mumkin. Standart kadensiya 5 s.

### `presence`

```json
{ "type": "presence", "status": "active" }
```

`status` qiymati: `"active"`, `"idle"`, `"away"`.

### `service.expose` / `service.unexpose`

Proxy qatlami uchun lokal fosh qilingan xizmatni e'lon qilish yoki olib tashlash.

```json
{ "type": "service.expose", "name": "Minecraft", "protocol": "tcp", "port": 25565 }
{ "type": "service.unexpose", "port": 25565 }
```

### `service.list_request` / `service.list_response`

```json
{ "type": "service.list_request" }
{ "type": "service.list_response", "services": [ ... ] }
```

---

## Kengaytirish siyosati

* Mavjud xabar turiga yangi ixtiyoriy maydon qo'shish har doim xavfsiz.
  Eski client lar noma'lum maydonlarni e'tiborsiz qoldiradi.
* Yangi yuqori darajadagi xabar turi qo'shish xavfsiz. Eski client lar
  `INVALID_MESSAGE` bilan javob beradi; yuboruvchilar bunga tayyor
  bo'lishi kerak.
* Maydonlarni o'zgartirish yoki olib tashlash buzuvchi o'zgarish.
  Qilmang.
* `portal_id` yoki `code` formatini o'zgartirish (masalan, 8 raqamga
  kengaytirish) major versiya o'sishidir.
