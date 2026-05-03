# Arxitektura

Bu hujjatda Portal qanday ishlashi tushuntiriladi. Aniq haqiqat — kod;
boshlash uchun [`shared/protocol/messages.go`](shared/protocol/messages.go)
ga qarang.

---

## Asosiy maqsadlar

1. **Asosan to'g'ridan-to'g'ri ulanish.** Ikki peer handshake o'tkazib
   bo'lgach, ularning paketlari biz boshqaradigan server orqali o'tmasligi
   shart. Latentlik, maxfiylik va xarajat — hammasi yutadi.
2. **Signal serveri almashtiriladigan.** U faqat peer larni topishtiradi
   va SDP/ICE ni yo'naltiradi. Istalgan kishi o'zinikini ishga tushira
   oladi; client lar qaysi URL ga sozlansa, o'shanga ulanadi.
3. **Signal serveri ilova trafigini hech qachon o'qiy olmaydi.** U faqat
   handshake uchun JSON konvertlarini ko'radi. WebRTC ning DTLS i va
   portal kodidan olingan app-layer secretbox bizga ikki mustaqil
   himoya qatlami beradi.
4. **A portalidagi peer B portalidagi peer ga zond yubora olmaydi.** Server
   hamma peer larni bitta WebSocket orqali bog'laydigan bo'lsa-da, relay
   yo'li uzatishdan oldin portal a'zoligini aniq tekshiradi.

---

## Yuqori darajadagi diagramma

```
┌──────────────────────────────────────────────┐
│      Signal serveri (Go, xususiy)            │
│  - Portal reyestri (6 xonali ID + kod)       │
│  - Taxalluslar katalogi (ixtiyoriy ochiq)    │
│  - WebRTC SDP/ICE relay (faqat handshake)    │
└──────────────────┬───────────────────────────┘
                   │ TLS WebSocket (faqat handshake)
        ┌──────────┴──────────┐
        │                     │
   ┌────▼────┐  ◄── P2P ───►  ┌────▼────┐
   │ Peer A  │   WebRTC mesh  │ Peer B  │
   └────┬────┘                └────┬────┘
        │   ▲           ▲           │
        │   └─── P2P ───┘           │
        │       ┌─────┐             │
        └──────►│  C  │◄────────────┘
                └─────┘
```

Mesh o'rnatilgandan so'ng, har bir peer juftligi to'g'ridan-to'g'ri
WebRTC ulanishiga ega va ular orqali to'rtta multipleks qilingan data
kanali oqadi:

| Kanal | Ishonchlilik | Nima olib boradi |
| --- | --- | --- |
| `control` | ishonchli, tartibli | heartbeat, presence, xizmat e'lonlari |
| `chat` | ishonchli, tartibli | matn xabarlar (guruh va shaxsiy) |
| `transfer` | ishonchli, tartibli | bo'lakli fayl uzatish |
| `proxy` | ishonchsiz, tartibsiz | tunnel qilingan TCP/UDP trafik |

---

## Portalga qo'shilish: to'liq ketma-ketlik

```
Bob                            Server                            Alice (egasi)
 │                                │                                │
 ├── portal.join(id, code) ──────▶│                                │
 │                                │                                │
 │                                │  id + kod tekshirildi          │
 │                                │  virtual IP ajratildi          │
 │                                │                                │
 │◀── portal.joined(peers) ───────┤                                │
 │                                ├── portal.peer_joined(bob) ────▶│
 │                                │                                │
 │                                │                                │
 │  har bir mavjud peer P uchun:                                   │
 │     ├── webrtc.offer(to=P) ──▶ ├── webrtc.offer(from=bob) ─▶ P  │
 │     │◀── webrtc.answer ◀────── ├──◀── webrtc.answer ────────── P│
 │     │── ICE kandidatlar ─────▶ ├── ICE kandidatlar ──────────▶ P│
 │     │      (ICE tugaguncha)                                     │
 │     │                                                           │
 │     └── to'g'ridan-to'g'ri WebRTC ulandi                        │
 │            ↓                                                    │
 │       data kanallar ochildi (control, chat, transfer, proxy)    │
```

Data kanallar ochilgandan keyin barcha trafik ular orqali
to'g'ridan-to'g'ri oqadi. Signal serverining bu sessiyada keyingi roli
yo'q.

---

## Komponentlar xaritasi

```
shared/
└── protocol/                Simli format — har bir xabar turi va payload struct.
                              Ikkala tomon uchun haqiqat manbai; uni o'zgartirish
                              orqaga moslashuvchan bo'lishi shart (faqat ixtiyoriy
                              maydon qo'shish ruxsat).

server/                       Xususiy. Cloudflare ortidagi VPS da ishlaydi.
├── main.go                  Kirish nuqtasi, TLS listener, /ws + /healthz
├── connection.go            Per-WS read/write pump lar, ping/pong handler,
│                              CF-Connecting-IP / X-Forwarded-For ga rioya qilish
├── hub.go                   Ulanish reyestri, broadcast yordamchi funksiyalari,
│                              kutilayotgan join larni kuzatish, GC ticker
├── portal.go                6 xonali ID generatsiyasi (crypto/rand), a'zolar,
│                              10.42.0.0/24 da virtual IP ajratish
├── nickname.go              Public/private taxalluslar katalogi, IP-hash hint
├── handlers.go              Xabar dispatch + WebRTC relay (server `from` ni
│                              shtamplaydi, peer lar yolg'on so'zlay olmaydi)
├── ratelimit.go             Per-IP token bucket (golang.org/x/time/rate)
└── deploy/                  systemd unit + Cloudflare deploy qo'llanma

client/                       Public. Wails + React desktop dastur.
└── (2-bosqich)
```

