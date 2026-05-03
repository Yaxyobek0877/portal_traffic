# NAT, ICE, TURN — nima va nega Portal'da kerak

Bu hujjat Portal foydalanuvchilari uchun. P2P (peer-to-peer)
tarmoqning **eng katta to'sig'i — NAT**. Quyida sodda tilda
tushuntiradi va sizning loglarni qanday o'qish kerakligini ko'rsatadi.

## TL;DR

- **NAT** = tarmoq qurilmangiz (router/operator) sizning lokal
  IP'ingizni tashqi internetga "tarjima qiladi". Bu xavfsizlik
  uchun kerak, lekin P2P uchun muammo.
- **3 xil NAT bor.** Bizning aksariyat foydalanuvchilarimizda
  **Simmetrik NAT** uchraydi — bu eng qiyin tur. Mobil
  internet provayderlari ham odatda shunday.
- **Simmetrik NAT'da to'g'ridan-to'g'ri ulanish ishlamaydi**
  (host yoki srflx kandidatlar bilan). Yagona yo'l — **TURN
  serveri** orqali o'tish.
- **TURN = relay** — har baytni siz va do'stingiz orasida
  o'tkazadigan oraliq server.

---

## NAT nima ekan?

Sizning kompyuteringiz "**lokal IP**" da ishlaydi
(masalan `192.168.1.53` yoki `10.42.128.118`). Ko'p qurilma uchun
bu IP'lar takrorlanadi — har bir uy/ofis xuddi shu seriyalarni
qayta qayta ishlatadi.

Internetda esa har kim "**tashqi IP**" bilan ko'rinadi
(masalan `95.214.211.112`). Routeringiz/mobile carrier tarjima
qiladi: lokal IP+port → tashqi IP+port. Bu jarayon **NAT** (Network
Address Translation).

```
sizning Mac (lokal)              router / mobile NAT          internet
192.168.1.53:51234   --->   95.214.211.112:43497   --->   istalgan server
```

## Uch xil NAT

| Tur | Xulq | Bizning holatda |
| --- | --- | --- |
| **Full Cone NAT** | Tashqi port stabil, har kim ulanadi | P2P **ishlaydi** to'g'ridan-to'g'ri |
| **Restricted NAT** | Stabil port, lekin faqat siz murojaat qilgan IP javob bera oladi | P2P STUN bilan **ishlaydi** |
| **Symmetric NAT** ⭐ | **Har destination uchun yangi port!** | P2P **ishlamaydi** — TURN majburiy |

Sizning loglardagi quyidagi qator — bu **Simmetrik NAT** ning
"barmoq izi":

```
local ice candidate  type=srflx  addr=95.214.211.112  port=43497
local ice candidate  type=srflx  addr=95.214.211.112  port=43544
                                                           ^^^^^
              Ikkala srflx ham bir IP, lekin har xil port = SIMMETRIK
```

Bu degani: do'stingiz `:43497` ga urinsa NAT uni rad qiladi, chunki
`:43497` faqat bir aniq destination uchun ochilgan. Boshqa har destination
uchun yangi port (`:43544`) ochiladi va kim o'zaro hamohanglikni
oldindan bilolmaydi. **Hole punching ishlamaydi.**

Bu O'zbekiston operatorlari uchun odatiy — mobile va aksariyat
home router'larda Simmetrik NAT yoki **CGNAT** ishlatiladi.

## ICE va kandidatlar

WebRTC peer'lar ulanish uchun **ICE (Interactive Connectivity
Establishment)** ishlatadi. Har peer 3 xil "kandidat" yig'adi:

| Kandidat | Manzil | Qachon ishlaydi |
| --- | --- | --- |
| `host` | Lokal IP (192.168.x.x) | Bir LAN ichida |
| `srflx` | Tashqi IP STUN orqali | Cone yoki Restricted NAT |
| `relay` | TURN server orqali | Doim ishlaydi (Cloudflare TURN bepul; o'z `coturn`'ingiz ham mumkin); P2P'dan biroz sekinroq |

**Loglarda ko'rishingiz mumkin:**

```
local ice candidate  type=host    addr=10.42.128.118    port=63154   ← LAN
local ice candidate  type=srflx   addr=95.214.211.112   port=43497   ← STUN
local ice candidate  type=relay   addr=...turn.server...port=...     ← TURN ✓
```

