<div align="center">

# Portal

**To'g'ridan-to'g'ri ulanish. Orada hech qanday server yo'q.**

Portal — bu odamlar o'rtasida xususiy peer-to-peer (tengma-teng) mesh tarmoq
quradigan desktop dastur. Portal oching, 6 xonali kodni ulashing — va
ichidagi har bir qurilma boshqa har bir qurilma bilan to'g'ridan-to'g'ri
ulanadi: chat, fayl uzatish, o'yin serveri, dev URL ulashish va boshqa
narsalar uchun. Signal serveri faqat dastlabki qo'l berishish (handshake)
uchun ishlatiladi; undan keyin hamma trafik peer-to-peer oqadi.

</div>

---

## Bu loyiha bilan nima qila olasiz

| | |
| --- | --- |
| **2 dan 16 tagacha qurilmani ulash** | Portal oching, 6 xonali ID + kodni ulashing. Internetning istalgan joyidan kim xohlasa qo'shila oladi — router sozlamasi yo'q, VPN yo'q, port forwarding yo'q. |
| **Chat va fayl uzatish** | Guruh chat, shaxsiy xabarlar, drag-and-drop fayl uzatish va progress bar. Portal kodidan olingan kalit yordamida uchidan-uchiga shifrlanadi. |
| **Istalgan TCP/UDP ni tunnel qilish** | Har bir peer `10.42.0.0/24` diapazonida virtual IP oladi. Minecraft serverni `localhost:25565` da ishga tushirasiz, do'stlaringiz `10.42.0.3:25565` orqali ulanishadi — Portal baytlarni mesh orqali uzatadi. |
| **Mesh ni jonli ko'rish** | Kim kim bilan ulangani jonli vizual ko'rinishda; haqiqiy vaqtda RTT (ping); NAT-traversal indikatorlari (to'g'ridan-to'g'ri yoki TURN orqali). |
| **Maxfiyligingizni saqlash** | WebRTC handshake dan keyin signal serveri sizning trafigingizni hech qachon ko'rmaydi. Hamma ma'lumot peer-to-peer oqadi. |

> **Holat:** signal serveri tayyor va ishlamoqda. Desktop client faol
> ishlab chiqilmoqda — qarang [ROADMAP.md](ROADMAP.md).

---

## O'rnatish

> Desktop dastur uchun tayyor build hali chiqarilmagan. 2-bosqich tugagandan
> keyin tavsiya etiladigan yo'l:
>
> 1. Releases sahifasidan o'z OS uchun installerni yuklab oling.
> 2. Oching, taxallus (nickname) yozing va **Portal yaratish** yoki
>    **Portalga qo'shilish** ni bosing.
>
> CLI yo'q, sozlash yo'q, hisob qaydnomasi yo'q.

Hozircha manbadan build qilishingiz mumkin — qarang [INSTALL.md](INSTALL.md).

### Sizning kompyuteringizda nima bo'lishi kerak

| Platforma | Nima kerak | Nega |
| --- | --- | --- |
| Hammasi | WebRTC ni qo'llab-quvvatlovchi internet ulanishi (deyarli barchasi qo'llab-quvvatlaydi) | Mesh transport |
| macOS 12+ | Xcode Command Line Tools | imzolash, native webview |
| Windows 10+ | WebView2 runtime (Win 11 da o'rnatilgan) | UI |
| Linux | `webkit2gtk-4.1`, `libgtk-3` | UI |

Agar siz qattiq NAT (CGNAT yoki simmetrik) ortida bo'lsangiz, ulanish
avtomatik tarzda TURN relay ga o'tadi — qarang
[ARCHITECTURE.md](ARCHITECTURE.md#nat-va-turn-zaxiraga-otish).

---

## Sessiya qanday ko'rinadi

```
1. Dasturni ochasiz, taxallus yozasiz.
2. "Portal yaratish" ni bosasiz.
3. Portal sizga ko'rsatadi:    ID:    428591
                                KOD:  739204
   (yoki QR kodni skanerlatib oling)
4. Do'stingiz dasturni ochadi, taxallus yozib "Portalga qo'shilish" ni bosadi.
5. ID + kodni kiritadi. Ichkariga kirdi.
6. Jonli mesh diagrammasi: har bir peer to'g'ridan-to'g'ri boshqa har bir peer ga ulangan.
```

---

## Hujjatlar

| Hujjat | Nima haqida |
| --- | --- |
| [ROADMAP.md](ROADMAP.md) | 1–5 bosqichlar va joriy holat |
| [ARCHITECTURE.md](ARCHITECTURE.md) | Mesh topologiyasi, NAT traversal, shifrlash, tahdid modeli |
| [PROTOCOL.md](PROTOCOL.md) | Signal va P2P kanallar uchun simli protokol |
| [INSTALL.md](INSTALL.md) | Har bir platforma uchun manbadan build qilish |
| [CONTRIBUTING.md](CONTRIBUTING.md) | Qanday hissa qo'shish, issue ochish, testlarni ishga tushirish |
| [SECURITY.md](SECURITY.md) | Xavfsizlik siyosati va zaifliklarni xabar qilish |

---

## Repository tuzilishi

```
portal_traffic/
├── shared/        Har bir client tomonidan ishlatiladigan simli protokol (shu yerda)
├── client/        Wails + React desktop dastur (2-bosqich — yo'lda)
└── server/        Signal serveri (xususiy — bizning VPS da ishlaydi, repodan tashqarida)
```

Signal serveri kodi ataylab ushbu repoga kiritilmagan. Portal client
o'zi sozlangan istalgan signal URL ga ulanadi, shuning uchun server
deyarli almashtirib qo'yiladigan komponent. Agar siz o'zingizniki ishga
tushirmoqchi bo'lsangiz, simli protokol to'liq
[PROTOCOL.md](PROTOCOL.md) va
[`shared/protocol/messages.go`](shared/protocol/messages.go) da
hujjatlashtirilgan — mos signal server bir necha yuz qator Go kodi.

---

## Litsenziya

Litsenziya hozircha tanlanmagan (TBD). Tanlangunga qadar bu kod faqat
ma'lumot uchun taqdim etilmoqda; qayta tarqatishdan oldin muallif
bilan bog'laning.
