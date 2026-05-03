# Foydalanish shartlari / Terms of Service

**Oxirgi yangilanish / Last updated:** 2026-05-01
**Tegishli versiya / Applies to:** Portal v0.4.0 va keyin / and later

---

## 1. Qabul qilish / Acceptance

Portal'ni o'rnatish yoki ishlatish orqali siz ushbu shartlarga
roziligingizni bildirasiz. Rozi bo'lmasangiz, dasturni ishlatmang.

By installing or using Portal, you agree to these terms. If you don't
agree, don't use it.

---

## 2. Litsenziya / License

Portal kodi [MIT litsenziyasi](LICENSE) ostida. Siz uni o'zgartirishingiz,
qayta tarqatishingiz, kommersiya maqsadida ishlatishingiz mumkin.
Faqat asl litsenziya matnini saqlang.

The Portal source code is under the [MIT License](LICENSE). You may
modify, redistribute, and use commercially. Just keep the original
license text.

"Portal" nomi va brendi (logo, ikonka, `portal.1pro.uz` domeni) — alohida.
Fork qilsangiz boshqa nom tanlang.

The "Portal" name and brand (logo, icon, `portal.1pro.uz` domain) are
separate. Use a different name if you fork.

---

## 3. Kafolat yo'q / No warranty

Portal "BORICHA" (AS IS) taqdim etiladi. Biz quyidagilarga kafolat
bermaymiz:

Portal is provided "AS IS". We make no guarantees about:

- Uzluksiz mavjudlik (signal serveri o'chib qolishi mumkin) / Continuous availability (signaling may go down)
- TURN serverining tezligi yoki o'tkazuvchanligi / TURN server speed or capacity
- Sizning ma'lumotlaringizning yo'qolmasligi / Data being preserved
- Hech qanday belgilangan maqsadga moslik / Fitness for any particular purpose

Sizning yo'qotgan ma'lumotingiz, vaqti yoki imkoniyatingiz uchun biz
javobgar emasmiz.

We are not liable for lost data, time, or opportunity.

---

## 4. Taqiqlangan foydalanish / Prohibited use

Portal'ni quyidagilar uchun ishlatish **taqiqlanadi** va siznig
xizmatlarga kirishingiz to'xtatilishi mumkin:

The following is **prohibited** and may result in service termination:

- **Noqonuniy faoliyat** — qonunlarni buzadigan har qanday narsa.
  **Illegal activity** — anything violating applicable laws.
- **Spam yoki mass-targeting** — boshqa foydalanuvchilarga so'ralmagan
  ulanish urinishlari.
  **Spam or mass-targeting** — unsolicited connection attempts.
- **DDoS yoki tarmoq hujumlari** — Portal'ni hujum vositasi sifatida
  ishlatish.
  **DDoS or network attacks** — using Portal as an attack tool.
- **Boshqalarning kontentini ruxsatsiz tarqatish** — mualliflik huquqi
  buzilishi.
  **Unauthorized content distribution** — copyright violations.
- **Yosh bolalarga zararli kontent** — har qanday shaklda. Bunday
  kontent server tomonida darhol bloklanadi va tegishli organlarga
  xabar beriladi.
  **Child sexual abuse material (CSAM)** — any form. Such content is
  immediately blocked at the server level and reported to authorities.
- **TURN xizmatini suiiste'mol qilish** — bandwidth oqishi yoki
  proxychain.
  **TURN abuse** — bandwidth siphoning or proxychain.

---

## 5. Servisning mavjudligi / Service availability

`signaling.1pro.uz` standart signal serveri xizmati **best-effort**
asosida taqdim etiladi. SLA yo'q, kafolatlangan uptime yo'q. Bizning
serverimiz o'chib qolgan taqdirda ham, siz o'zingizniki ishga tushira
olasiz: [PROTOCOL.md](PROTOCOL.md) batafsil qilingan.

The default `signaling.1pro.uz` signaling service is provided on a
**best-effort** basis. No SLA, no guaranteed uptime. Even if our server
goes down, you can run your own — [PROTOCOL.md](PROTOCOL.md) documents
the wire protocol.

---

## 6. Foydalanuvchining javobgarligi / User responsibility

Portal — vosita. Siz uni qanday ishlatishga, kim bilan ulashishga, qanday
trafik yuborishga **siz javobgarsiz**. Portal kodini ulashish — uy
kalitini berishga o'xshaydi: kodga ega bo'lgan har kim sizning portalingizda
bo'lishi mumkin.

Portal is a tool. **You are responsible** for how you use it, who you
share with, and what traffic you send. Sharing your portal code is like
giving out a house key: anyone with it can join.

---

## 7. PCP-1 (Portal Cipher Protocol) shartlari / PCP-1 terms

Portal Cipher Protocol (PCP-1) standart kriptografik primitivlardan
(Ed25519, X25519, ChaCha20-Poly1305, HKDF-SHA256) qurilgan hibrid
protokoldir. Spetsifikatsiya MIT ostida ochiq:
[`client/crypt/pcp/SPEC.md`](client/crypt/pcp/SPEC.md).

The Portal Cipher Protocol (PCP-1) is a hybrid protocol built on
standard cryptographic primitives (Ed25519, X25519, ChaCha20-Poly1305,
HKDF-SHA256). The specification is open under MIT.

Hech qachon kafolat bermaymiz: PCP-1 hozirgi kunda mavjud bo'lgan barcha
hujumlardan himoya qiladi deyishlik mumkin emas. Bu sohada hech kim
absolute kafolat bera olmaydi. PCP-1 modern best practices'ga muvofiq
loyihalashtirilgan, lekin xavfsizlik audit uchun ochiq.

We never guarantee PCP-1 protects against all current attacks. No one in
this field can give an absolute guarantee. PCP-1 is designed against
modern best practices, but is open for security audit.

---

## 8. O'zgarishlar / Changes

Ushbu shartlar o'zgartirilishi mumkin. Ahamiyatli o'zgarishlar haqida
ilova ichidan xabar beriladi (yangilash paytida) va CHANGELOG'da
qayd etiladi.

These terms may change. Material changes will be announced inside the
app (during update) and recorded in CHANGELOG.

---

## 9. Bog'lanish / Contact

- GitHub: https://github.com/Yaxyobek0877/portal_traffic/issues
- Email: portal@1pro.uz