Agar `type=relay` qatori **yo'q** bo'lsa va sizda Simmetrik NAT
bo'lsa — **ulanish hosil bo'lmaydi**. Loglar 30 sekunddan keyin:

```
ice state          state=failed
peer state         state=failed
peer ice failed — diagnostic
  candidate_types=map[host:1 srflx:2]
  had_relay=false
  hint="TURN serveri zarur — siz Simmetrik NAT ortidasiz"
```

## Nima qilish kerak

**Variant A — coturn ni VPS yoki uy serveringizda ishga tushirish (eng yaxshi)**

```bash
sudo apt install coturn
sudo nano /etc/turnserver.conf
# listening-port=3478
# tls-listening-port=5349
# lt-cred-mech
# user=portal:STRONG_RANDOM_PASSWORD
# realm=1pro.uz
# fingerprint
sudo systemctl enable --now coturn
```

Keyin Settings → TURN ga:
```
URL:        turn:turn.1pro.uz:3478
            turn:turn.1pro.uz:3478?transport=tcp
            turns:turn.1pro.uz:5349
Username:   portal
Password:   <yuqoridagi>
```

**Variant B — Cloudflare TURN (~$0.05/GB)**

Cloudflare dashboard → Workers & Pages → Calls → TURN Service →
yoqib qo'ying va creds oling.

**Variant C — Twilio NAT Traversal (free trial)**

https://www.twilio.com/console/voice/calls → Network Traversal Service.
Bepul tarif cheklangan, lekin sinash uchun yetadi.

## Loglar — qanday o'qish

Settings → Loglar → "Oxirgi 200 qatorni ko'rish" bosing.

Asosiy qatorlar:

| Qator | Ma'no |
| --- | --- |
| `peer state state=connecting` | WebRTC handshake boshlandi |
| `ice gathering state=gathering` | Kandidat yig'ilmoqda |
| `local ice candidate type=host` | LAN IP ko'rindi |
| `local ice candidate type=srflx` | STUN orqali tashqi IP topildi |
| `local ice candidate type=relay` | **TURN ulandi! Yaxshi belgi.** |
| `ice gathering state=complete` | Yig'ish tugadi |
| `ice state state=checking` | Konnektivlik tekshiruvi |
| `ice state state=connected` | **Muvaffaqiyat! Mesh ulandi.** |
| `ice state state=failed` | Ulanish bo'lmadi (odatda 30s timeout) |
| `peer ice failed — diagnostic ... had_relay=false` | TURN ishlamayapti / sozlanmagan |

Agar `had_relay=true` bo'lsa-yu, ammo `connected` bo'lmasa —
TURN serveriga kirish bor lekin TURN-relay-rerouting ham yopilgan.
Bu juda kam uchraydigan holat — ISP firewall'i extreme bo'lsa.

## Nega bu shunday murakkab?

WebRTC P2P internetning **eng qiyin masalasi**. Tarmoq texnologiyasi
1990-larda P2P uchun loyihalashtirilmagan — markazlashgan
(client-server) model uchun. NAT esa keyinroq, IPv4 manzil
tanqisligi tufayli paydo bo'lgan. Nathiyada:

- Discord, Zoom, Google Meet kabi markazlashgan ovoz/video xizmatlari
  — TURN'siz **deyarli ishlamaydi**. Shularning hammasida o'z
  TURN cluster'lari bor.
- Tor, BitTorrent, IPFS kabi P2P loyihalar ham ko'pincha "tracker"
  yoki DHT serverlarga muhtoj.
- **Portal** — minimal markazlashtirilgan: faqat handshake uchun
  signaling. Lekin TURN majburiy holat ham uchraydi va biz uni
  yashirib ololmaymiz.

## Hech narsa yordam bermaganda

1. Loglarni nusxa oling (Settings → Loglar → Nusxa)
2. Ikki qurilmangiz qanday tarmoqda ekanini menga ayting:
   - Bir Wi-Fi'da bormi?
   - Mobile internet (4G/5G)?
   - Office VPN ortida?
3. `had_relay=false` bormi yoki `had_relay=true` bo'lib turibmi?

Shunda aniq sabab topiladi.
