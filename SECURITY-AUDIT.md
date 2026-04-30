# Xavfsizlik va to'g'rilik auditi (2026-04-30)

Bu hujjat 1+2-bosqichlardan keyin to'liq audit natijalarini va `crypt` +
`proxy` paketlari bilan birga keltirilgan kuchaytirishlarni qayd etadi.

## TL;DR

**Loyiha endi siz xohlagan narsani qila oladi:**

- 16 tagacha qurilma o'rtasida to'g'ridan-to'g'ri P2P mesh hosil bo'ladi.
- Minecraft / Valheim / boshqa TCP server larni bir kompyuterda ochasiz va
  do'stlaringiz `localhost:<port>` orqali ulanadi (LAN da bo'lgandek).
- Web ilovalar, SSH, lokal dev URL ulashish — `tcp:` ni qo'llab-quvvatlovchi
  hamma narsa.
- Hammasi DTLS + NaCl secretbox bilan ikki qatlam shifrlanadi.

**Audit davomida 5 ta haqiqiy bug topildi va to'g'rilandi.**

---

## Topilgan va to'g'rilangan kamchiliklar

### 1. Data race: heartbeat outstanding map (jiddiy)

**Muammo.** `mesh.Manager.sendPings` `RLock` ostida `p.outstanding[ts] = now`
yozardi. Go map'ni bir vaqtda ko'p goroutine yozsa panic yoki silent buzilish.

```go
// Eski (xato):
m.mu.RLock()
for _, p := range m.peers {
    p.outstanding[ts] = now  // RLock ostida map yozish — RACE
}
m.mu.RUnlock()
```

**Yechim.** Har bir `Peer` o'z `sync.Mutex` iga ega bo'ldi. Manager mu faqat
peers map a'zoligini himoya qiladi; har peer ning ichki holati (rtt,
outstanding, services) per-peer lock ostida.

Tasdiqlandi: `go test -race ./...` 5 marta yashil.

### 2. Glare: 3+ peer mesh ulana olmaydi (jiddiy)

**Muammo.** Lexicographic tie-break orqali ham mavjud, ham yangi peer
o'zini "offerer" deb hisoblardi → ikkita SDP offer bir-biriga uchirib,
hech qaysi tomon answer qaytarmasdi.

**Yechim.** Glare-free qoida: **joiner har doim offerer**, mavjud peer
har doim answerer. Bitta "yangi" tomon bor — masala yopildi.

### 3. Channel close race teardown da

**Muammo.** Close() goroutine ichida `time.Sleep(100ms)` keyin
`close(c.localICE)` qilardi. Lekin pion DTLS callback lar ko'pincha
shu oraliqda yana `c.localICE <- ...` qilardi → **panic on closed channel**.

**Yechim.** Per-event channel'lar endi yopilmaydi. Tugash signal'i sifatida
`Done() <-chan struct{}` kanali ishlatiladi. Iste'molchilar
`select { case <-Done(): ... }` orqali to'xtaydilar.

### 4. Test ham rejasiz tugatish racega kirardi

**Muammo.** `proxy` test `t.Cleanup` ham aktiv goroutine pump ham
fakeMesh queue ga yozardi → race.

**Yechim.** `done chan struct{}` non-blocking close pattern bilan tartibli teardown.

### 5. App-layer encryption va'da qilingan, lekin amalda yo'q (xavfsizlik)

**Muammo.** `ARCHITECTURE.md` "All chat/file content encrypted with
NaCl secretbox using a key derived from the portal code (PBKDF2)"
deb da'vo qilardi. Kodda buni hech kim qilmasdi.

**Yechim.** Yangi `client/crypt/` paketi:

- `Derive(code) → Key`: PBKDF2-SHA256, 200k iteratsiya, 32-bayt kalit.
- `Seal(key, plain) → ciphertext`: NaCl secretbox, har frame uchun
  yangi 24-baytlik nonce.
- `Open(key, cipher) → plain | err`: MAC tekshirish bilan.
- 7 ta test (round-trip, derterminism, wrong-key reject, tamper detect,
  nonce uniqueness, short-input reject).

Endi chat / proxy frame'lar **DTLS + secretbox** — ikki qatlam.

---

## Yangi imkoniyat: TCP proxy (Phase 5 dan ko'chirilgan)

Bu 5-bosqichning asosiy yutug'i edi va biz uni endi qildik. Yangi paket:
`client/proxy/`.

**Frame format** (proxy data channel ustida):

```
| type (1) | stream_id (4 BE) | payload... |

types:
  0x01 OPEN     "tcp:<port>"        — peer A bizning <port>imizni ochmoqchi
  0x02 OPEN_OK                       — qabul qilindi
  0x03 OPEN_ERR "<msg>"              — rad etildi
  0x04 DATA     <bytes>              — har ikki yo'nalishda
  0x05 CLOSE                         — ulanish tugadi
```

**Foydalanuvchi yo'li:**

```sh
# Alice — Minecraft serverni ochib portalga e'lon qiladi
portal-cli -mode create -nick alice -expose tcp:25565:minecraft

# Bob — qo'shiladi va alice'ning :25565 ni o'z mashinasidagi :25565 ga uladi
portal-cli -mode join -nick bob -portal <ID> -code <KOD> \
    -dial tcp:<alice-peer-id>:25565:25565

# Endi bob Minecraft client'da "localhost:25565" ga ulanadi → mesh orqali alice ga.
```