---

## Konkurrentlik modeli (server)

Har bir WebSocket uchun bitta read va bitta write goroutine bor. Read
goroutine freym larni dekod qiladi va to'g'ridan-to'g'ri hub method larini
chaqiradi; hub har bir reyestr uchun `sync.RWMutex` va per-portal qulflar
ishlatadi. Markaziy event loop yo'q — handler va freym ni o'qigan goroutine
bir xil, bu trace qilishni soddalashtiradi.

Tashqi freymlar har-bir-ulanish uchun bufferli kanal orqali oqadi
(`send chan []byte`, sukut bo'yicha 64 slot). Agar client juda sekin
bo'lib kanal to'lib qolsa, ulanish uziladi — biz tiqilib qolgan client ni
saqlab tursak, butun hub ni ushlab turgan bo'lardik.

---

## NAT va TURN zaxiraga o'tish

WebRTC ning ICE i kandidatlarni uch manbadan to'playdi:

1. **Host kandidatlar** — lokal IP/port. Bir LAN da ishlaydi.
2. **Server-reflexive kandidatlar** — STUN orqali topilgan ochiq IP/port.
   Aksariyat uy/ofis NAT lari shunda ishlaydi.
3. **Relay kandidatlar** — TURN serveri orqali. Simmetrik NAT va
   aksariyat CGNAT uchun zarur.

Portal sukut bo'yicha bir nechta STUN serverlari bilan keladi (Google,
Cloudflare). TURN ixtiyoriy: Settings → Network bo'limidan **Cloudflare
TURN** (bepul tarif) yoki o'zingizning `coturn` URL'ingizni ulashingiz
mumkin. v0.4.0 dan boshlab desktop dastur har peer ustida ⚡ **P2P**
(host/srflx — to'g'ridan-to'g'ri) yoki ☁️ **TURN** (relay orqali) belgisini
ko'rsatadi, shunda foydalanuvchi trafik qaysi yo'ldan oqayotganini
darhol ko'radi.

NAT turi aniqlash dastur ishga tushganda STUN orqali bo'ladi; natija
tarmoq o'zgarmaguncha sozlamalarda kesh qilinadi.

---

## Shifrlash

Peer-to-peer trafikni ikki mustaqil qatlam himoya qiladi:

1. **WebRTC DTLS-SRTP.** Hamma data kanal trafigi uchun standart. Kalitlar
   SDP handshake davomida muzokara qilinadi. Hatto signal serverini
   qo'lga olgan tajovuzkor ham buni ocha olmaydi, chunki u efemera DTLS
   kalitlarini olmaydi — ular SDP fingerprint almashinuvidan olinadi.
2. **App-layer secretbox.** Hamma chat, fayl va proxy payload lar
   qo'shimcha ravishda PBKDF2 (100k iteratsiya) orqali portal kodidan
   olingan kalit bilan NaCl secretbox ga o'raladi. Bu *portal ichidagi
   yomonniyatli peer ga* qarshi himoya qiladi — u data kanallariga ega
   bo'lsa ham, biror narsani o'qish uchun portal kodi kerak.

Portal kodi ataylab qisqa: u Portal dan tashqari yon kanal orqali (chat,
ovoz, qog'oz) yetkaziladi. O'sha qisqa kod ikkinchi omil sirga aylanadi.

---

## Tahdid modeli

| Tajovuzkor | Nimani oladi | Nimani ololmaydi |
| --- | --- | --- |
| Peer va server o'rtasidagi tarmoq kuzatuvchisi | TLS bilan o'ralgan WebSocket freymlarini ko'radi | TLS dan keyin SDP ni o'qiy olmaydi |
| Qo'lga kiritilgan signal serveri | Handshake, peer ID, IP, taxalluslar | DTLS bilan himoyalangan ilova trafigini |
| Internetdagi tasodifiy skanerchi | Ochiq `/ws` endpoint ni ko'radi | ID + kodsiz portalga kira olmaydi |
| Kodlarni taxmin qiluvchi brute-forcer | 30 join/min/IP rate limit, 1M kod fazosi | Realistik ~10 yil yarmini sinash uchun |
| Portal ichidagi yomonniyatli peer | Chat va fayl qabul qila oladi (u qo'shilgan!) | Boshqa peer larni taqlid qila olmaydi (server `from` ni shtamplaydi); boshqa portallardagi peer larga yeta olmaydi |
| Qurilmasini yo'qotgan foydalanuvchi | Eng yomon holat: kelajakda o'sha foydalanuvchi egasi bo'lgan portallar | DTLS kalitlari efemera; eski sessiyalardagi trafikni qayta tiklab bo'lmaydi |

Himoya *qilinmaydigan* narsalar:

- Portal ichida bo'lgan va kodi bor sodiq tajovuzkor halol peer qila
  oladigan har bir narsani qila oladi. Notanish odamlar bilan kod
  ulashmang.
- Cloudflare ni proksi sifatida ishlatsangiz, u client va origin
  o'rtasidagi TLS bilan o'ralgan trafikni ko'radi. **Full (strict)** rejim
  + Origin Certificate ishlating, shunda Cloudflare↔origin bo'lagi ham
  shifrlanadi.
- PBKDF2 ga qarshi side-channel timing hujumlar qamrovga kirmaydi; portal
  kodlarini brute-force qilmoqchi bo'lgan tajovuzkor baribir foydali
  o'tkazuvchanlikdan ancha pastda rate-limit qilinadi.