End-to-end demo: alice'ning `localhost:19000`'dagi Python HTTP serveri
mesh orqali bob'ning `localhost:19500`'iga uzatildi va `curl` to'g'ri
javob qaytardi (RTT ~0.5ms lokal).

**Cheklovlar (v1):**

- Faqat TCP. UDP framing keyingi versiyada.
- Flow control faqat WebRTC data channel buferiga tayanadi.
- Stream ID lar reuse qilinmaydi (har stream — yangi).

---

## Yana topilgan, lekin tuzatilmagan masalalar

| # | Masala | Ta'sir | Holat |
| - | --- | --- | --- |
| 6 | Signaling uzilsa avtomatik qayta ulanish yo'q | Qisqa internet uzilishlari mesh ni yopadi | 4-bosqich rejasida |
| 7 | TURN server bundled emas | Simmetrik NAT li foydalanuvchilar relay sez kelmaydi | TURN'ni alohida deploy qilish kerak (coturn) |
| 8 | 6 xonali kod fazosi 1M | Botnet bilan ~soatlar ichida brute-force qilinishi mumkin | Rate-limit + foydalanuvchini xabardor qilish kerak. v2 da uzunroq kod ixtiyoriy |
| 9 | Failed join lar bir xil rate limit ostida | Ko'p marta noto'g'ri kod yuborish — cheklov hosil qiladi, lekin alohida hisoblanmaydi | v2 da owner ga "N marta noto'g'ri urinish" alert |
| 10 | Origin allowlist sukut bo'yicha ochiq | Brauzerda ishlatilsa CSRF mumkin | Wails ilovada faqat lokal trafik bor — CSRF ahamiyatsiz; lekin doc da yodda tutish |

---

## Tahdid modeli — yangilangan

| Tajovuzkor | Nimani oladi | Nimani ololmaydi |
| --- | --- | --- |
| Tarmoq kuzatuvchisi | TLS-wrapped frame'lar | DTLS dan keyin ham, secretbox dan keyin ham hech narsa |
| Qo'lga olingan signal serveri | Handshake metadata, peer ID, IP, taxallus | Ilova ma'lumotlari (chat / fayl / proxy) sealed |
| Internet skaneri | Ochiq /ws | ID + kodsiz portalga kirolmaydi |
| Brute-forcer (bitta IP) | 6 xonali kodning 30 min/IP urinishini | ~22 kun/IP yarmini sinash uchun |
| Brute-forcer (10k IP botnet) | Tezroq, lekin server tomonida ko'rinadi | Owner kuzatishi mumkin |
| Portal ichidagi yomonniyatli peer | Boshqa peer larga chat/fayl yuborishi | Boshqa portallardagi peer larga yetolmaydi (relay reject); o'z `from` ni soxtalashtirolmaydi |
| Cloudflare proxy | TLS terminate qiladi | Origin Cert bilan Full (strict) rejim — Cloudflare uchun ham app-layer secretbox protected |

---

## Test natijalari

```
go test -race ./... -count=1
ok      portal_traffic_client/crypt      4.054s
ok      portal_traffic_client/peer       2.640s
ok      portal_traffic_client/proxy      1.696s
ok      portal_traffic_client/signaling  2.109s

ok      portal_traffic_server            2.870s
ok      portal_traffic/shared/protocol   0.518s
```

5 marta peshma-pesh ishga tushirildi — barchasi yashil.

End-to-end demo'lar:

1. **2 peer mesh** (signaling.1pro.uz orqali production): RTT ~30-50ms.
2. **3 peer mesh** (lokal): har juftlik 200-700µs.
3. **TCP proxy** (lokal): alice HTTP server :19000 → bob `curl localhost:19500` → muvaffaqiyatli javob.

---

## Foydalanish hollari — bugun ishlaydi

| Use case | Qanday qilamiz | Holat |
| --- | --- | --- |
| LAN o'yin server (Minecraft, Valheim, Terraria, Factorio) | Alice expose tcp:25565, bob dial → Minecraft client localhost:25565 | ✅ |
| Lokal dev server ulashish (web, API) | Alice expose tcp:3000, bob dial → curl localhost:3000 | ✅ |
| Lokal SSH kirish (uy serveriga ofisdan) | SSH server ochiq, expose tcp:22, bob ssh -p <local> | ✅ |
| Fayl uzatish (kichik) | chat channel orqali | ✅ (lekin progress bar yo'q) |
| Fayl uzatish (katta, progress bilan) | transfer channel + chunker | 4-bosqich |
| Ovozli aloqa | WebRTC audio track + push-to-talk | 5-bosqich |
| Brauzer-asosli kirish | yo'q — desktop client kerak | rejada yo'q |
| Mobile app | yo'q | rejada yo'q |

---

## Xulosa

- Kerakli **xavfsizlik teshiklari yopildi** (race + missing crypto layer).
- **LAN-style P2P** ishlaydi — TCP servis ulashish, masofadan boshqarish.
- Asosiy oqim **doc va kodda mos** — va'da qilingan crypto layer endi haqiqatan ham bor.
- Asosiy qolgan ish: **desktop UI** (3-bosqich), keyin sayqal va kuchli imkoniyatlar.
